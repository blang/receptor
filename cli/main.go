package main

import (
	"flag"
	"fmt"
	"github.com/blang/receptor"
	_ "github.com/blang/receptor/plugins"
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

	cfg, err := receptor.NewConfigFromFile(*cfgFile)
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
	for name, _ := range receptor.Watchers {
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}

func availableReactors() string {
	var names []string
	for name, _ := range receptor.Reactors {
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}
