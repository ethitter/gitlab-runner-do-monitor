package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/digitalocean/godo"
	"github.com/dustin/go-humanize"
	"github.com/robfig/cron"
	"golang.org/x/oauth2"
)

type config struct {
	DebugDest   string `json:"debug-dest"`
	Debug       bool   `json:"debug"`
	APIKey      string `json:"api-key"`
	Threshold   int    `json:"threshold"`
	Schedule    string `json:"schedule"`
	DeleteStale bool   `json:"delete-stale"`
}

type tokenSource struct {
	AccessToken string
}

var (
	configPath string

	logger    *log.Logger
	debugDest string
	debug     bool

	apiKey string

	threshold   int
	schedule    string
	deleteStale bool

	client *godo.Client
)

func init() {
	flag.StringVar(&configPath, "config", "./config.json", "Path to configuration file")
	flag.Parse()

	cfgPathValid := validatePath(&configPath)
	if !cfgPathValid {
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

	debugDest = cfg.DebugDest
	debug = cfg.Debug

	apiKey = cfg.APIKey

	threshold = cfg.Threshold
	schedule = cfg.Schedule
	deleteStale = cfg.DeleteStale

	setUpLogger()
}

func main() {
	logger.Printf("Starting GitLab Runner monitoring with config %s", configPath)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	if deleteStale {
		logger.Println("Stale droplets WILL BE DELETED automatically")
	} else {
		logger.Println("Stale droplets will be logged, but not deleted")
	}

	authenticate()
	startCron()

	caughtSig := <-sig

	logger.Printf("Stopping, got signal %s", caughtSig)
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

func startCron() {
	c := cron.New()
	c.AddFunc(schedule, checkAPI)
	c.Start()
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
		go checkDropletAge(droplet)
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

func checkDropletAge(droplet godo.Droplet) {
	logger.Print(droplet.ID)
	logger.Print(droplet.Created)

	thr := time.Now().Add(time.Duration(threshold))
	created, err := time.Parse(time.RFC3339, droplet.Created)
	if err != nil {
		logger.Printf("Could not parse created-timestamp for droplet ID %d", droplet.ID)
		return
	}

	if thr.After(created) {
		logger.Printf("Stale droplet => ID: %d; name: \"%s\"; created: %s, %s (%d)", droplet.ID, droplet.Name, humanize.Time(created), droplet.Created, created.Unix())

		if deleteStale {
			deleteDroplet(droplet)
		}
	}
}

func deleteDroplet(droplet godo.Droplet) bool {
	logger.Printf("Deleting droplet %d", droplet.ID)
	return false
}

func setUpLogger() {
	logOpts := log.Ldate | log.Ltime | log.LUTC | log.Lshortfile

	if debugDest == "os.Stdout" {
		logger = log.New(os.Stdout, "DEBUG: ", logOpts)
	} else {
		path, err := filepath.Abs(debugDest)
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
