package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeRequester struct {
	tee *testing.T
}

func (f *fakeRequester) RequestFeenstra(action string) string {
	f.tee.Logf("RequestFeenstra was called with action: %v", action)
	return "big xml"
}

func (f *fakeRequester) RequestMaker(detector, status string) string {
	f.tee.Logf("RequestMaker was called with detect '%v' and status '%v'", detector, status)
	return "maker response"
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

	requester := &fakeRequester{
		tee: t,
	}
	handler := NewHandler(requester)

	for _, test := range tests {
		req, err := http.NewRequest("GET", test.route, nil)
		if err != nil {
			t.Fatal(err)
		}

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

	requester := &fakeRequester{
		tee: t,
	}
	handler := NewHandler(requester)

	for _, test := range tests {
		req, err := http.NewRequest("GET", test.route, nil)
		if err != nil {
			t.Fatal(err)
		}

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
