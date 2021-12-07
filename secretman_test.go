package main

import (
	"testing"
)

var fakeAccessSecrets secretAccess = func(name string) (string, error) {
	return "new", nil
}

func TestPopulateFlags(t *testing.T) {
	oldTestString := "old"
	oldTest := &oldTestString
	testCases := map[string]struct {
		input   map[string]*string
		want    map[string]string
		wantErr error
	}{
		"success": {
			input: map[string]*string{
				"MAKER_KEY": oldTest,
			},
			want: map[string]string{
				"MAKER_KEY": "new",
			},
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			sa := SecretAccessor{
				projectName: "test",
			}
			err := sa.PopulateFlags(testCase.input, fakeAccessSecrets)
			if err != testCase.wantErr {
				t.Errorf("got '%v', wanted '%v'", err, testCase.wantErr)
			}

			for k, _ := range testCase.input {
				got := testCase.input[k]
				want := testCase.want[k]
				if *got != want {
					t.Errorf("got '%v', wanted '%v'", *got, want)
				}
			}

			if *oldTest != "new" {
				t.Errorf("got '%v', wanted 'new'", *oldTest)
			}
		})
	}
}
