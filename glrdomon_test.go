package main

import "testing"

func TestValidatePath(t *testing.T) {
	emptyString := ""
	notValid := validatePath(&emptyString)

	if notValid == true {
		t.Error("Empty path shouldn't validate")
	}

	sampleConfig := "./config-sample.json"
	valid := validatePath(&sampleConfig)

	if valid != true {
		t.Error("Couldn't validate path to sample config")
	}
}
