package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/calpa/urusai/config"
	"github.com/calpa/urusai/crawler"
)

// Version is set via ldflags at build time (e.g., -X main.Version=v1.2.0).
var Version = "dev"

func main() {
	var configFile string
	var logLevel string
	var showVersion bool
	var timeout int

	flag.StringVar(&configFile, "config", "", "path to config file")
	flag.StringVar(&logLevel, "log", "info", "logging level (debug, info, warn, error)")
	flag.IntVar(&timeout, "timeout", -1, "for how long the crawler should be running, in seconds (-1 means use config, 0 means no timeout)")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.Parse()

	if showVersion {
		fmt.Printf("urusai %s\n", Version)
		os.Exit(0)
	}

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

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	// Override timeout if explicitly set via flag
	if timeout >= 0 {
		cfg.Timeout = timeout
	}

	c := crawler.NewCrawler(cfg)

	// Handle graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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
