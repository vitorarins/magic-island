package main

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestParseNames(t *testing.T) {
	t.Run("ReturnsListOfNamesIfXMLIsValid", func(t *testing.T) {
		xmlFile, _ := ioutil.ReadFile("testdata/detectors.xml")
		xml := string(xmlFile)
		got, err := parseDetectors(xml)

		assert.Len(t, got, 7)

		assert.Equal(t, got[0].Id, "0")
		assert.Equal(t, got[0].Name, "1 Voordeur")
		assert.Equal(t, got[0].Status, "Off")

		assert.Equal(t, got[1].Id, "1")
		assert.Equal(t, got[1].Name, "2 Meterkast")
		assert.Equal(t, got[1].Status, "Off")

		assert.Equal(t, got[2].Id, "2")
		assert.Equal(t, got[2].Name, "3 Hal Pir")
		assert.Equal(t, got[2].Status, "Off")

		assert.Equal(t, got[3].Id, "3")
		assert.Equal(t, got[3].Name, "4 Hal Rook")
		assert.Equal(t, got[3].Status, "Off")

		assert.Equal(t, got[4].Id, "4")
		assert.Equal(t, got[4].Name, "5 Woonkamer Pir")
		assert.Equal(t, got[4].Status, "Off")

		assert.Equal(t, got[5].Id, "5")
		assert.Equal(t, got[5].Name, "6 Keukendeur")
		assert.Equal(t, got[5].Status, "Off")

		assert.Equal(t, got[6].Id, "6")
		assert.Equal(t, got[6].Name, "7 Balkondeur")
		assert.Equal(t, got[6].Status, "Off")

		assert.Nil(t, err)
	})

	t.Run("ReturnsErrorIfXMLIsEmpty", func(t *testing.T) {
		xml := ""

		_, err := parseDetectors(xml)

		assert.NotNil(t, err)
	})
}
