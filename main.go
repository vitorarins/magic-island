package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/alecthomas/kingpin"
)

var (
	buildDate string
	goVersion = runtime.Version()

	// flags
	port              = kingpin.Flag("port", "The port to be allocated for this http service.").Default("8080").Envar("PORT").String()
	actionsLocation   = kingpin.Flag("actions-location", "The location where to get data for actions against APIs.").Default("action-data").Envar("ACTIONS_LOCATION").String()
	feenstraPassCode  = kingpin.Flag("pass-code", "Pass code used for Feenstra system.").Envar("PASS_CODE").Required().String()
	feenstraKey       = kingpin.Flag("feenstra-key", "Key used for requests against Feenstra sytem.").Envar("FEENSTRA_KEY").Required().String()
	makerKey          = kingpin.Flag("maker-key", "Key used for requests against IFTT Maker sytem.").Envar("MAKER_KEY").Required().String()
	firestoreProject  = kingpin.Flag("firestore-project", "Id of GCP project of firestore instance.").Envar("FIRESTORE_PROJECT_ID").Required().String()
	oauthClientId     = kingpin.Flag("client-id", "Id of Client to do OAuth.").Envar("OAUTH_CLIENT_ID").Required().String()
	oauthClientSecret = kingpin.Flag("client-secret", "OAuth server client secret.").Envar("OAUTH_CLIENT_SECRET").Required().String()
	redirectURIs      = kingpin.Flag("redirect-uris", "Comma separated list of authorized redirect URIs.").Envar("REDIRECT_URIS").String()
	domain            = kingpin.Flag("domain", "Domain that this application will serve.").Envar("DOMAIN").Required().String()
)

func main() {

	// parse command line parameters
	kingpin.Parse()
	redirectURIList := strings.Split(*redirectURIs, ",")

	// log to stdout and hide timestamp
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	log.Println("Alarm System is up and running...")

	// setup firestore client
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, *firestoreProject)
	if err != nil {
		log.Fatalf("Could not create firestore client: %v", err)
	}

	// setup requester, storer and http handler
	requester := NewRequester(*actionsLocation, *feenstraPassCode, *feenstraKey, *makerKey)
	storer := NewStorer(ctx, client)
	handler := NewHandler(*oauthClientId, *oauthClientSecret, *domain, redirectURIList, requester, client)

	http.HandleFunc("/login", handler.LoginHandler)
	http.HandleFunc("/auth", handler.AuthHandler)
	http.HandleFunc("/authorize", handler.AuthorizeHandler)
	http.HandleFunc("/token", handler.TokenHandler)
	http.HandleFunc("/", handler.IndexHandler)
	http.HandleFunc("/alarm/", handler.AlarmHandler)
	http.HandleFunc("/ifttt/v1/actions/partarm", handler.AlarmHandler)
	http.HandleFunc("/ifttt/v1/actions/disarm", handler.AlarmHandler)
	http.HandleFunc("/ifttt/v1/actions/fullarm", handler.AlarmHandler)
	http.HandleFunc("/ifttt/v1/user/info", handler.IFTTTHandler)
	http.HandleFunc("/ifttt/v1/status", handler.StatusHandler)
	http.HandleFunc("/status", handler.StatusHandler)
	http.HandleFunc("/ifttt/v1/actions/nothome", handler.NotHomeHandler)
	http.HandleFunc("/ifttt/v1/actions/home", handler.HomeHandler)

	log.Println("Managing Detectors Alert")
	go ManageDectetorsAlert(storer, requester)

	log.Printf("Listening on port %s", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", *port), nil))
}
