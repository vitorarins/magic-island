package main

import (
	"fmt"
	"net/http"
	"path"
)

type Handler interface {
	IndexHandler(w http.ResponseWriter, r *http.Request)
	AlarmHandler(w http.ResponseWriter, r *http.Request)
}

type handlerImpl struct {
	requester      Requester
	allowedActions map[string]string
}

func NewHandler(requester Requester) Handler {
	return &handlerImpl{
		requester: requester,
		allowedActions: map[string]string{
			"arm":     "arm",
			"partarm": "partarm",
			"disarm":  "disarm",
		},
	}
}

// IndexHandler responds to requests with our greeting.
func (h *handlerImpl) IndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	fmt.Fprint(w, "Matrix")
}

// AlarmHandler sets up the alarm system with arm, partarm or disarm
func (h *handlerImpl) AlarmHandler(w http.ResponseWriter, r *http.Request) {

	action, ok := h.allowedActions[path.Base(r.URL.Path)]
	if !ok {
		http.NotFound(w, r)
		return
	}
	h.requester.RequestFeenstra(action)
	fmt.Fprintf(w, "Successfuly executed action %s", action)
}
