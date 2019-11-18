package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/valarpirai/vardis/cache"
	"github.com/valarpirai/vardis/proto"
)

type Server struct {
	PORT  uint16
	cache *cache.CacheStorage
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
		var result interface{}
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
		str_result, ok := result.(string)
		var nil_resp bool
		if ok && 0 < len(str_result) {
			conn.Write([]byte(proto.EncodeString(str_result)))
			nil_resp = false
		} else {
			num_result, ok := result.(int)
			if ok {
				conn.Write([]byte(proto.EncodeInt(int64(num_result))))
				nil_resp = false
			}
		}

		if nil_resp {
			conn.Write([]byte(proto.EncodeNull()))
		}
	}
}

// This method associated with Client connection
func (s *Server) processCommands(args []string) (result interface{}) {
	cmd := strings.ToUpper(args[0])
	fmt.Printf("%#v", cmd)
	switch cmd {
	case "PING":
		result = "PONG"
	case "SET":
		key, val := args[1], args[2]
		result = s.cache.Set(key, val)
	case "GET":
		key := args[1]
		res, ok := s.cache.Get(key)
		if ok {
			result = res
		}
	case "EXISTS":
		key := args[1]
		result = s.cache.Exists(key)
	default:
		result = "Command Not Found"
	}
	return
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
