package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/axsh/leeroy/github"
	"github.com/axsh/leeroy/jenkins"
)

const (
	// VERSION is the version
	VERSION = "v0.1.0"
	// DEFAULTCONTEXT is the default github context for a build
	DEFAULTCONTEXT = "janky"
)

var (
	certFile   string
	keyFile    string
	port       string
	configFile string
	debug      bool
	version    bool

	config Config
)

// Config describes the leeroy config file
type Config struct {
	Jenkins      jenkins.Client `json:"jenkins"`
	BuildCommits string         `json:"build_commits"`
	GHToken      string         `json:"github_token"`
	GHUser       string         `json:"github_user"`
	Builds       []Build        `json:"builds"`
	User         string         `json:"user"`
	Pass         string         `json:"pass"`
	Repository   *Repository    `json:"repository"`
}

// Build describes the paramaters for a build
type Build struct {
	Repo         string `json:"github_repo"`
	Job          string `json:"jenkins_job_name"`
	Context      string `json:"context"`
	Custom       bool   `json:"custom"`
	HandleIssues bool   `json:"handle_issues"`
	IsPipeline   bool   `json:"is_pipeline"`
}

type Repository struct {
	Repo string `json:"github_repo"`
	Job  string `json:"jenkins_job_name"`
}

func init() {
	// parse flags
	flag.BoolVar(&version, "version", false, "print version and exit")
	flag.BoolVar(&version, "v", false, "print version and exit (shorthand)")
	flag.BoolVar(&debug, "d", false, "run in debug mode")
	flag.StringVar(&certFile, "cert", "", "path to ssl certificate")
	flag.StringVar(&keyFile, "key", "", "path to ssl key")
	flag.StringVar(&port, "port", "80", "port to use")
	flag.StringVar(&configFile, "config", "/etc/leeroy/config.json", "path to config file")
	flag.Parse()
}

func main() {
	// set log level
	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if version {
		fmt.Println(VERSION)
		return
	}

	// read the config file
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		logrus.Errorf("config file does not exist: %s", configFile)
		return
	}
	c, err := ioutil.ReadFile(configFile)
	if err != nil {
		logrus.Errorf("could not read config file: %v", err)
		return
	}
	if err := json.Unmarshal(c, &config); err != nil {
		logrus.Errorf("error parsing config file as json: %v", err)
		return
	}

	if config.Repository != nil {
		stopCh := make(chan os.Signal, 1)
		signal.Notify(stopCh, syscall.SIGTERM, syscall.SIGINT)

		w := github.NewGithubWatcher(config.GHToken)
		slug := strings.SplitN(config.Repository.Repo, "/", 2)
		ch, errCh := w.PollRepository(slug[0], slug[1])
		logrus.Info("Started repository poller: ", config.Repository.Repo)

		go func(repo *Repository) {
			defer func() {
				logrus.Info("Halting repository poller: ", w)
				w.Stop()
			}()
			for {
				select {
				case changes := <-ch:
					for _, ref := range changes.NewRefs {
						if !strings.HasPrefix(*ref.Ref, "refs/heads/") {
							continue
						}
						refabbrev := strings.TrimPrefix(*ref.Ref, "refs/heads/")
						if err := config.Jenkins.BuildPipeline(repo.Job, 0, refabbrev); err != nil {
							logrus.Error("Failed to send Jenkins build request:", err)
						}
					}
					for _, ref := range changes.UpdatedRefs {
						if !strings.HasPrefix(*ref.Ref, "refs/heads/") {
							continue
						}
						refabbrev := strings.TrimPrefix(*ref.Ref, "refs/heads/")
						if err := config.Jenkins.BuildPipeline(repo.Job, 0, refabbrev); err != nil {
							logrus.Error("Failed to send Jenkins build request:", err)
						}
					}
				case err := <-errCh:
					logrus.Error("Repository poller failed:", err)
					return
				case <-stopCh:
					return
				}
			}
		}(config.Repository)
	}

	// create mux server
	mux := http.NewServeMux()

	// ping endpoint
	mux.HandleFunc("/ping", pingHandler)

	// jenkins notification endpoint
	mux.HandleFunc("/notification/jenkins", jenkinsHandler)

	// github webhooks endpoint
	mux.HandleFunc("/notification/github", githubHandler)

	// retry build endpoint
	mux.HandleFunc("/build/retry", customBuildHandler)

	// custom build endpoint
	mux.HandleFunc("/build/custom", customBuildHandler)

	// cron endpoint to reschedule bulk jobs
	mux.HandleFunc("/build/cron", cronBuildHandler)

	// set up the server
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		stopCh := make(chan os.Signal, 1)
		signal.Notify(stopCh, syscall.SIGTERM, syscall.SIGINT)

		<-stopCh
		server.Close()
	}()

	logrus.Printf("Starting server on port %q", port)
	if certFile != "" && keyFile != "" {
		logrus.Fatal(server.ListenAndServeTLS(certFile, keyFile))
	} else {
		logrus.Fatal(server.ListenAndServe())
	}
}
