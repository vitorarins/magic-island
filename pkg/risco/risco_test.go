package risco_test

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/vitorarins/magic-island/pkg/risco"
)

func TestLogin(t *testing.T) {
	feenstraUsername := os.Getenv("FEENSTRA_USERNAME")
	feenstraPassword := os.Getenv("FEENSTRA_PASSWORD")
	feenstraPassCode := os.Getenv("PASS_CODE")
	requester := risco.NewRiscoClient(feenstraUsername, feenstraPassword, feenstraPassCode)

	tests := map[string]struct {
		wantErr bool
	}{
		"success": {
			wantErr: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := requester.Login()

			if err != nil && !test.wantErr {
				t.Errorf("Expected no errors but got: %v", err)
			}

			if err == nil && test.wantErr {
				t.Errorf("Expected errors but got nothing")
			}
		})
	}
}

func TestGetDetectors(t *testing.T) {
	feenstraUsername := os.Getenv("FEENSTRA_USERNAME")
	feenstraPassword := os.Getenv("FEENSTRA_PASSWORD")
	feenstraPassCode := os.Getenv("PASS_CODE")
	requester := risco.NewRiscoClient(feenstraUsername, feenstraPassword, feenstraPassCode)

	tests := map[string]struct {
		want    []risco.Detector
		wantErr bool
	}{
		"success": {
			want: []risco.Detector{
				{Id: 0, Name: "1 Voordeur"},
				{Id: 1, Name: "2 Meterkast"},
				{Id: 2, Name: "3 Hal Pir"},
				{Id: 3, Name: "4 Hal Rook"},
				{Id: 4, Name: "5 Woonkamer Pir"},
				{Id: 5, Name: "6 Keukendeur"},
				{Id: 6, Name: "7 Balkondeur"},
			},
			wantErr: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := requester.GetDetectors()

			if err != nil && !test.wantErr {
				t.Errorf("Expected no errors but got: %v", err)
			}

			if err == nil && test.wantErr {
				t.Errorf("Expected errors but got nothing")
			}

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("GetDetectors() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
