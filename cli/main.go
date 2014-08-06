package main

import (
	"flag"
	"fmt"
	"github.com/blang/receptor"
	"github.com/blang/receptor/config"
	"github.com/blang/receptor/discovery"
	_ "github.com/blang/receptor/plugins"
	"github.com/blang/receptor/reactor"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	cfgFile := flag.String("config", "./receptor.conf.json", "Path to config file")
	flag.Parse()

	log.Printf("Available watchers: [%s]\n", availableWatchers())
	log.Printf("Available reactors: [%s]\n", availableReactors())

	cfg, err := config.ReadFromFile(*cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while reading config file: %s\n", err)
		os.Exit(1)
	}

	r := receptor.NewReceptor()
	err = r.Init(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while setup services: %s\n", err)
	}

	log.Println("Starting services")
	r.Start()
	log.Println("Services running")

	// Manage clean shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGINT, syscall.SIGTERM)
	<-c
	fmt.Println("Shutdown initiated")

	r.Stop()
	fmt.Println("Shutdown complete")

}

func availableWatchers() string {
	var names []string
	for name, _ := range discovery.Watchers {
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}

func availableReactors() string {
	var names []string
	for name, _ := range reactor.Reactors {
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}
