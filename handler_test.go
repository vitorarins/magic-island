package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"cloud.google.com/go/firestore"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"google.golang.org/api/iterator"
)

type fakeRequester struct{}

func (f *fakeRequester) RequestFeenstra(action string) string {
	log.Printf("RequestFeenstra was called with action: %v", action)
	return "big xml"
}

func (f *fakeRequester) RequestMakerDetector(detector, status string) string {
	log.Printf("RequestMakerDetector was called with detect '%v' and status '%v'", detector, status)
	return "maker response"
}

func (f *fakeRequester) RequestMaker(event string) string {
	log.Printf("RequestMaker was called with event '%v'", event)
	return "maker response"
}

var (
	globalCode      string
	globalToken     oauth2.Token
	globalCookieJar []*http.Cookie

	testOauthClientId     = "222222"
	testOauthClientSecret = "222222"
	testDomain            = "https://magic.com"
	testRedirectUrl       = "https://redirect.com/test"

	requester            = &fakeRequester{}
	ctx                  = context.Background()
	firestoreClient, err = firestore.NewClient(ctx, "test")
	handler              = NewHandler(testOauthClientId, testOauthClientSecret, testDomain, []string{testRedirectUrl}, requester, firestoreClient)
)

func TestLoginHandler(t *testing.T) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("test"), 10)
	if err != nil {
		t.Fatal(err)
	}

	user := map[string]string{
		"username": "vitorarins",
		"password": string(hashedPassword),
	}
	firestoreClient.Collection("users").Doc("vitorarins").Set(ctx, user, firestore.MergeAll)

	req, err := http.NewRequest("GET", "/login", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	server := http.HandlerFunc(handler.LoginHandler)
	server.ServeHTTP(rr, req)

	resp := rr.Result()
	if status := resp.StatusCode; status != http.StatusOK {
		t.Errorf("unexpected status: got (%v) want (%v)", status, http.StatusOK)
	}

	responseHtml := rr.Body.String()
	expectedHtml := "<title>Login</title>"
	if !strings.Contains(responseHtml, expectedHtml) {
		t.Errorf("unexpected body: %v should contain (%v)", responseHtml, expectedHtml)
	}

	req, err = http.NewRequest("POST", "/login", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	server = http.HandlerFunc(handler.LoginHandler)
	server.ServeHTTP(rr, req)

	resp = rr.Result()
	if status := resp.StatusCode; status != http.StatusInternalServerError {
		t.Errorf("unexpected status: got (%v) want (%v)", status, http.StatusInternalServerError)
	}

	if rr.Body.String() != "missing form body\n" {
		t.Errorf("unexpected body: got (%v) want (%v)", rr.Body.String(), "missing form body")
	}

	formData := url.Values{
		"username": {"vitorarins"},
		"password": {"test"},
	}
	req, err = http.NewRequest("POST", "/login", strings.NewReader(formData.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()
	server = http.HandlerFunc(handler.LoginHandler)
	server.ServeHTTP(rr, req)

	resp = rr.Result()
	if status := resp.StatusCode; status != http.StatusFound {
		t.Errorf("unexpected status: got (%v) want (%v)", status, http.StatusFound)
	}

	if rr.Body.String() != "" {
		t.Errorf("unexpected body: got (%v) want (%v)", rr.Body.String(), "")
	}

	if resp.Header.Get("Location") != "/auth" {
		t.Errorf("unexpected redirect url: got (%v) want (%v)", resp.Header.Get("Location"), "/auth")
	}

	if len(resp.Cookies()) <= 0 {
		t.Errorf("No cookies!")
	}
	globalCookieJar = resp.Cookies()
}

func TestAuthHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/auth", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	server := http.HandlerFunc(handler.AuthHandler)
	server.ServeHTTP(rr, req)

	resp := rr.Result()
	if status := resp.StatusCode; status != http.StatusFound {
		t.Errorf("unexpected status: got (%v) want (%v)", status, http.StatusFound)
	}

	if resp.Header.Get("Location") != "/login" {
		t.Errorf("unexpected redirect url: got (%v) want (%v)", resp.Header.Get("Location"), "/login")
	}

	req, err = http.NewRequest("GET", "/auth", nil)
	if err != nil {
		t.Fatal(err)
	}
	restoreCookies(req)

	rr = httptest.NewRecorder()
	server = http.HandlerFunc(handler.AuthHandler)
	server.ServeHTTP(rr, req)

	resp = rr.Result()
	if status := resp.StatusCode; status != http.StatusOK {
		t.Errorf("unexpected status: got (%v) want (%v)", status, http.StatusOK)
	}

	responseHtml := rr.Body.String()
	expectedHtml := "<title>Auth</title>"
	if !strings.Contains(responseHtml, expectedHtml) {
		t.Errorf("unexpected body: %v should contain (%v)", responseHtml, expectedHtml)
	}
}

func TestAuthorizeHandler(t *testing.T) {
	clientConfig := oauth2.Config{
		Scopes: []string{"all"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "/authorize",
			TokenURL: "/token",
		},
	}

	tests := []struct {
		caseNumber   int
		clientId     string
		clientSecret string
		redirectUrl  string
		status       int
		body         string
	}{
		{
			caseNumber:   1,
			clientId:     "",
			clientSecret: "",
			redirectUrl:  "",
			status:       http.StatusBadRequest,
			body:         "invalid_request\n",
		},
		{
			caseNumber:   2,
			clientId:     "0000000000",
			clientSecret: testOauthClientSecret,
			redirectUrl:  testRedirectUrl,
			status:       http.StatusFound,
			body:         "",
		},
		{
			caseNumber:   3,
			clientId:     testOauthClientId,
			clientSecret: "0000000000000",
			redirectUrl:  testRedirectUrl,
			status:       http.StatusFound,
			body:         "",
		},
		{
			caseNumber:   4,
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
			t.Errorf("unexpected status on test case '%v': got (%v) want (%v)", test.caseNumber, status, test.status)
		}

		if rr.Body.String() != test.body {
			t.Errorf("unexpected body on test case '%v': got (%v) want (%v)", test.caseNumber, rr.Body.String(), test.body)
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
	restoreCookies(req)

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
		caseNumber   int
		clientId     string
		userId       string
		clientSecret string
		redirectUrl  string
		code         string
		status       int
		body         string
	}{
		{
			caseNumber:   1,
			clientId:     "",
			userId:       "vitorarins",
			clientSecret: "",
			redirectUrl:  "",
			code:         "",
			status:       http.StatusUnauthorized,
			body:         `{"error":"invalid_client","error_description":"Client authentication failed"}` + "\n",
		},
		{
			caseNumber:   2,
			clientId:     "0000000000",
			userId:       "vitorarins",
			clientSecret: testOauthClientSecret,
			redirectUrl:  testRedirectUrl,
			code:         globalCode,
			status:       http.StatusInternalServerError,
			body:         `{"error":"server_error","error_description":"The authorization server encountered an unexpected condition that prevented it from fulfilling the request"}` + "\n",
		},
		{
			caseNumber:   3,
			clientId:     testOauthClientId,
			userId:       "vitorarins",
			clientSecret: "0000000000000",
			redirectUrl:  testRedirectUrl,
			code:         globalCode,
			status:       http.StatusUnauthorized,
			body:         `{"error":"invalid_client","error_description":"Client authentication failed"}` + "\n",
		},
		{
			caseNumber:   4,
			clientId:     testOauthClientId,
			userId:       "vitorarins",
			clientSecret: testOauthClientSecret,
			redirectUrl:  "http://wrong.com",
			code:         globalCode,
			status:       http.StatusInternalServerError,
			body:         `{"error":"server_error","error_description":"The authorization server encountered an unexpected condition that prevented it from fulfilling the request"}` + "\n",
		},
		{
			caseNumber:   5,
			clientId:     testOauthClientId,
			userId:       "vitorarins",
			clientSecret: testOauthClientSecret,
			redirectUrl:  testRedirectUrl,
			code:         "randomCode",
			status:       http.StatusInternalServerError,
			body:         `{"error":"server_error","error_description":"The authorization server encountered an unexpected condition that prevented it from fulfilling the request"}` + "\n",
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
		q.Add("user_id", test.userId)
		q.Add("client_secret", test.clientSecret)
		q.Add("redirect_uri", test.redirectUrl)
		q.Add("code", test.code)
		req.URL.RawQuery = q.Encode()

		rr := httptest.NewRecorder()
		server := http.HandlerFunc(handler.TokenHandler)
		server.ServeHTTP(rr, req)

		resp := rr.Result()
		if status := resp.StatusCode; status != test.status {
			t.Errorf("unexpected status on test case '%v': got (%v) want (%v)", test.caseNumber, status, test.status)
		}

		if rr.Body.String() != test.body {
			t.Errorf("unexpected body on test case '%v': got (%v) want (%v)", test.caseNumber, rr.Body.String(), test.body)
		}
	}

	req, err := http.NewRequest("GET", "/token", nil)
	if err != nil {
		t.Fatal(err)
	}

	q := req.URL.Query()
	q.Add("grant_type", "authorization_code")
	q.Add("client_id", testOauthClientId)
	q.Add("user_id", "vitorarins")
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
		caseNumber int
		route      string
		status     int
		body       string
		token      string
	}{
		{
			caseNumber: 1,
			route:      "/",
			status:     http.StatusOK,
			body:       "Matrix",
			token:      globalToken.AccessToken,
		},
		{
			caseNumber: 2,
			route:      "/404",
			status:     http.StatusNotFound,
			body:       "404 page not found\n",
			token:      globalToken.AccessToken,
		},
		{
			caseNumber: 3,
			route:      "/",
			status:     http.StatusUnauthorized,
			body:       "no more items in iterator\n",
			token:      "unauthorized",
		},
		{
			caseNumber: 4,
			route:      "/",
			status:     http.StatusUnauthorized,
			body:       "invalid access token\n",
			token:      "",
		},
		{
			caseNumber: 5,
			route:      "/unauthorized",
			status:     http.StatusUnauthorized,
			body:       "invalid access token\n",
			token:      "",
		},
	}

	for _, test := range tests {
		req, err := http.NewRequest("GET", test.route, nil)
		if err != nil {
			t.Fatal(err)
		}

		if test.token != "" {
			q := req.URL.Query()
			q.Add("access_token", test.token)
			req.URL.RawQuery = q.Encode()
		}

		rr := httptest.NewRecorder()
		server := http.HandlerFunc(handler.IndexHandler)
		server.ServeHTTP(rr, req)

		if status := rr.Code; status != test.status {
			t.Errorf("unexpected status on test case '%v': got (%v) want (%v)", test.caseNumber, status, test.status)
		}

		if rr.Body.String() != test.body {
			t.Errorf("unexpected body on test case '%v': got (%v) want (%v)", test.caseNumber, rr.Body.String(), test.body)
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
	if !ok || userName != "vitorarins" {
		t.Errorf("unexpected user data name: got (%v) want (%v)", userName, "vitorarins")
	}

	userId, ok := userData["id"]
	if !ok || userId != "vitorarins" {
		t.Errorf("unexpected user data id: got (%v) want (%v)", userId, "vitorarins")
	}
}

func TestNotHomeHandler(t *testing.T) {
	tests := []struct {
		caseNumber int
		route      string
		status     int
		body       string
		users      []map[string]interface{}
	}{
		{
			caseNumber: 1,
			route:      "/ifttt/v1/actions/nothome",
			status:     http.StatusInternalServerError,
			body:       `rpc error: code = NotFound desc = "projects/test/databases/(default)/documents/users/vitorarins" not found` + "\n",
			users:      []map[string]interface{}{},
		},
		{
			caseNumber: 2,
			route:      "/ifttt/v1/actions/nothome",
			status:     http.StatusOK,
			body:       "Successfuly executed action arm",
			users: []map[string]interface{}{
				map[string]interface{}{
					"username": "vitorarins",
				},
			},
		},
		{
			caseNumber: 3,
			route:      "/ifttt/v1/actions/nothome",
			status:     http.StatusOK,
			body:       "Successfuly executed action arm",
			users: []map[string]interface{}{
				map[string]interface{}{
					"username": "vitorarins",
					"home":     false,
				},
			},
		},
		{
			caseNumber: 4,
			route:      "/ifttt/v1/actions/nothome",
			status:     http.StatusOK,
			body:       "Successfuly executed action arm",
			users: []map[string]interface{}{
				map[string]interface{}{
					"username": "vitorarins",
					"home":     true,
				},
			},
		},
		{
			caseNumber: 5,
			route:      "/ifttt/v1/actions/nothome",
			status:     http.StatusOK,
			body:       "Successfuly marked user as not home",
			users: []map[string]interface{}{
				map[string]interface{}{
					"username": "vitorarins",
					"home":     true,
				},
				map[string]interface{}{
					"username": "testuser",
					"home":     true,
				},
			},
		},
		{
			caseNumber: 6,
			route:      "/ifttt/v1/actions/nothome",
			status:     http.StatusOK,
			body:       "Successfuly executed action arm",
			users: []map[string]interface{}{
				map[string]interface{}{
					"username": "vitorarins",
					"home":     true,
				},
				map[string]interface{}{
					"username": "testuser",
					"home":     false,
				},
			},
		},
		{
			caseNumber: 7,
			route:      "/ifttt/v1/actions/nothome",
			status:     http.StatusOK,
			body:       "Successfuly marked user as not home",
			users: []map[string]interface{}{
				map[string]interface{}{
					"username": "vitorarins",
					"home":     true,
				},
				map[string]interface{}{
					"username": "testuser",
				},
			},
		},
	}

	for _, test := range tests {
		deleteCollection(ctx, firestoreClient, firestoreClient.Collection("users"), 10)
		for _, user := range test.users {
			firestoreClient.Collection("users").Doc(user["username"].(string)).Set(ctx, user, firestore.MergeAll)
		}

		req, err := http.NewRequest("GET", test.route, nil)
		if err != nil {
			t.Fatal(err)
		}

		q := req.URL.Query()
		q.Add("access_token", globalToken.AccessToken)
		req.URL.RawQuery = q.Encode()

		rr := httptest.NewRecorder()
		server := http.HandlerFunc(handler.NotHomeHandler)
		server.ServeHTTP(rr, req)

		if status := rr.Code; status != test.status {
			t.Errorf("unexpected status on test case '%v': got (%v) want (%v)", test.caseNumber, status, test.status)
		}

		if rr.Body.String() != test.body {
			t.Errorf("unexpected body on test case '%v': got (%v) want (%v)", test.caseNumber, rr.Body.String(), test.body)
		}
	}
}

func TestHomeHandler(t *testing.T) {
	tests := []struct {
		caseNumber int
		route      string
		status     int
		body       string
		users      []map[string]interface{}
	}{
		{
			caseNumber: 1,
			route:      "/ifttt/v1/actions/home",
			status:     http.StatusInternalServerError,
			body:       `rpc error: code = NotFound desc = "projects/test/databases/(default)/documents/users/vitorarins" not found` + "\n",
			users:      []map[string]interface{}{},
		},
		{
			caseNumber: 2,
			route:      "/ifttt/v1/actions/home",
			status:     http.StatusOK,
			body:       "Successfuly marked user as at home",
			users: []map[string]interface{}{
				map[string]interface{}{
					"username": "vitorarins",
				},
			},
		},
		{
			caseNumber: 3,
			route:      "/ifttt/v1/actions/home",
			status:     http.StatusOK,
			body:       "Successfuly marked user as at home",
			users: []map[string]interface{}{
				map[string]interface{}{
					"username": "vitorarins",
					"home":     false,
				},
			},
		},
		{
			caseNumber: 4,
			route:      "/ifttt/v1/actions/home",
			status:     http.StatusOK,
			body:       "Successfuly marked user as at home",
			users: []map[string]interface{}{
				map[string]interface{}{
					"username": "vitorarins",
					"home":     true,
				},
			},
		},
		{
			caseNumber: 5,
			route:      "/ifttt/v1/actions/home",
			status:     http.StatusOK,
			body:       "Successfuly marked user as at home",
			users: []map[string]interface{}{
				map[string]interface{}{
					"username": "vitorarins",
					"home":     true,
				},
				map[string]interface{}{
					"username": "testuser",
					"home":     true,
				},
			},
		},
		{
			caseNumber: 6,
			route:      "/ifttt/v1/actions/home",
			status:     http.StatusOK,
			body:       "Successfuly marked user as at home",
			users: []map[string]interface{}{
				map[string]interface{}{
					"username": "vitorarins",
					"home":     true,
				},
				map[string]interface{}{
					"username": "testuser",
					"home":     false,
				},
			},
		},
		{
			caseNumber: 7,
			route:      "/ifttt/v1/actions/home",
			status:     http.StatusOK,
			body:       "Successfuly marked user as at home",
			users: []map[string]interface{}{
				map[string]interface{}{
					"username": "vitorarins",
					"home":     true,
				},
				map[string]interface{}{
					"username": "testuser",
				},
			},
		},
	}

	for _, test := range tests {
		deleteCollection(ctx, firestoreClient, firestoreClient.Collection("users"), 10)
		for _, user := range test.users {
			firestoreClient.Collection("users").Doc(user["username"].(string)).Set(ctx, user, firestore.MergeAll)
		}

		req, err := http.NewRequest("GET", test.route, nil)
		if err != nil {
			t.Fatal(err)
		}

		q := req.URL.Query()
		q.Add("access_token", globalToken.AccessToken)
		req.URL.RawQuery = q.Encode()

		rr := httptest.NewRecorder()
		server := http.HandlerFunc(handler.HomeHandler)
		server.ServeHTTP(rr, req)

		if status := rr.Code; status != test.status {
			t.Errorf("unexpected status on test case '%v': got (%v) want (%v)", test.caseNumber, status, test.status)
		}

		if rr.Body.String() != test.body {
			t.Errorf("unexpected body on test case '%v': got (%v) want (%v)", test.caseNumber, rr.Body.String(), test.body)
		}
	}
}

func restoreCookies(request *http.Request) {
	for _, cookie := range globalCookieJar {
		request.AddCookie(cookie)
	}
}

func deleteCollection(ctx context.Context, client *firestore.Client,
	ref *firestore.CollectionRef, batchSize int) error {

	for {
		// Get a batch of documents
		iter := ref.Limit(batchSize).Documents(ctx)
		numDeleted := 0

		// Iterate through the documents, adding
		// a delete operation for each one to a
		// WriteBatch.
		batch := client.Batch()
		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}

			batch.Delete(doc.Ref)
			numDeleted++
		}

		// If there are no documents to delete,
		// the process is over.
		if numDeleted == 0 {
			return nil
		}

		_, err := batch.Commit(ctx)
		if err != nil {
			return err
		}
	}
}
