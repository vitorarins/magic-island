package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"golang.org/x/oauth2"
)

type fakeRequester struct{}

func (f *fakeRequester) RequestFeenstra(action string) string {
	log.Printf("RequestFeenstra was called with action: %v", action)
	return "big xml"
}

func (f *fakeRequester) RequestMaker(detector, status string) string {
	log.Printf("RequestMaker was called with detect '%v' and status '%v'", detector, status)
	return "maker response"
}

var (
	globalCode  string
	globalToken oauth2.Token

	testOauthClientId     = "222222"
	testOauthClientSecret = "222222"
	testDomain            = "https://magic.com"
	testRedirectUrl       = "https://redirect.com/test"

	requester = &fakeRequester{}
	handler   = NewHandler(testOauthClientId, testOauthClientSecret, testDomain, []string{testRedirectUrl}, requester)
)

func TestAuthorizeHandler(t *testing.T) {
	clientConfig := oauth2.Config{
		Scopes: []string{"all"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "/authorize",
			TokenURL: "/token",
		},
	}

	tests := []struct {
		clientId     string
		clientSecret string
		redirectUrl  string
		status       int
		body         string
	}{
		{
			clientId:     "",
			clientSecret: "",
			redirectUrl:  "",
			status:       http.StatusBadRequest,
			body:         "invalid_request\n",
		},
		{
			clientId:     "0000000000",
			clientSecret: testOauthClientSecret,
			redirectUrl:  testRedirectUrl,
			status:       http.StatusFound,
			body:         "",
		},
		{
			clientId:     testOauthClientId,
			clientSecret: "0000000000000",
			redirectUrl:  testRedirectUrl,
			status:       http.StatusFound,
			body:         "",
		},
		{
			clientId:     testOauthClientId,
			clientSecret: testOauthClientSecret,
			redirectUrl:  "http://wrong",
			status:       http.StatusFound,
			body:         "",
		},
	}

	for _, test := range tests {
		clientConfig.ClientID = test.clientId
		clientConfig.ClientSecret = test.clientSecret
		clientConfig.RedirectURL = test.redirectUrl
		u := clientConfig.AuthCodeURL("xyz")
		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		server := http.HandlerFunc(handler.AuthorizeHandler)
		server.ServeHTTP(rr, req)

		resp := rr.Result()
		if status := resp.StatusCode; status != test.status {
			t.Errorf("unexpected status: got (%v) want (%v)", status, test.status)
		}

		if rr.Body.String() != test.body {
			t.Errorf("unexpected body: got (%v) want (%v)", rr.Body.String(), test.body)
		}
	}

	clientConfig.ClientID = testOauthClientId
	clientConfig.ClientSecret = testOauthClientId
	clientConfig.RedirectURL = testRedirectUrl
	u := clientConfig.AuthCodeURL("xyz")
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	server := http.HandlerFunc(handler.AuthorizeHandler)
	server.ServeHTTP(rr, req)

	resp := rr.Result()
	if status := resp.StatusCode; status != http.StatusFound {
		t.Errorf("unexpected status: got (%v) want (%v)", status, http.StatusFound)
	}

	if rr.Body.String() != "" {
		t.Errorf("unexpected body: got (%v) want (%v)", rr.Body.String(), "")
	}

	location := resp.Header.Get("Location")
	redirectUrl, err := url.Parse(location)
	if err != nil {
		t.Errorf("Error parsing location URL: %v", err)
	}

	q := redirectUrl.Query()
	globalCode = q.Get("code")
}

func TestTokenHandler(t *testing.T) {
	tests := []struct {
		clientId     string
		clientSecret string
		redirectUrl  string
		code         string
		status       int
		body         string
	}{
		{
			clientId:     "",
			clientSecret: "",
			redirectUrl:  "",
			code:         "",
			status:       http.StatusUnauthorized,
			body:         `{"error":"invalid_client","error_description":"Client authentication failed"}` + "\n",
		},
		{
			clientId:     "0000000000",
			clientSecret: testOauthClientSecret,
			redirectUrl:  testRedirectUrl,
			code:         globalCode,
			status:       http.StatusInternalServerError,
			body:         `{"error":"server_error","error_description":"The authorization server encountered an unexpected condition that prevented it from fulfilling the request"}` + "\n",
		},
		{
			clientId:     testOauthClientId,
			clientSecret: "0000000000000",
			redirectUrl:  testRedirectUrl,
			code:         globalCode,
			status:       http.StatusUnauthorized,
			body:         `{"error":"invalid_client","error_description":"Client authentication failed"}` + "\n",
		},
		{
			clientId:     testOauthClientId,
			clientSecret: testOauthClientSecret,
			redirectUrl:  "http://wrong.com",
			code:         globalCode,
			status:       http.StatusInternalServerError,
			body:         `{"error":"server_error","error_description":"The authorization server encountered an unexpected condition that prevented it from fulfilling the request"}` + "\n",
		},
		{
			clientId:     testOauthClientId,
			clientSecret: testOauthClientSecret,
			redirectUrl:  testRedirectUrl,
			code:         "randomCode",
			status:       http.StatusUnauthorized,
			body:         `{"error":"invalid_grant","error_description":"The provided authorization grant (e.g., authorization code, resource owner credentials) or refresh token is invalid, expired, revoked, does not match the redirection URI used in the authorization request, or was issued to another client"}` + "\n",
		},
	}

	for _, test := range tests {
		req, err := http.NewRequest("GET", "/token", nil)
		if err != nil {
			t.Fatal(err)
		}

		q := req.URL.Query()
		q.Add("grant_type", "authorization_code")
		q.Add("client_id", test.clientId)
		q.Add("client_secret", test.clientSecret)
		q.Add("redirect_uri", test.redirectUrl)
		q.Add("code", test.code)
		req.URL.RawQuery = q.Encode()

		rr := httptest.NewRecorder()
		server := http.HandlerFunc(handler.TokenHandler)
		server.ServeHTTP(rr, req)

		resp := rr.Result()
		if status := resp.StatusCode; status != test.status {
			t.Errorf("unexpected status: got (%v) want (%v)", status, test.status)
		}

		if rr.Body.String() != test.body {
			t.Errorf("unexpected body: got (%v) want (%v)", rr.Body.String(), test.body)
		}
	}

	req, err := http.NewRequest("GET", "/token", nil)
	if err != nil {
		t.Fatal(err)
	}

	q := req.URL.Query()
	q.Add("grant_type", "authorization_code")
	q.Add("client_id", testOauthClientId)
	q.Add("client_secret", testOauthClientSecret)
	q.Add("redirect_uri", testRedirectUrl)
	q.Add("code", globalCode)
	req.URL.RawQuery = q.Encode()

	rr := httptest.NewRecorder()
	server := http.HandlerFunc(handler.TokenHandler)
	server.ServeHTTP(rr, req)

	resp := rr.Result()
	if status := resp.StatusCode; status != http.StatusOK {
		t.Errorf("unexpected status: got (%v) want (%v)", status, http.StatusOK)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Error reading response from token handler: %v", err)
	}

	err = json.Unmarshal(body, &globalToken)
	if err != nil {
		t.Errorf("Error decoding response from token handler: %v", err)
	}

	if globalToken.AccessToken == "" {
		t.Errorf("Access token came empty.")
	}
}

