package connection

import (
	"time"

	"github.com/valarpirai/vardis/cache"
	"github.com/valarpirai/vardis/proto"
)

// "\r\n"
// "+OK\r\n"
// "-ERR\r\n"
// "$0\r\n\r\n"
// ":0\r\n"
// ":1\r\n"
// "*0\r\n"
// "+PONG\r\n"
// "+QUEUED\r\n"
// "*2\r\n$1\r\n0\r\n*0\r\n"
// "-WRONGTYPE Operation against a key holding the wrong kind of value\r\n"
// "-ERR no such key\r\n"
// "-ERR syntax error\r\n"
// "-ERR source and destination objects are the same\r\n"
// "-ERR index out of range\r\n"
// "-NOSCRIPT No matching script. Please use EVAL.\r\n"
// "-LOADING Redis is loading the dataset in memory\r\n"
// "-BUSY Redis is busy running a script. You can only call SCRIPT KILL or SHUTDOWN NOSAVE.\r\n"
// "-MASTERDOWN Link with MASTER is down and replica-serve-stale-data is set to 'no'.\r\n"
// "-MISCONF Redis is configured to save RDB snapshots, but it is currently not able to persist on disk. Commands that may modify the data set are disabled, because this instance is configured to report errors during writes if RDB snapshotting fails (stop-writes-on-bgsave-error option). Please check the Redis logs for details about the RDB error.\r\n"
// "-READONLY You can't write against a read only replica.\r\n"
// "-NOAUTH Authentication required.\r\n"
// "-OOM command not allowed when used memory > 'maxmemory'.\r\n"
// "-EXECABORT Transaction discarded because of previous errors.\r\n"
// "-NOREPLICAS Not enough good replicas to write.\r\n"
// "-BUSYKEY Target key name already exists.\r\n"

func genericGet(key string, conn *ClientConnection) {
	// Check key exists
	// Expire if needed, don't delete key and return nil
	data := expireIfNeeded(key, conn.cache)
	if data == nil {
		addReply(conn, nil)
	}

	addReply(conn, data)
	// If string or other object
	// To avoid additional memory allocation
	// Don't construct full response,
	// just write the response bytes to Connection

	// Reduce memory allocation and GC as much as possible
}

func expireIfNeeded(key string, db cache.ICacheStorage) *cache.CacheData {
	store := db.Store()
	if data, ok := store[key]; ok {
		if 0 == data.Expires() || time.Now().UnixNano() < data.Expires() {
			return data
		}
	}
	return nil
}

func addReply(c *ClientConnection, reply *cache.CacheData) {
	if nil == c.cconn {
		return
	}
	if nil != reply {
		switch reply.Type() {
		case cache.OBJ_STRING:
			str_result := reply.Value().(string)
			WriteStringReply(c, str_result)
		case cache.OBJ_LIST:
			aInterface := reply.Value().([]string)
			aString := make([][]byte, len(aInterface))
			for i, v := range aInterface {
				aString[i] = proto.EncodeBulkString(v)
			}
			c.cconn.Write(proto.EncodeArray(aString))
		case cache.OBJ_SET:
		case cache.OBJ_ZSET:
		case cache.OBJ_HASH:
		}
	} else {
		c.cconn.Write(proto.EncodeNull())
	}
}

func WriteStringReply(c *ClientConnection, msg string) {
	c.cconn.Write(proto.EncodeString(msg))
}

func genericeSet(key string, value string) {
	// set expiry in SECONDS, MILLI-SECONDS

}

// String command implementation
func getCommand(req *proto.Request, conn *ClientConnection) {
	genericGet(req.Key(), conn)
}

/* SET key value [NX] [XX] [EX <seconds>] [PX <milliseconds>] */
func setCommand(req *proto.Request, conn *ClientConnection) {
	key, val := req.Key(), req.Value()
	conn.cache.Set(key, val)
}
func delCommand(req *proto.Request, conn *ClientConnection) {
}
func keysCommand(req *proto.Request, conn *ClientConnection) {
	response := new(cache.CacheData)
	response.SetValue(conn.cache.Keys(req.Key()))
	response.SetType(cache.OBJ_LIST)
	addReply(conn, response)
}
func existsCommand(req *proto.Request, conn *ClientConnection) {
	key := req.Key()
	conn.cache.Exists(key)
}
func pingCommand(req *proto.Request, conn *ClientConnection) {
	WriteStringReply(conn, "PONG")
}
