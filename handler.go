package main

import (
	"fmt"
	"net/http"
)

type Handler interface {
	IndexHandler(w http.ResponseWriter, r *http.Request)
}

type handlerImpl struct {
	requester Requester
}

func NewHandler(requester Requester) Handler {
	return &handlerImpl{
		requester: requester,
	}
}

// indexHandler responds to requests with our greeting.
func (h *handlerImpl) IndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	fmt.Fprint(w, "Alarm System")
}
