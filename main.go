package main

import (
	"github.com/alecthomas/kingpin"
	"log"
	"os"
	"runtime"
	"time"
)

var (
	buildDate string
	goVersion = runtime.Version()

	// flags
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

	for {
		ManageDectetorsAlert(statusDir, requester)
		time.Sleep(1 * time.Second)
	}
}
