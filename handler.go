package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/go-session/session"
	"github.com/tslamic/go-oauth2-firestore"
	"gopkg.in/oauth2.v3/errors"
	"gopkg.in/oauth2.v3/manage"
	"gopkg.in/oauth2.v3/models"
	"gopkg.in/oauth2.v3/server"
	"gopkg.in/oauth2.v3/store"
)

type Handler interface {
	IndexHandler(w http.ResponseWriter, r *http.Request)
	AlarmHandler(w http.ResponseWriter, r *http.Request)
	AuthorizeHandler(w http.ResponseWriter, r *http.Request)
	TokenHandler(w http.ResponseWriter, r *http.Request)
	StatusHandler(w http.ResponseWriter, r *http.Request)
	IFTTTHandler(w http.ResponseWriter, r *http.Request)
	LoginHandler(w http.ResponseWriter, r *http.Request)
	AuthHandler(w http.ResponseWriter, r *http.Request)
}

type handlerImpl struct {
	requester      Requester
	allowedActions map[string]string
	srv            *server.Server
}

func NewHandler(oauthClientId, oauthClientSecret, domain string, redirectURIs []string, requester Requester, firestoreClient *firestore.Client) Handler {

	// setup OAuth stuff
	manager := manage.NewDefaultManager()
	manager.SetRefreshTokenCfg(&manage.RefreshingConfig{IsGenerateRefresh: true, IsRemoveAccess: false, IsRemoveRefreshing: false})
	manager.SetValidateURIHandler(func(baseURI string, redirectURI string) (err error) {
		base, err := url.Parse(baseURI)
		if err != nil {
			return
		}
		redirect, err := url.Parse(redirectURI)
		if err != nil {
			return
		}
		if !strings.HasSuffix(redirect.Host, base.Host) {
			for _, uri := range redirectURIs {
				if redirectURI == uri {
					return
				}
			}
			err = errors.ErrInvalidRedirectURI
		}
		return
	})

	// token firestore
	storage := fstore.New(firestoreClient, "tokens")
	manager.MapTokenStorage(storage)
	// client memory store
	clientStore := store.NewClientStore()
	clientStore.Set(oauthClientId, &models.Client{
		ID:     oauthClientId,
		Secret: oauthClientSecret,
		Domain: domain,
	})
	manager.MapClientStorage(clientStore)
	srv := server.NewDefaultServer(manager)
	srv.SetAllowGetAccessRequest(true)
	srv.SetClientInfoHandler(server.ClientFormHandler)

	srv.SetUserAuthorizationHandler(userAuthorizeHandler)

	srv.SetInternalErrorHandler(func(err error) (re *errors.Response) {
		log.Println("Internal Error:", err.Error())
		return
	})

	srv.SetResponseErrorHandler(func(re *errors.Response) {
		log.Println("Response Error:", re.Error.Error())
	})

	return &handlerImpl{
		requester: requester,
		srv:       srv,
		allowedActions: map[string]string{
			"fullarm": "arm",
			"arm":     "arm",
			"partarm": "partarm",
			"disarm":  "disarm",
		},
	}
}

// IndexHandler responds to requests with our greeting.
func (h *handlerImpl) IndexHandler(w http.ResponseWriter, r *http.Request) {
	_, err := h.srv.ValidationBearerToken(r)
	if err != nil {
		log.Printf("Error validating token: %v", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	fmt.Fprint(w, "Matrix")
}

// AlarmHandler sets up the alarm system with arm, partarm or disarm
func (h *handlerImpl) AlarmHandler(w http.ResponseWriter, r *http.Request) {
	_, err := h.srv.ValidationBearerToken(r)
	if err != nil {
		log.Printf("Error validating token: %v", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	action, ok := h.allowedActions[path.Base(r.URL.Path)]
	if !ok {
		http.NotFound(w, r)
		return
	}
	h.requester.RequestFeenstra(action)
	fmt.Fprintf(w, "Successfuly executed action %s", action)
}

// AuthorizeHandler authorizes oauth clients
func (h *handlerImpl) AuthorizeHandler(w http.ResponseWriter, r *http.Request) {
	err := h.srv.HandleAuthorizeRequest(w, r)
	if err != nil {
		log.Printf("Error authorizing client: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

// TokenHandler creates refresh tokens for oauth clients
func (h *handlerImpl) TokenHandler(w http.ResponseWriter, r *http.Request) {
	h.srv.HandleTokenRequest(w, r)
}

// StatusHandler always responds with 200 OK
func (h *handlerImpl) StatusHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "OK")
}

// IFTTTHandler handles every request that is not an action from IFTTT
func (h *handlerImpl) IFTTTHandler(w http.ResponseWriter, r *http.Request) {
	_, err := h.srv.ValidationBearerToken(r)
	if err != nil {
		log.Printf("Error validating token: %v", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	switch requestPath := r.URL.Path; requestPath {
	case "/ifttt/v1/user/info":
		data := map[string]interface{}{
			"data": map[string]string{
				"name": "Only user",
				"id":   "onlyuserwehave",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(data)
	default:
		http.NotFound(w, r)
	}
}

func userAuthorizeHandler(w http.ResponseWriter, r *http.Request) (userID string, err error) {
	store, err := session.Start(nil, w, r)
	if err != nil {
		return
	}

	uid, ok := store.Get("LoggedInUserID")
	if !ok {
		if r.Form == nil {
			r.ParseForm()
		}

		store.Set("ReturnUri", r.Form)
		store.Save()

		w.Header().Set("Location", "/login")
		w.WriteHeader(http.StatusFound)
		return
	}

	userID = uid.(string)
	store.Delete("LoggedInUserID")
	store.Save()
	return
}

func (h *handlerImpl) LoginHandler(w http.ResponseWriter, r *http.Request) {
	store, err := session.Start(nil, w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Method == "POST" {
		if r.Form == nil {
			if err := r.ParseForm(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		store.Set("LoggedInUserID", r.Form.Get("username"))
		store.Save()

		w.Header().Set("Location", "/auth")
		w.WriteHeader(http.StatusFound)
		return
	}
	outputHTML(w, r, "static/login.html")
}

func (h *handlerImpl) AuthHandler(w http.ResponseWriter, r *http.Request) {
	store, err := session.Start(nil, w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, ok := store.Get("LoggedInUserID"); !ok {
		w.Header().Set("Location", "/login")
		w.WriteHeader(http.StatusFound)
		return
	}

	outputHTML(w, r, "static/auth.html")
}

func outputHTML(w http.ResponseWriter, req *http.Request, filename string) {
	file, err := os.Open(filename)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer file.Close()
	fi, _ := file.Stat()
	http.ServeContent(w, req, file.Name(), fi.ModTime(), file)
}
