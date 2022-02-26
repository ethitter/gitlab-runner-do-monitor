package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/digitalocean/godo"
	"github.com/dustin/go-humanize"
	"golang.org/x/oauth2"
)

type config struct {
	LogDest     string `json:"log-dest"`
	APIKey      string `json:"api-key"`
	Threshold   int    `json:"threshold"`
	DeleteStale bool   `json:"delete-stale"`
}

type tokenSource struct {
	AccessToken string
}

var (
	configPath string

	logger *log.Logger

	apiKey      string
	threshold   int
	deleteStale bool

	wg sync.WaitGroup

	client *godo.Client
)

func initConfig() {
	flag.StringVar(&configPath, "config", "./config.json", "Path to configuration file")
	flag.Parse()

	if !validatePath(&configPath) {
		usage()
	}

	configFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		usage()
	}

	cfg := config{}
	if err = json.Unmarshal(configFile, &cfg); err != nil {
		usage()
	}

	apiKey = cfg.APIKey
	threshold = cfg.Threshold
	deleteStale = cfg.DeleteStale

	setUpLogger(cfg.LogDest)

	logger.Printf("Starting GitLab Runner monitoring with config %s", configPath)

	if deleteStale {
		logger.Println("Stale droplets WILL BE DELETED automatically")
	} else {
		logger.Println("Stale droplets will be logged, but not deleted")
	}
}

func main() {
	initConfig()
	authenticate()
	checkAPI()

	wg.Wait()
	logger.Println("Execution complete!")
}

func authenticate() {
	tokenSource := &tokenSource{
		AccessToken: apiKey,
	}

	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	client = godo.NewClient(oauthClient)
}

// oAuth token
func (t *tokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}

	return token, nil
}

func checkAPI() {
	ctx := context.TODO()
	droplets, err := listDroplets(ctx, client)
	if err != nil {
		logger.Println("Warning! Failed to retrieve droplet list.")
		logger.Print(err)
		return
	}

	for _, droplet := range droplets {
		wg.Add(1)
		go checkDroplet(droplet)
	}
}

func listDroplets(ctx context.Context, client *godo.Client) ([]godo.Droplet, error) {
	list := []godo.Droplet{}

	opt := &godo.ListOptions{}
	for {
		droplets, resp, err := client.Droplets.List(ctx, opt)
		if err != nil {
			return nil, err
		}

		for _, d := range droplets {
			list = append(list, d)
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, err
		}

		opt.Page = page + 1
	}

	return list, nil
}

func checkDroplet(droplet godo.Droplet) {
	defer wg.Done()

	if !checkDropletAge(droplet) {
		return
	}

	deleted := deleteDroplet(droplet)
	if deleteStale {
		if deleted {
			logger.Printf("Removed droplet %s (%d)", droplet.Name, droplet.ID)
		} else {
			logger.Printf("Failed to delete droplet %s (%d)", droplet.Name, droplet.ID)
		}
	}
}

func checkDropletAge(droplet godo.Droplet) bool {
	thr := time.Now().Add(time.Duration(-threshold) * time.Second)
	created, err := time.Parse(time.RFC3339, droplet.Created)
	if err != nil {
		logger.Printf("Could not parse created-timestamp for droplet %s (%d)", droplet.Name, droplet.ID)
		return false
	}

	stale := thr.After(created)

	if stale {
		logger.Printf("Stale droplet => ID: %d; name: \"%s\"; created: %s, %s (%d)", droplet.ID, droplet.Name, humanize.Time(created), droplet.Created, created.Unix())
	}

	return stale
}

func deleteDroplet(droplet godo.Droplet) bool {
	if !deleteStale {
		return false
	}

	logger.Printf("Deleting droplet %s (%d)", droplet.Name, droplet.ID)

	ctx := context.TODO()
	_, err := client.Droplets.Delete(ctx, droplet.ID)

	return err == nil
}

func setUpLogger(logDest string) {
	logOpts := log.Ldate | log.Ltime | log.LUTC | log.Lshortfile

	if logDest == "os.Stdout" {
		logger = log.New(os.Stdout, "DEBUG: ", logOpts)
	} else {
		path, err := filepath.Abs(logDest)
		if err != nil {
			logger.Fatal(err)
		}

		logFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Fatal(err)
		}

		logger = log.New(logFile, "", logOpts)
	}
}

func validatePath(path *string) bool {
	if len(*path) <= 1 {
		return false
	}

	var err error
	*path, err = filepath.Abs(*path)

	if err != nil {
		logger.Printf("Error: %s", err.Error())
		return false
	}

	if _, err = os.Stat(*path); os.IsNotExist(err) {
		return false
	}

	return true
}

func usage() {
	flag.Usage()
	os.Exit(3)
}
