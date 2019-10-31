package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/valarpirai/vardis/resp"
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
	result, _ := resp.Decode(reader)
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

func handleConnection(c net.Conn) {
	defer c.Close()
	fmt.Printf("Serving %s\n", c.RemoteAddr().String())
	for {
		// TODO - Read until EOF
		netData, err := bufio.NewReader(c).ReadString('\n')
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("%#v \n", netData)

		temp := strings.TrimSpace(string(netData))
		if temp == "STOP" {
			break
		}

		result := temp + "\n"
		c.Write([]byte(string(result)))
	}
}
