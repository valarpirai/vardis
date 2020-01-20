package connection

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
	PORT        uint16
	cache       [MAX_DB_COUNT]*cache.CacheStorage
	persistance *cache.Persistance
	commandMap  map[string]*RedisCommand
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
	server.commandMap = PopulateCommandTable()
	return server
}

func (s *Server) Start() {
	l, err := net.Listen("tcp4", ":"+fmt.Sprint(s.PORT))
	if err != nil {
		log.Errorln(err)
		return
	}
	defer l.Close()

	log.Infof("Started vardis server on port: %d\n", s.PORT)
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
			// result := c_conn.cache.ProcessCommands(request)
			result := s.ProcessCommands(request, c_conn)
			c_conn.resultHandler(result)
		} else {
			c_conn.resultHandler(nil)
		}
	}
}

func (s *Server) ProcessCommands(req *proto.Request, conn *ClientConnection) (result interface{}) {
	log.Debug("Command Length: " + strconv.Itoa(req.CommandLength()))
	log.Debugf("COMMAND -> %#v", req.Command())

	redisCmd := s.commandMap[req.Command()]
	if redisCmd.Writable() == true {
		s.persistance.WriteCommand(req.String())
	}

	return redisCmd.Proc(req, conn)
}

func (s *ClientConnection) resultHandler(result interface{}) {
	// Handling string, int, array(set) and hash resposes

	log.Debugf("Result  -> %#v\n", result)
	// log.Debugf("Result Type  -> %T\n", result)
	switch result.(type) {
	case int:
		num_result := result.(int)
		s.conn.Write([]byte(proto.EncodeInt(int64(num_result))))
	case string:
		str_result := result.(string)
		s.conn.Write([]byte(proto.EncodeString(str_result)))
	case []string:
		aInterface := result.([]string)
		aString := make([][]byte, len(aInterface))
		for i, v := range aInterface {
			aString[i] = proto.EncodeBulkString(v)
		}
		s.conn.Write([]byte(proto.EncodeArray(aString)))
	default:
		// Write nil as response
		s.conn.Write([]byte(proto.EncodeNull()))
	}
}

func (server *Server) LoadFromDisk() {
	log.SetLevel(log.WarnLevel)
	defer log.SetLevel(log.DebugLevel)
	reader := bufio.NewReader(server.persistance.AofFile)
	cc := new(ClientConnection)
	cc.cache = server.cache[0]
	for {
		netData, _, err := proto.Decode(reader)
		if err != nil {
			log.Error(err)
			break
		}
		request := proto.ParseCommand(netData)
		server.ProcessCommands(request, cc)
	}
	log.Warn("Data Loaded successfully")
}
