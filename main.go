package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"

	"cloud.google.com/go/datastore"
	"github.com/alecthomas/kingpin"
)

var (
	buildDate string
	goVersion = runtime.Version()

	// flags
	port             = kingpin.Flag("port", "The port to be allocated for this http service.").Default("443").Envar("PORT").String()
	actionsLocation  = kingpin.Flag("actions-location", "The location where to get data for actions against APIs.").Default("action-data").Envar("ACTIONS_LOCATION").String()
	feenstraPassCode = kingpin.Flag("pass-code", "Pass code used for Feenstra system.").Envar("PASS_CODE").Required().String()
	feenstraKey      = kingpin.Flag("feenstra-key", "Key used for requests against Feenstra sytem.").Envar("FEENSTRA_KEY").Required().String()
	makerKey         = kingpin.Flag("maker-key", "Key used for requests against IFTT Maker sytem.").Envar("MAKER_KEY").Required().String()
	datastoreProject = kingpin.Flag("datastore-project", "Id of GCP project of datastore instance.").Envar("DATASTORE_PROJECT_ID").Required().String()
)

func main() {

	// parse command line parameters
	kingpin.Parse()

	// log to stdout and hide timestamp
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))

	log.Println("Alarm System is up and running...")

	ctx := context.Background()
	client, err := datastore.NewClient(ctx, *datastoreProject)
	if err != nil {
		log.Fatalf("Could not create datastore client: %v", err)
	}

	requester := NewRequester(*actionsLocation, *feenstraPassCode, *feenstraKey, *makerKey)
	storer := NewStorer(ctx, client)
	handler := NewHandler(requester)

	http.HandleFunc("/", handler.IndexHandler)

	log.Println("Managing Detectors Alert")
	go ManageDectetorsAlert(storer, requester)

	log.Printf("Listening on port %s", *port)
	log.Fatal(http.ListenAndServeTLS(fmt.Sprintf(":%s", *port), "server.crt", "server.key", nil))
}
