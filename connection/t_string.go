package connection

import (
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

func genericGet(key string) {
	// Check key exists
	//
	// Expire if needed, don't delete key and return nil
	// If string or other object

	// To avoid additional memory allocation
	// Don't construct full response,
	// just write the response bytes to Connection

	// Reduce memory allocation and GC as much as possible
}

func genericeSet(key string, value string) {
	// set expiry in SECONDS, MILLI-SECONDS

}

// String command implementation
func getCommand(req *proto.Request, conn *ClientConnection) (result interface{}) {
	res, ok := conn.cache.Get(req.Key())
	if ok {
		result = res
	}
	return result
}

/* SET key value [NX] [XX] [EX <seconds>] [PX <milliseconds>] */
func setCommand(req *proto.Request, conn *ClientConnection) (result interface{}) {
	key, val := req.Key(), req.Value()
	result = conn.cache.Set(key, val)
	return result
}
func delCommand(req *proto.Request, conn *ClientConnection) (result interface{}) {
	return
}
func keysCommand(req *proto.Request, conn *ClientConnection) (result interface{}) {
	result = conn.cache.Keys(req.Key())
	return
}
func existsCommand(req *proto.Request, conn *ClientConnection) (result interface{}) {
	key := req.Key()
	result = conn.cache.Exists(key)
	return
}
func pingCommand(req *proto.Request, conn *ClientConnection) (result interface{}) {
	return "PONG"
}
