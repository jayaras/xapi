package main

import (
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/jayaras/xapi"
	"gopkg.in/yaml.v2"
)

type timeline struct {
	Title   string        `yaml:"title,omitempty"`
	Message string        `yaml:"message,omitempty"`
	Pause   time.Duration `yaml:"pause,omitempty"`
}

type config struct {
	URL         string        `yaml:"url,omitempty"`
	User        string        `yaml:"user,omitempty"`
	Password    string        `yaml:"password,omitempty"`
	Insecure    bool          `yaml:"insecure,omitempty"`
	Reconnect   bool          `yaml:"reconnect,omitempty"`
	TimeLine    []timeline    `yaml:"timeLine,omitempty"`
	DisplayTime time.Duration `yaml:"displayTime"`
}

func main() {
	data, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Printf("could not read config file: %v", err)
		os.Exit(1)
	}

	cfg := config{}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Printf("could not parse config file: %v", err)
		os.Exit(1)
	}

	client := &xapi.Client{
		URL:      cfg.URL,
		User:     cfg.User,
		Password: cfg.Password,
		Insecure: cfg.Insecure,
	}

	client.OnConnectFunc = func(c *xapi.Client) {
		log.Printf("connected to: %s", cfg.URL)

		for _, v := range cfg.TimeLine {
			log.Printf("sleeping for: %v", v.Pause)
			time.Sleep(v.Pause)
			log.Printf("Sending Message: %v", v.Message)

			if err := c.Alert(v.Title, v.Message, cfg.DisplayTime); err != nil {
				log.Printf("alert error: %v", err)
			}
		}

		os.Exit(0)
	}

	if err := client.ConnectAndRun(); err != nil {
		log.Printf("connect error: %v", err)
		os.Exit(1)
	}
}
