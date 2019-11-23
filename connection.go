package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/valarpirai/vardis/cache"
	"github.com/valarpirai/vardis/proto"
)

const MAX_DB_COUNT = 15

type Server struct {
	PORT  uint16
	cache [MAX_DB_COUNT]*cache.CacheStorage
}

type ClientConnection struct {
	conn  net.Conn
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
		log.Errorln(err)
		return
	}
	defer l.Close()

	log.Infof("Started echo server on port: %d\n", s.PORT)
	for {
		c, err := l.Accept()
		if err != nil {
			log.Errorln(err)
			return
		}
		cc := new(ClientConnection)
		cc.conn = c
		cc.cache = s.cache[0]
		go s.handleConnection(cc)
	}
}

func (s *Server) handleConnection(c_conn *ClientConnection) {
	conn := c_conn.conn
	defer conn.Close()
	log.Infof("Serving client: %s\n", conn.RemoteAddr().String())
	reader := bufio.NewReader(conn)
	for {
		// Reading Commands and decoding
		netData, err := proto.Decode(reader)
		if err != nil {
			log.Error(err)
			return
		}

		log.Debugf("%#v\n", netData)
		// log.Debugln(netData)

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
			log.Debug("Command Length: " + strconv.Itoa(len(aString)))

			result = c_conn.processCommands(aString)
		} else {
			// Parsing simple commands
			if params, ok := netData.(string); ok {
				result = c_conn.processCommand(params)
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
func (s *ClientConnection) processCommands(args []string) (result interface{}) {
	cmd := strings.ToUpper(args[0])
	log.Debugf("%#v", cmd)
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

func (s *ClientConnection) processCommand(cmd string) string {
	if "PING" == cmd {
		return "PONG"
	}
	return "commandProcessor(cmd)"
}

func commandProcessor() string {
	return "SUCCESS"
}
