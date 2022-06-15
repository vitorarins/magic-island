package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"text/template"
	"time"
)

type Requester interface {
	RequestFeenstra(action string) string
	RequestMakerDetector(detector, status string) string
	RequestMaker(event string) string
}

type requesterImpl struct {
	ActionsLocation  string
	FeenstraPassCode string
	FeenstraKey      string
	FeenstraUrl      string
	MakerKey         string
	MakerUrl         string
}

func NewRequester(actionsLocation, feenstraPassCode, feenstraKey, makerKey string) Requester {
	return &requesterImpl{
		ActionsLocation:  actionsLocation,
		FeenstraPassCode: feenstraPassCode,
		FeenstraKey:      feenstraKey,
		FeenstraUrl:      "https://www.feenstraveilig.nl:450/ELAS/WUWS/WUREQUEST.ASMX",
		MakerKey:         makerKey,
		MakerUrl:         "https://maker.ifttt.com/trigger",
	}
}

func (r *requesterImpl) RequestFeenstra(action string) string {
	file := fmt.Sprintf("%v/%v.xml", r.ActionsLocation, action)

	actionTemplate, err := ioutil.ReadFile(file)
	if err != nil {
		log.Printf("Error opening action file %s", file)
		panic(err)
	}
	t := template.Must(template.New("action").Parse(string(actionTemplate)))

	var actionData bytes.Buffer
	var templateData = map[string]string{
		"PassCode": r.FeenstraPassCode,
	}
	err = t.Execute(&actionData, templateData)
	if err != nil {
		log.Printf("Error executing template for action %s", action)
		panic(err)
	}

	req, err := http.NewRequest("POST", r.FeenstraUrl, &actionData)
	if err != nil {
		log.Printf("Error creating request for action %s", action)
		panic(err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Basic %v", r.FeenstraKey))
	req.Header.Set("User-Agent", "ksoap2-android/2.6.0+")
	req.Header.Set("Content-Type", "application/soap+xml;charset=utf-8")
	req.Host = "www.feenstraveilig.nl:450"

	trRenegotiate := &http.Transport{
		MaxIdleConnsPerHost: 10,
		TLSClientConfig:     &tls.Config{Renegotiation: tls.RenegotiateFreelyAsClient},
	}
	timeout := time.Duration(30 * time.Second)

	client := http.Client{
		Transport: trRenegotiate,
		Timeout:   timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error executing request for action %s: %v", action, err)
		return ""
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return ""
	}

	return string(body)
}

func (r *requesterImpl) RequestMakerDetector(detector, status string) string {
	url := fmt.Sprintf("%v/%v-%v/with/key/%v", r.MakerUrl, detector, status, r.MakerKey)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error executing request for detector '%s' and status '%s': %v", detector, status, err)
		return ""
	}
	defer resp.Body.Close()

	log.Println("response Status:", resp.Status)
	body, _ := ioutil.ReadAll(resp.Body)
	log.Println("response Body:", string(body))

	return string(body)
}

func (r *requesterImpl) RequestMaker(event string) string {
	url := fmt.Sprintf("%v/%v/with/key/%v", r.MakerUrl, event, r.MakerKey)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error executing request for event '%s': %v", event, err)
		return ""
	}
	defer resp.Body.Close()

	log.Println("response Status:", resp.Status)
	body, _ := ioutil.ReadAll(resp.Body)
	log.Println("response Body:", string(body))

	return string(body)
}
