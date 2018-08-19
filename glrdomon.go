package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/robfig/cron"
)

type config struct {
	DebugDest string `json:"debug-dest"`
	Debug     bool   `json:"debug"`
	ApiKey    string `json:"api-key"`
	Threshold int    `json:"threshold"`
	Schedule  string `json:"schedule"`
}

var (
	configPath string

	logger    *log.Logger
	debugDest string
	debug     bool

	threshold int
	schedule  string
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

	threshold = cfg.Threshold
	schedule = cfg.Schedule

	setUpLogger()
}

func main() {
	logger.Printf("Starting GitLab Runner monitoring with config %s", configPath)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	if debug {
		logger.Println("Test")
	}

	if authenticate() {
		startCron()
	} else {
		sig <- syscall.SIGTERM
	}

	caughtSig := <-sig

	logger.Printf("Stopping, got signal %s", caughtSig)
}

func authenticate() bool {
	return true
}

func startCron() {
	c := cron.New()
	c.AddFunc(schedule, check)
	c.Start()
}

func check() {
	logger.Println("Check!")
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
