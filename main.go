package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	cache "github.com/valarpirai/vardis/cache"
	"github.com/valarpirai/vardis/proto"
)

type App struct {
	server Server
}

var PORT uint16

func main() {
	confgureApp()

	log.Info("Starting Vardis server...\n")
	arguments := os.Args
	log.Debug("Arguments...")
	log.Debug(arguments)
	if len(arguments) > 1 {
		port, err := strconv.ParseUint(arguments[1], 10, 16)
		if nil == err {
			PORT = uint16(port)
		}
	}
	if 0 == PORT {
		PORT = 6379
	}

	reader := bufio.NewReader(strings.NewReader("$6\r\nfoobar\r\n"))
	result, _, _ := proto.Decode(reader)
	log.Debug(result)

	NewApp()
}

func NewApp() {
	cache := cache.NewCache()
	cache.Exists("Test")

	server := NewServer(PORT)
	server.cache[0] = cache
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
