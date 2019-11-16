package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"

	"github.com/valarpirai/vardis/proto"
)

type Server struct {
	PORT uint16
}

// NewServer
func NewServer(port uint16) *Server {
	server := new(Server)
	server.PORT = port
	return server
}

func (s *Server) Start() {
	l, err := net.Listen("tcp4", ":"+fmt.Sprint(s.PORT))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()

	fmt.Printf("Started echo server on port: %d\n", s.PORT)
	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		go s.handleConnection(c)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	fmt.Printf("Serving %s\n", conn.RemoteAddr().String())
	reader := bufio.NewReader(conn)
	for {
		// Reading Commands and decoding
		netData, err := proto.Decode(reader)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("%#v\n", netData)
		// fmt.Println(netData)

		if netData == "STOP" || netData == "QUIT" {
			return
		}

		// Parsing Array of commands
		params, ok := netData.([]interface{})
		var result string
		if ok {
			aString := make([]string, len(params))
			for i, v := range params {
				aString[i] = v.(string)
			}
			fmt.Println("Command Length: " + strconv.Itoa(len(aString)))

			result = s.processCommands(aString)
		} else {
			// Parsing simple commands
			if params, ok := netData.(string); ok {
				result = s.processCommand(params)
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

func (s *Server) processCommands(cmd []string) string {
	if "PING" == cmd[0] {
		return "+PONG"
	}
	return ""
}

func (s *Server) processCommand(cmd string) string {
	if "PING" == cmd {
		return "PONG"
	}
	return "commandProcessor(cmd)"
}

func commandProcessor() string {
	return "SUCCESS"
}
