package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/valarpirai/vardis/proto"
)

func main() {
	fmt.Printf("hello, world\n")
	arguments := os.Args
	var PORT string
	if len(arguments) > 1 {
		fmt.Println(arguments)
		PORT = ":" + arguments[1]
	} else {
		PORT = ":6379"
	}

	reader := bufio.NewReader(strings.NewReader("$6\r\nfoobar\r\n"))
	result, _ := proto.Decode(reader)
	fmt.Println(result)

	l, err := net.Listen("tcp4", PORT)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()
	// rand.Seed(time.Now().Unix())

	fmt.Printf("Started echo server on port: %s\n", PORT)
	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		go handleConnection(c)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	fmt.Printf("Serving %s\n", conn.RemoteAddr().String())
	for {
		// TODO - Read until Command End
		netData, err := proto.Decode(bufio.NewReader(conn))
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println(netData)

		if netData == "STOP" || netData == "QUIT" {
			return
		}

		result := "Received\n"
		conn.Write([]byte(result))
	}
}
