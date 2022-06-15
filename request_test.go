package main

import (
	"encoding/xml"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRequestFeenstra(t *testing.T) {
	feenstraPassCode := os.Getenv("PASS_CODE")
	feenstraKey := os.Getenv("FEENSTRA_KEY")
	requester := NewRequester("action-data", feenstraPassCode, feenstraKey, "")

	xmlFile, _ := ioutil.ReadFile("testdata/detectors.xml")

	defaultWant := RespEnvelope{}

	if err := xml.Unmarshal(xmlFile, &defaultWant); err != nil {
		t.Fatalf("failed to parse %q with error: %v", xmlFile, err)
	}

	tests := map[string]struct {
		action string
		want   RespEnvelope
	}{
		"success.status": {
			action: "status",
			want: RespEnvelope{
				XMLName: xml.Name{
					Space: "http://www.w3.org/2003/05/soap-envelope",
					Local: "Envelope",
				},
				Body: Body{
					XMLName: xml.Name{
						Space: "http://www.w3.org/2003/05/soap-envelope",
						Local: "Body",
					},
				},
			},
		},
		"success.partarm": {
			action: "partarm",
			want: RespEnvelope{
				XMLName: xml.Name{
					Space: "http://www.w3.org/2003/05/soap-envelope",
					Local: "Envelope",
				},
				Body: Body{
					XMLName: xml.Name{
						Space: "http://www.w3.org/2003/05/soap-envelope",
						Local: "Body",
					},
				},
			},
		},
		"success.disarm": {
			action: "disarm",
			want: RespEnvelope{
				XMLName: xml.Name{
					Space: "http://www.w3.org/2003/05/soap-envelope",
					Local: "Envelope",
				},
				Body: Body{
					XMLName: xml.Name{
						Space: "http://www.w3.org/2003/05/soap-envelope",
						Local: "Body",
					},
				},
			},
		},
		"success.get-detectors": {
			action: "get-detectors",
			want:   defaultWant,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gotString := requester.RequestFeenstra(test.action)

			got := RespEnvelope{}

			if err := xml.Unmarshal([]byte(gotString), &got); err != nil {
				t.Fatalf("failed to parse %q with error: %v", gotString, err)
			}

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("RequestFeenstra() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
