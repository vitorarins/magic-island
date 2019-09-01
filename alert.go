package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
)

type Zone struct {
	Id     string `xml:"ID"`
	Name   string `xml:"Name"`
	Status string `xml:"Status"`
}

type Body struct {
	XMLName xml.Name `xml:"Body"`
	Zones   []Zone   `xml:"GetCPStateResponse>Rep>ECReply>Zones"`
}

type RespEnvelope struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    Body     `xml:"Body"`
}

func parseDetectors(detectorsXML string) ([]Zone, error) {

	var envelope RespEnvelope

	err := xml.Unmarshal([]byte(detectorsXML), &envelope)
	if err != nil {
		return nil, err
	}

	return envelope.Body.Zones, nil
}

func writeStatusFile(content, filename string) error {
	message := []byte(content)
	err := ioutil.WriteFile(filename, message, 0644)
	if err != nil {
		return err
	}
	return nil
}

func ManageDectetorsAlert(detectorStatusDir string, requester Requester) {
	detectorsXML := requester.RequestFeenstra("get-detectors")

	detectorsList, err := parseDetectors(detectorsXML)
	if err != nil {
		log.Printf("Got the following error trying to parse the detectors XML: %s", err)
		log.Printf("Detectors xml: \n%v", detectorsXML)
	}

	for _, detector := range detectorsList {
		detectorSafeName := strings.Replace(detector.Name, " ", "-", -1)
		detectorStatusFilename := fmt.Sprintf("%s/%s", detectorStatusDir, detectorSafeName)
		status, err := ioutil.ReadFile(detectorStatusFilename)
		if err != nil {
			log.Println("Error reading status file:", err)
		}
		storedDetectorStatus := string(status)

		if storedDetectorStatus != "" {
			if storedDetectorStatus != detector.Status {
				fmt.Printf("Alerting for detector: %s with current status: %s", detectorSafeName, detector.Status)
				requester.RequestMaker(detectorSafeName, detector.Status)
			}
		}

		err = writeStatusFile(detector.Status, detectorStatusFilename)
		if err != nil {
			log.Printf("Got the following error trying to write status file: %s", err)
		}
	}
}
