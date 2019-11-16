package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
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

		fmt.Printf("%#v\n", netData)
		// fmt.Println(netData)

		if netData == "STOP" || netData == "QUIT" {
			return
		}

		params, ok := netData.([]interface{})
		// aString, ok := netData.([]string)
		// var aString []string
		var result string
		if ok {
			aString := make([]string, len(params))
			for i, v := range params {
				aString[i] = v.(string)
			}
			fmt.Println("Command Length: " + strconv.Itoa(len(aString)))

			result = processCommands(aString)
		} else {
			if params, ok := netData.(string); ok {
				result = processCommand(params)
			} else {
				result = ""
			}
		}
		if 0 == len(result) {
			conn.Write([]byte(proto.EncodeNull()))
		} else {
			conn.Write([]byte(proto.EncodeString(result)))
		}
	}
}

func processCommands(cmd []string) string {
	if "PING" == cmd[0] {
		return "+PONG"
	}
	return ""
}

func processCommand(cmd string) string {
	if "PING" == cmd {
		return "PONG"
	}
	return "Test"
}
