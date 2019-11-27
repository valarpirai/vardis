package main

import (
	"os"
	"strconv"

	log "github.com/Sirupsen/logrus"
	cache "github.com/valarpirai/vardis/cache"
	"github.com/valarpirai/vardis/connection"
)

type App struct {
	server connection.Server
}

var PORT uint16

func main() {
	confgureApp()

	log.Infoln("Starting Vardis server...")
	arguments := os.Args
	log.Debugln("Arguments...")
	log.Debugln(arguments)
	if len(arguments) > 1 {
		port, err := strconv.ParseUint(arguments[1], 10, 16)
		if nil == err {
			PORT = uint16(port)
		}
	}
	if 0 == PORT {
		PORT = 6379
	}

	NewApp()
}

func NewApp() {
	cacheStore := cache.NewCache()
	persistant := cache.NewStorage()
	cacheStore.LoadFromDisk(persistant)
	cacheStore.Exists("Test")

	server := connection.NewServer(PORT, cacheStore, persistant)
	server.Start()
}

func confgureApp() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

func CheckErrorAndReturnDefault(default_value interface{}, err error) interface{} {
	if nil == err {
		return default_value
	}
	return nil
}
