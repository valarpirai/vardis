package main

import (
	"os"
	"strconv"

	log "github.com/Sirupsen/logrus"
	cache "github.com/valarpirai/vardis/cache"
	"github.com/valarpirai/vardis/connection"
)

type VardisApp struct {
	server *connection.Server
	PORT   uint16
}

func main() {
	confgureApp()

	log.Infoln("Starting Vardis server...")
	arguments := os.Args
	log.Debugln("Arguments...")
	log.Debugln(arguments)

	app := new(VardisApp)
	if len(arguments) > 1 {
		port, err := strconv.ParseUint(arguments[1], 10, 16)
		if nil == err {
			app.PORT = uint16(port)
		}
	}
	if 0 == app.PORT {
		app.PORT = 6379
	}

	NewApp(app)
}

func NewApp(app *VardisApp) {
	cacheStore := cache.NewCache()
	persistant := cache.NewStorage()

	log.Info("Loading data from disk")

	app.server = connection.NewServer(app.PORT, cacheStore, persistant)
	app.server.Start()

	go app.server.LoadFromDisk()
	cacheStore.Exists("Test")
}

func confgureApp() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}
