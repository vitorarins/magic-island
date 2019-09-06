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
	requester := &fakeRequester{
		tee: t,
	}

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler := NewHandler(requester)
	rr := httptest.NewRecorder()
	server := http.HandlerFunc(handler.IndexHandler)
	server.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf(
			"unexpected status: got (%v) want (%v)",
			status,
			http.StatusOK,
		)
	}

	expected := "Alarm System"
	if rr.Body.String() != expected {
		t.Errorf(
			"unexpected body: got (%v) want (%v)",
			rr.Body.String(),
			expected,
		)
	}
}

func TestIndexHandlerNotFound(t *testing.T) {
	requester := &fakeRequester{
		tee: t,
	}

	req, err := http.NewRequest("GET", "/404", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler := NewHandler(requester)
	rr := httptest.NewRecorder()
	server := http.HandlerFunc(handler.IndexHandler)
	server.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf(
			"unexpected status: got (%v) want (%v)",
			status,
			http.StatusNotFound,
		)
	}
}
