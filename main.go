package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	cache "github.com/valarpirai/vardis/cache"
	"github.com/valarpirai/vardis/proto"
)

type App struct {
	server Server
}

var PORT uint16

func main() {
	fmt.Printf("Starting Vardis server...\n")
	arguments := os.Args
	fmt.Println("Arguments...")
	fmt.Println(arguments)
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
	result, _ := proto.Decode(reader)
	fmt.Println(result)

	NewApp()
}

func NewApp() {
	cache := cache.NewCache()
	cache.Exists("Test")

	server := NewServer(PORT)
	server.cache = cache

	server.Start()
}

func CheckErrorAndReturnDefault(default_value interface{}, err error) interface{} {
	if nil == err {
		return default_value
	}
	return nil
}
