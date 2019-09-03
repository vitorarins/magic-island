package main

import (
	"fmt"
	"github.com/alecthomas/kingpin"
	"log"
	"net/http"
	"os"
	"runtime"
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
)

func main() {

	// parse command line parameters
	kingpin.Parse()

	// log to stdout and hide timestamp
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))

	log.Println("Feenstra is up and running...")

	var statusDir = "status"

	if _, err := os.Stat(statusDir); os.IsNotExist(err) {
		err := os.Mkdir(statusDir, os.ModePerm)
		if err != nil {
			log.Printf("Error creating dir %s: %v\n", statusDir, err)
			os.Exit(1)
		}
	}

	requester := NewRequester(*actionsLocation, *feenstraPassCode, *feenstraKey, *makerKey)

	http.HandleFunc("/", indexHandler)

	log.Println("Managing Detectors Alert")
	go ManageDectetorsAlert(statusDir, requester)

	log.Printf("Listening on port %s", *port)
	log.Fatal(http.ListenAndServeTLS(fmt.Sprintf(":%s", *port), "server.crt", "server.key", nil))
}
