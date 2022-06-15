package risco

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

type RiscoError int

type Detector struct {
	Id       int    `json:id`
	Name     string `json:name`
	ByPassed bool   `json:bypassed`
}

type Part struct {
	Name      string     `json:name`
	Detectors []Detector `json:detectors`
}

type Detectors struct {
	Parts []Part `json:parts`
}

type RiscoResponse struct {
	Error     RiscoError `json:error`
	Detectors *Detectors `json:detectors`
}

type RiscoClient struct {
	FeenstraUsername string
	FeenstraPassword string
	FeenstraPassCode string
	FeenstraUrl      string
	RiscoJar         http.CookieJar
}

const (
	NoneRiscoError        RiscoError = 0
	NotLoggedInRiscoError RiscoError = 3
)

func NewRiscoClient(feenstraUsername, feenstraPassword, feenstraPassCode string) *RiscoClient {

	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatalf("Got error while creating cookie jar %s", err.Error())
	}

	return &RiscoClient{
		FeenstraUsername: feenstraUsername,
		FeenstraPassword: feenstraPassword,
		FeenstraPassCode: feenstraPassCode,
		FeenstraUrl:      "https://www.feenstraveilig.nl/ELAS/WebUI/",
		RiscoJar:         jar,
	}
}

func (r *RiscoClient) Login() error {
	requestUrl, err := url.Parse(r.FeenstraUrl)
	if err != nil {
		return err
	}

	query := requestUrl.Query()
	query.Set("username", r.FeenstraUsername)
	query.Set("password", r.FeenstraPassword)
	query.Set("code", r.FeenstraPassCode)

	requestUrl.RawQuery = query.Encode()

	req, err := http.NewRequest("POST", requestUrl.String(), nil)
	if err != nil {
		return err
	}

	timeout := time.Duration(30 * time.Second)

	client := http.Client{
		Timeout: timeout,
		Jar:     r.RiscoJar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 302 {
		return fmt.Errorf("Response failed with status code: %d and\nbody: %s\n", resp.StatusCode, body)
	}

	return nil
}

func (r *RiscoClient) GetDetectors() ([]Detector, error) {
	requestUrl, err := url.Parse(r.FeenstraUrl + "Detectors/Get")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", requestUrl.String(), nil)
	if err != nil {
		return nil, err
	}

	timeout := time.Duration(30 * time.Second)

	client := http.Client{
		Timeout: timeout,
		Jar:     r.RiscoJar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Response failed with status code: %d and\nbody: %s\n", resp.StatusCode, body)
	}

	riscoResponse := RiscoResponse{}
	if err := json.Unmarshal(body, &riscoResponse); err != nil {
		return nil, err
	}

	if riscoResponse.Error == NotLoggedInRiscoError {
		if err := r.Login(); err != nil {
			return nil, fmt.Errorf("failed to login while getting detectors: %w", err)
		}
	}

	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Response failed with status code: %d and\nbody: %s\n", resp.StatusCode, body)
	}

	if err := json.Unmarshal(body, &riscoResponse); err != nil {
		return nil, err
	}

	if riscoResponse.Detectors != nil &&
		riscoResponse.Detectors.Parts != nil &&
		len(riscoResponse.Detectors.Parts) == 1 &&
		riscoResponse.Detectors.Parts[0].Detectors != nil {

		return riscoResponse.Detectors.Parts[0].Detectors, nil
	}

	return nil, fmt.Errorf("response doesn't have detectors: %v", riscoResponse)
}