func TestIndexHandler(t *testing.T) {
	tests := []struct {
		route  string
		status int
		body   string
	}{
		{
			route:  "/",
			status: http.StatusOK,
			body:   "Matrix",
		},
		{
			route:  "/404",
			status: http.StatusNotFound,
			body:   "404 page not found\n",
		},
	}

	for _, test := range tests {
		req, err := http.NewRequest("GET", test.route, nil)
		if err != nil {
			t.Fatal(err)
		}

		q := req.URL.Query()
		q.Add("access_token", globalToken.AccessToken)
		req.URL.RawQuery = q.Encode()

		rr := httptest.NewRecorder()
		server := http.HandlerFunc(handler.IndexHandler)
		server.ServeHTTP(rr, req)

		if status := rr.Code; status != test.status {
			t.Errorf("unexpected status: got (%v) want (%v)", status, test.status)
		}

		if rr.Body.String() != test.body {
			t.Errorf("unexpected body: got (%v) want (%v)", rr.Body.String(), test.body)
		}
	}
}

func TestAlarmHandler(t *testing.T) {
	tests := []struct {
		route  string
		status int
		body   string
	}{
		{
			route:  "/ifttt/v1/actions/fullarm",
			status: http.StatusOK,
			body:   "Successfuly executed action arm",
		},
		{
			route:  "/ifttt/v1/actions/partarm",
			status: http.StatusOK,
			body:   "Successfuly executed action partarm",
		},
		{
			route:  "/alarm/arm",
			status: http.StatusOK,
			body:   "Successfuly executed action arm",
		},
		{
			route:  "/alarm/404",
			status: http.StatusNotFound,
			body:   "404 page not found\n",
		},
	}

	for _, test := range tests {
		req, err := http.NewRequest("GET", test.route, nil)
		if err != nil {
			t.Fatal(err)
		}

		q := req.URL.Query()
		q.Add("access_token", globalToken.AccessToken)
		req.URL.RawQuery = q.Encode()

		rr := httptest.NewRecorder()
		server := http.HandlerFunc(handler.AlarmHandler)
		server.ServeHTTP(rr, req)

		if status := rr.Code; status != test.status {
			t.Errorf("unexpected status: got (%v) want (%v)", status, test.status)
		}

		if rr.Body.String() != test.body {
			t.Errorf("unexpected body: got (%v) want (%v)", rr.Body.String(), test.body)
		}
	}
}

func TestStatusHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/status", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	server := http.HandlerFunc(handler.StatusHandler)
	server.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("unexpected status: got (%v) want (%v)", status, http.StatusOK)
	}

	if rr.Body.String() != "OK" {
		t.Errorf("unexpected body: got (%v) want (%v)", rr.Body.String(), "OK")
	}
}

func TestIFTTTHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/ifttt/v1/user/info", nil)
	if err != nil {
		t.Fatal(err)
	}

	q := req.URL.Query()
	q.Add("access_token", globalToken.AccessToken)
	req.URL.RawQuery = q.Encode()

	rr := httptest.NewRecorder()
	server := http.HandlerFunc(handler.IFTTTHandler)
	server.ServeHTTP(rr, req)

	resp := rr.Result()
	if status := resp.StatusCode; status != http.StatusOK {
		t.Errorf("unexpected status: got (%v) want (%v)", status, http.StatusOK)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Error reading response from ifttt handler: %v", err)
	}

	var userInfo map[string](map[string]string)
	err = json.Unmarshal(body, &userInfo)
	if err != nil {
		t.Errorf("Error unmarshaling user info: %v", err)
	}

	userData, ok := userInfo["data"]
	if !ok {
		t.Errorf("Could not find data inside user info.")
	}

	userName, ok := userData["name"]
	if !ok || userName != "Only user" {
		t.Errorf("unexpected user data name: got (%v) want (%v)", userName, "Only user")
	}

	userId, ok := userData["id"]
	if !ok || userId != "onlyuserwehave" {
		t.Errorf("unexpected user data id: got (%v) want (%v)", userId, "onlyuserwehave")
	}
}
