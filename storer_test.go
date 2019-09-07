package main

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"testing"
)

func TestStorer(t *testing.T) {

	tests := []struct {
		name   string
		status string
		putErr string
		getErr string
	}{
		{
			name:   "1-Voordeur",
			status: "On",
		},
		{
			name:   "2 Balkondeur",
			status: "Off",
		},
		{
			name:   "3-Woonkamer-Pir",
			status: "",
		},
		{
			name:   "",
			status: "Off",
			putErr: "Name cannot be empty (name: , status: Off)",
			getErr: "Cannot get detector with empty name",
		},
		{
			name:   "",
			status: "",
			putErr: "Name cannot be empty (name: , status: )",
			getErr: "Cannot get detector with empty name",
		},
	}

	ctx := context.Background()
	client, err := firestore.NewClient(ctx, "test")
	if err != nil {
		t.Fatalf("Could not create firestore client: %v", err)
	}

	storer := NewStorer(ctx, client)

	for _, test := range tests {

		if err := storer.PutDetector(test.name, test.status); err != nil {
			if test.putErr != fmt.Sprintf("%v", err) {
				t.Errorf("unexpected error: got (%v) when putting detector (%v) with status (%v). Maybe it should be (%v)", err, test.name, test.status, test.putErr)
			}
		}

		if _, err := storer.GetDetector(test.name); err != nil {
			if test.getErr != fmt.Sprintf("%v", err) {
				t.Errorf("unexpected error: got (%v) when getting detector (%v) with status (%v). Maybe it should be (%v)", err, test.name, test.status, test.getErr)
			}
		}
	}
}
