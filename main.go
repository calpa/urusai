package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	"github.com/calpa/urusai/config"
	"github.com/calpa/urusai/crawler"
)

func main() {
	var configFile string
	var logLevel string
	var timeout int

	flag.StringVar(&configFile, "config", "", "path to config file")
	flag.StringVar(&logLevel, "log", "info", "logging level (debug, info, warn, error)")
	flag.IntVar(&timeout, "timeout", 0, "for how long the crawler should be running, in seconds (0 means no timeout)")
	flag.Parse()

	setLogLevel(logLevel)

	var cfg *config.Config
	var err error

	if configFile == "" {
		log.Println("No config file specified, using default configuration")
		cfg, err = config.LoadDefaultConfig()
		if err != nil {
			log.Fatalf("Failed to load default config: %v", err)
		}
	} else {
		cfg, err = config.LoadFromFile(configFile)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
	}

	if timeout > 0 {
		cfg.Timeout = timeout
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	c := crawler.NewCrawler(cfg)

	log.Println("Starting urusai - HTTP/DNS traffic noise generator")
	c.Crawl(ctx)
}

func setLogLevel(level string) {
	switch level {
	case "debug":
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
		log.SetPrefix("DEBUG: ")
	case "info":
		log.SetFlags(log.Ldate | log.Ltime)
		log.SetPrefix("INFO: ")
	case "warn":
		log.SetFlags(log.Ldate | log.Ltime)
		log.SetPrefix("WARNING: ")
	case "error":
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
		log.SetPrefix("ERROR: ")
	default:
		log.SetFlags(log.Ldate | log.Ltime)
		log.SetPrefix("INFO: ")
	}
}
