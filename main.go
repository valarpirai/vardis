package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
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

func handleConnection(c net.Conn) {
	defer c.Close()
	fmt.Printf("Serving %s\n", c.RemoteAddr().String())
	for {
		netData, err := bufio.NewReader(c).ReadString('\n')
		if err != nil {
			fmt.Println(err)
			return
		}

		temp := strings.TrimSpace(string(netData))
		if temp == "STOP" {
			break
		}

		fmt.Println(temp)

		result := temp + "\n"
		c.Write([]byte(string(result)))
	}
}
