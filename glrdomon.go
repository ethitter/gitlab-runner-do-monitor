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
)

type config struct {
	DebugDest string `json:"debug-dest"`
	Debug     bool   `json:"debug"`
	ApiKey    string `json:"api-key"`
	Threshold int    `json:"threshold"`
}

var (
	configPath string

	logger    *log.Logger
	debugDest string
	debug     bool
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

	setUpLogger()
}

func main() {
	logger.Printf("Starting GitLab Runner monitoring with config %s", configPath)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	// TODO: something!
	if debug {
		logger.Println("Test")
	}

	caughtSig := <-sig

	logger.Printf("Stopping, got signal %s", caughtSig)
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
