package main

import (
	"encoding/xml"
	"log"
	"strings"
	"time"
)

type Zone struct {
	Id     int64  `xml:"ID"`
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

func ManageDectetorsAlert(storer Storer, requester Requester) {

	for {
		detectorsXML := requester.RequestFeenstra("get-detectors")

		detectorsList, err := parseDetectors(detectorsXML)
		if err != nil {
			log.Printf("Got the following error trying to parse the detectors XML: %s", err)
			log.Printf("Detectors xml: \n%v", detectorsXML)
		}

		for _, detector := range detectorsList {
			detectorSafeName := strings.Replace(detector.Name, " ", "-", -1)
			storedDetector, err := storer.GetDetector(detectorSafeName)
			if err != nil {
				log.Printf("Could not read stored status for detector '%v': %v", detectorSafeName, err)
				err = storer.PutDetector(detectorSafeName, detector.Status)
				if err != nil {
					log.Printf("Got the following error trying to save detector: %s", err)
				}
				continue
			}

			if storedDetector.Status != detector.Status {
				log.Printf("Alerting for detector: %s with current status: %s", detectorSafeName, detector.Status)
				requester.RequestMakerDetector(detectorSafeName, detector.Status)
				err = storer.PutDetector(detectorSafeName, detector.Status)
				if err != nil {
					log.Printf("Got the following error trying to save detector: %s", err)
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
}
