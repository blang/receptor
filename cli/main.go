package main

import (
	"flag"
	"fmt"
	"github.com/blang/receptor"
	"github.com/blang/receptor/plugin"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfgFile := flag.String("config", "./receptor.conf.json", "Path to config file")
	pluginPath := flag.String("plugins", "./plugins", "Path to plugins")
	flag.Parse()

	cfg, err := receptor.NewConfigFromFile(*cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while reading config file: %s\n", err)
		os.Exit(1)
	}
	lookupService := plugin.NewLookup(*pluginPath)
	r := receptor.NewReceptor(lookupService)
	err = r.Init(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while setup services: %s\n", err)
		os.Exit(1)
	}

	log.Println("Starting services")
	r.Start()
	log.Println("Services running")

	// Manage clean shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-c:
		fmt.Println("Graceful shutdown initiated")

		r.Stop()
		fmt.Println("Graceful shutdown complete")
	case <-r.FailureCh:
		fmt.Println("At least one service failed, shutdown")
		r.Stop()
		fmt.Println("Shutdown complete")
		os.Exit(2)
	}
}
