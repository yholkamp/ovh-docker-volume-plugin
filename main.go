package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"flag"
	"os"
	"github.com/docker/go-plugins-helpers/volume"
)

const (
	VERSION = "0.10"
)

func main() {
	showVersion := flag.Bool("version", false, "Display version number of plugin and exit")
	cfgFile := flag.String("config", "/etc/ovh-docker-config.json", "path to config file")
	debug := flag.Bool("debug", true, "enable/disable debug logging")
	flag.Parse()

	if *debug == true {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	if *showVersion == true {
		fmt.Println("Version: ", VERSION)
		os.Exit(0)
	}

	log.Info("Starting ovh-docker-volume-plugin version: ", VERSION)
	d := New(*cfgFile)
	h := volume.NewHandler(d)
	log.Info(h.ServeUnix(d.Conf.SocketGroup, "ovh"))
}