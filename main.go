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

// type App struct {
// 	server Server
// }

func main() {
	fmt.Printf("Starting Vardis server...\n")
	arguments := os.Args
	fmt.Println("Arguments...")
	fmt.Println(arguments)
	var PORT uint16
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

	server := NewServer(PORT)
	server.Start()
}

func New() {
	cache := cache.NewCache()
	cache.Exists("Test")
}

func CheckErrorAndReturnDefault(default_value interface{}, err error) interface{} {
	if nil == err {
		return default_value
	}
	return nil
}
