package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"

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
	conn   net.Conn
	cache  *cache.CacheStorage
	reader *bufio.Reader
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
		connection, err := l.Accept()
		if err != nil {
			log.Errorln(err)
			return
		}
		cc := new(ClientConnection)
		cc.conn = connection
		cc.cache = s.cache[0]
		cc.reader = bufio.NewReader(connection)
		go s.handleConnection(cc)
	}
}

func (s *Server) handleConnection(c_conn *ClientConnection) {
	conn := c_conn.conn
	defer conn.Close()
	log.Infof("Serving client: %s\n", conn.RemoteAddr().String())
	for {
		// Reading Commands and decoding
		netData, rawCmd, err := proto.Decode(c_conn.reader)
		if err != nil {
			log.Error(err)
			return
		}

		request := proto.ParseCommand(netData)
		request.SetRawCommand(rawCmd)
		log.Debugf("REQEUST -> %#v", request)

		log.Debugf("Full command %#v\n", netData)
		// log.Debugln(netData)

		if netData == "STOP" || netData == "QUIT" {
			return
		}
		log.Debugf("%T\n", netData)

		if !request.Error() {
			result := c_conn.processCommands(request)
			c_conn.resultHandler(result)
		} else {
			c_conn.resultHandler(nil)
		}
	}
}

// This method associated with Client connection
func (s *ClientConnection) processCommands(req proto.RequestInterface) (result interface{}) {
	log.Debug("Command Length: " + strconv.Itoa(req.CommandLength()))
	log.Debugf("COMMAND -> %#v", req.Command())
	switch req.Command() {
	case "PING":
		result = "PONG"
	case "SET":
		key, val := req.Key(), req.Value()
		result = s.cache.Set(key, val)
	case "GET":
		key := req.Key()
		res, ok := s.cache.Get(key)
		if ok {
			result = res
		}
	case "EXISTS":
		key := req.Key()
		result = s.cache.Exists(key)
	default:
		result = nil
	}
	return
}

func (s *ClientConnection) resultHandler(result interface{}) {
	// Handling string, int, array(set) and hash resposes

	log.Debugf("Result  -> %#v\n", result)
	switch result.(type) {
	case int:
		num_result := result.(int)
		s.conn.Write([]byte(proto.EncodeInt(int64(num_result))))
	case string:
		str_result := result.(string)
		s.conn.Write([]byte(proto.EncodeString(str_result)))
	default:
		// Write nil as response
		s.conn.Write([]byte(proto.EncodeNull()))
	}
}
