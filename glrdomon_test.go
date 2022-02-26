package main

import (
	"testing"
	"time"

	"github.com/digitalocean/godo"
)

func TestCheckDropletAge(t *testing.T) {
	threshold = 3600
	setUpLogger("os.Stdout")

	staleDroplet := godo.Droplet{
		ID:      1234,
		Name:    "very-old",
		Created: "2000-08-19T19:07:04Z",
	}

	if !checkDropletAge(staleDroplet) {
		t.Error("Failed to assert that very old droplet is stale")
	}

	now := time.Now()
	newDroplet := godo.Droplet{
		ID:      5678,
		Name:    "new",
		Created: now.Format(time.RFC3339),
	}

	if checkDropletAge(newDroplet) {
		t.Error("Asserted that brand-new droplet is stale")
	}
}

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
