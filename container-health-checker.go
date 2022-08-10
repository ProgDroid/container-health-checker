package main

import (
	"fmt"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/BurntSushi/toml"
)

const CONFIG_FILE string = "config.toml"
const EXIT_CONFIG int = 78

type (
	Config struct {
		Interval   uint8
		Timeout    uint8
		Retries    uint8
		Containers map[string]Container
	}

	Container struct {
		Protocol string
		Host     string
		Port     string
		healthy  bool
	}

	Result struct {
		key     string
		healthy bool
	}
)

func checkStatus(client *http.Client, c chan Result, key string, container Container, interval uint8, retries uint8) {
	var res Result = Result{key, false}

	var url string = container.Protocol + "://" + container.Host

	if container.Port != "" {
		url += ":" + container.Port
	}

	var i uint8 = 0

	for i < retries {
		i++

		resp, err := client.Get(url)
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode <= 299 {
			res.healthy = true
			break
		}

		time.Sleep(time.Duration(interval) * time.Second)
	}

	c <- res
}

// TODO run as docker container

func main() {
	if _, err := os.Stat(CONFIG_FILE); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var cfg Config
	meta, err := toml.DecodeFile(CONFIG_FILE, &cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if !meta.IsDefined("containers") {
		fmt.Fprintln(os.Stderr, "No containers defined")
		os.Exit(EXIT_CONFIG)
	}

	if len(meta.Undecoded()) > 0 {
		fmt.Fprintf(os.Stderr, "Undecoded fields found in config: %q\n", meta.Undecoded())
		os.Exit(EXIT_CONFIG)
	}

	tr := &http.Transport{
		IdleConnTimeout: time.Duration(cfg.Timeout) * time.Second,
	}
	client := &http.Client{Transport: tr}

	c := make(chan Result)
	defer close(c)

	for key, container := range cfg.Containers {
		go checkStatus(client, c, key, container, cfg.Interval, cfg.Retries)
	}

	var containersUpdated int = 0

	for {
		select {
		case value := <-c:
			var containerData = cfg.Containers[value.key]
			containerData.healthy = value.healthy
			cfg.Containers[value.key] = containerData

			containersUpdated++
		default:
			continue
		}

		if containersUpdated >= len(cfg.Containers) {
			break
		}
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 4, 1, '\t', tabwriter.AlignRight)

	for k, cont := range cfg.Containers {
		var status string = "UNHEALTHY"

		if cont.healthy {
			status = "OK"
		}

		fmt.Fprintf(writer, "%s:\t%s\n", k, status)
	}

	writer.Flush()
}
