package connection

import (
	"bufio"
	"fmt"
	"net"

	log "github.com/Sirupsen/logrus"
	"github.com/valarpirai/vardis/cache"
	"github.com/valarpirai/vardis/proto"
)

const MAX_DB_COUNT = 15

type Server struct {
	PORT        uint16
	cache       [MAX_DB_COUNT]*cache.CacheStorage
	persistance *cache.Persistance
}

type ClientConnection struct {
	conn    net.Conn
	cache   *cache.CacheStorage
	reader  *bufio.Reader
	storage *cache.Persistance
}

// NewServer
func NewServer(port uint16, cacheStore *cache.CacheStorage, persistant *cache.Persistance) *Server {
	server := new(Server)
	server.PORT = port
	server.cache[0] = cacheStore
	server.persistance = persistant
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
		go s.handleConnection(cc, s.persistance)
	}
}

func (s *Server) handleConnection(c_conn *ClientConnection, persistance *cache.Persistance) {
	conn := c_conn.conn
	defer conn.Close()
	log.Infof("Serving client: %s\n", conn.RemoteAddr().String())
	for {
		// Reading Commands and decoding
		// Parse RAW commands properly
		netData, rawCmd, err := proto.Decode(c_conn.reader)
		if err != nil {
			log.Error(err)
			return
		}

		request := proto.ParseCommand(netData)
		request.SetRawCommand(rawCmd)
		log.Debugf("REQEUST -> %#v\n", request)
		// log.Debugf("RAW -> %#v", rawCmd)

		log.Debugf("Full command %#v\n", netData)
		// log.Debugln(netData)
		// log.Debugf("%T\n", netData)

		if netData == "STOP" || netData == "QUIT" {
			return
		}

		if !request.Error() {
			result := c_conn.cache.ProcessCommands(request)
			c_conn.resultHandler(result)
			persistance.WriteCommand(request.String())
		} else {
			c_conn.resultHandler(nil)
		}
	}
}

// This method associated with Client connection
// func (s *ClientConnection) processCommands(req proto.RequestInterface) (result interface{}) {
// 	return s.cache.ProcessCommands(req)
// }

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
