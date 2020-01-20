package connection

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/valarpirai/vardis/proto"
)

// Command spec

/* Our command table. - From redis source
 *
 * Every entry is composed of the following fields:
 *
 * name:        A string representing the command name.
 *
 * function:    Pointer to the C function implementing the command.
 *
 * arity:       Number of arguments, it is possible to use -N to say >= N
 *
 * sflags:      Command flags as string. See below for a table of flags.
 *
 * flags:       Flags as bitmask. Computed by Redis using the 'sflags' field.
 *
 * get_keys_proc: An optional function to get key arguments from a command.
 *                This is only used when the following three fields are not
 *                enough to specify what arguments are keys.
 *
 * first_key_index: First argument that is a key
 *
 * last_key_index: Last argument that is a key
 *
 * key_step:    Step to get all the keys from first to last argument.
 *              For instance in MSET the step is two since arguments
 *              are key,val,key,val,...
 *
 * microseconds: Microseconds of total execution time for this command.
 *
 * calls:       Total number of calls of this command.
 *
 * id:          Command bit identifier for ACLs or other goals.
 *
 * The flags, microseconds and calls fields are computed by Redis and should
 * always be set to zero.
 *
 * Command flags are expressed using space separated strings, that are turned
 * into actual flags by the populateCommandTable() function.
 *
 * This is the meaning of the flags:
 *
 * write:       Write command (may modify the key space).
 *
 * read-only:   All the non special commands just reading from keys without
 *              changing the content, or returning other informations like
 *              the TIME command. Special commands such administrative commands
 *              or transaction related commands (multi, exec, discard, ...)
 *              are not flagged as read-only commands, since they affect the
 *              server or the connection in other ways.
 *
 * use-memory:  May increase memory usage once called. Don't allow if out
 *              of memory.
 *
 * admin:       Administrative command, like SAVE or SHUTDOWN.
 *
 * pub-sub:     Pub/Sub related command.
 *
 * no-script:   Command not allowed in scripts.
 *
 * random:      Random command. Command is not deterministic, that is, the same
 *              command with the same arguments, with the same key space, may
 *              have different results. For instance SPOP and RANDOMKEY are
 *              two random commands.
 *
 * to-sort:     Sort command output array if called from script, so that the
 *              output is deterministic. When this flag is used (not always
 *              possible), then the "random" flag is not needed.
 *
 * ok-loading:  Allow the command while loading the database.
 *
 * ok-stale:    Allow the command while a slave has stale data but is not
 *              allowed to serve this data. Normally no command is accepted
 *              in this condition but just a few.
 *
 * no-monitor:  Do not automatically propagate the command on MONITOR.
 *
 * no-slowlog:  Do not automatically propagate the command to the slowlog.
 *
 * cluster-asking: Perform an implicit ASKING for this command, so the
 *              command will be accepted in cluster mode if the slot is marked
 *              as 'importing'.
 *
 * fast:        Fast command: O(1) or O(log(N)) command that should never
 *              delay its execution as long as the kernel scheduler is giving
 *              us time. Note that commands that may trigger a DEL as a side
 *              effect (like SET) are not fast commands.
 *
 * The following additional flags are only used in order to put commands
 * in a specific ACL category. Commands can have multiple ACL categories.
 *
 * @keyspace, @read, @write, @set, @sortedset, @list, @hash, @string, @bitmap,
 * @hyperloglog, @stream, @admin, @fast, @slow, @pubsub, @blocking, @dangerous,
 * @connection, @transaction, @scripting, @geo.
 *
 * Note that:
 *
 * 1) The read-only flag implies the @read ACL category.
 * 2) The write flag implies the @write ACL category.
 * 3) The fast flag implies the @fast ACL category.
 * 4) The admin flag implies the @admin and @dangerous ACL category.
 * 5) The pub-sub flag implies the @pubsub ACL category.
 * 6) The lack of fast flag implies the @slow ACL category.
 * 7) The non obvious "keyspace" category includes the commands
 *    that interact with keys without having anything to do with
 *    specific data structures, such as: DEL, RENAME, MOVE, SELECT,
 *    TYPE, EXPIRE*, PEXPIRE*, TTL, PTTL, ...
 */
type RedisCommand struct {
	name   string
	Proc   redisCommandProc // function pointer
	arity  int
	sflags string /* Flags as string representation, one char per flag. */
	flags  uint64 /* The actual flags, obtained from the 'sflags' field. */
	/* Use a function to determine keys arguments in a command line.
	 * Used for Redis Cluster redirect. */
	getkeys_proc redisGetKeysProc
	/* What keys should be loaded in background when calling this command? */
	firstkey     int /* The first argument that's a key (0 = no keys) */
	lastkey      int /* The last argument that's a key */
	keystep      int /* The step between first and last key */
	microseconds int
	calls        int64
	id           int /* Command ID. This is a progressive ID starting from 0 that
	   is assigned at runtime, and is used in order to check
	   ACLs. A connection is able to execute a given command if
	   the user associated to the connection has this command
	   bit set in the bitmap of allowed commands. */
}

type redisCommandProc func(req *proto.Request, conn *ClientConnection) (result interface{})
type redisGetKeysProc func(cmd *RedisCommand, argc int, numkeys int)

var redisCommandTable = []*RedisCommand{
	// {"module", moduleCommand, -2,
	// 	"admin no-script",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	{"get", getCommand, 2,
		"read-only fast @string",
		0, nil, 1, 1, 1, 0, 0, 0},

	/* Note that we can't flag set as fast, since it may perform an
	 * implicit DEL of a large key. */
	{"set", setCommand, -3,
		"write use-memory @string",
		0, nil, 1, 1, 1, 0, 0, 0},

	// {"setnx", setnxCommand, 3,
	// 	"write use-memory fast @string",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"setex", setexCommand, 4,
	// 	"write use-memory @string",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"psetex", psetexCommand, 4,
	// 	"write use-memory @string",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"append", appendCommand, 3,
	// 	"write use-memory fast @string",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"strlen", strlenCommand, 2,
	// 	"read-only fast @string",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	{"del", delCommand, -2,
		"write @keyspace",
		0, nil, 1, -1, 1, 0, 0, 0},

	// {"unlink", unlinkCommand, -2,
	// 	"write fast @keyspace",
	// 	0, nil, 1, -1, 1, 0, 0, 0},

	{"exists", existsCommand, -2,
		"read-only fast @keyspace",
		0, nil, 1, -1, 1, 0, 0, 0},

	// {"setbit", setbitCommand, 4,
	// 	"write use-memory @bitmap",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"getbit", getbitCommand, 3,
	// 	"read-only fast @bitmap",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"bitfield", bitfieldCommand, -2,
	// 	"write use-memory @bitmap",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"setrange", setrangeCommand, 4,
	// 	"write use-memory @string",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"getrange", getrangeCommand, 4,
	// 	"read-only @string",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"substr", getrangeCommand, 4,
	// 	"read-only @string",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"incr", incrCommand, 2,
	// 	"write use-memory fast @string",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"decr", decrCommand, 2,
	// 	"write use-memory fast @string",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"mget", mgetCommand, -2,
	// 	"read-only fast @string",
	// 	0, nil, 1, -1, 1, 0, 0, 0},

	// {"rpush", rpushCommand, -3,
	// 	"write use-memory fast @list",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"lpush", lpushCommand, -3,
	// 	"write use-memory fast @list",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"rpushx", rpushxCommand, -3,
	// 	"write use-memory fast @list",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"lpushx", lpushxCommand, -3,
	// 	"write use-memory fast @list",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"linsert", linsertCommand, 5,
	// 	"write use-memory @list",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"rpop", rpopCommand, 2,
	// 	"write fast @list",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"lpop", lpopCommand, 2,
	// 	"write fast @list",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"brpop", brpopCommand, -3,
	// 	"write no-script @list @blocking",
	// 	0, nil, 1, -2, 1, 0, 0, 0},

	// {"brpoplpush", brpoplpushCommand, 4,
	// 	"write use-memory no-script @list @blocking",
	// 	0, nil, 1, 2, 1, 0, 0, 0},

	// {"blpop", blpopCommand, -3,
	// 	"write no-script @list @blocking",
	// 	0, nil, 1, -2, 1, 0, 0, 0},

	// {"llen", llenCommand, 2,
	// 	"read-only fast @list",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"lindex", lindexCommand, 3,
	// 	"read-only @list",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"lset", lsetCommand, 4,
	// 	"write use-memory @list",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"lrange", lrangeCommand, 4,
	// 	"read-only @list",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"ltrim", ltrimCommand, 4,
	// 	"write @list",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"lrem", lremCommand, 4,
	// 	"write @list",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"rpoplpush", rpoplpushCommand, 3,
	// 	"write use-memory @list",
	// 	0, nil, 1, 2, 1, 0, 0, 0},

	// {"sadd", saddCommand, -3,
	// 	"write use-memory fast @set",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"srem", sremCommand, -3,
	// 	"write fast @set",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"smove", smoveCommand, 4,
	// 	"write fast @set",
	// 	0, nil, 1, 2, 1, 0, 0, 0},

	// {"sismember", sismemberCommand, 3,
	// 	"read-only fast @set",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"scard", scardCommand, 2,
	// 	"read-only fast @set",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"spop", spopCommand, -2,
	// 	"write random fast @set",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"srandmember", srandmemberCommand, -2,
	// 	"read-only random @set",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"sinter", sinterCommand, -2,
	// 	"read-only to-sort @set",
	// 	0, nil, 1, -1, 1, 0, 0, 0},

	// {"sinterstore", sinterstoreCommand, -3,
	// 	"write use-memory @set",
	// 	0, nil, 1, -1, 1, 0, 0, 0},

	// {"sunion", sunionCommand, -2,
	// 	"read-only to-sort @set",
	// 	0, nil, 1, -1, 1, 0, 0, 0},

	// {"sunionstore", sunionstoreCommand, -3,
	// 	"write use-memory @set",
	// 	0, nil, 1, -1, 1, 0, 0, 0},

	// {"sdiff", sdiffCommand, -2,
	// 	"read-only to-sort @set",
	// 	0, nil, 1, -1, 1, 0, 0, 0},

	// {"sdiffstore", sdiffstoreCommand, -3,
	// 	"write use-memory @set",
	// 	0, nil, 1, -1, 1, 0, 0, 0},

	// {"smembers", sinterCommand, 2,
	// 	"read-only to-sort @set",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"sscan", sscanCommand, -3,
	// 	"read-only random @set",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"zadd", zaddCommand, -4,
	// 	"write use-memory fast @sortedset",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"zincrby", zincrbyCommand, 4,
	// 	"write use-memory fast @sortedset",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"zrem", zremCommand, -3,
	// 	"write fast @sortedset",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"zremrangebyscore", zremrangebyscoreCommand, 4,
	// 	"write @sortedset",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"zremrangebyrank", zremrangebyrankCommand, 4,
	// 	"write @sortedset",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"zremrangebylex", zremrangebylexCommand, 4,
	// 	"write @sortedset",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"zunionstore", zunionstoreCommand, -4,
	// 	"write use-memory @sortedset",
	// 	0, zunionInterGetKeys, 0, 0, 0, 0, 0, 0},

	// {"zinterstore", zinterstoreCommand, -4,
	// 	"write use-memory @sortedset",
	// 	0, zunionInterGetKeys, 0, 0, 0, 0, 0, 0},

	// {"zrange", zrangeCommand, -4,
	// 	"read-only @sortedset",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"zrangebyscore", zrangebyscoreCommand, -4,
	// 	"read-only @sortedset",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"zrevrangebyscore", zrevrangebyscoreCommand, -4,
	// 	"read-only @sortedset",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"zrangebylex", zrangebylexCommand, -4,
	// 	"read-only @sortedset",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"zrevrangebylex", zrevrangebylexCommand, -4,
	// 	"read-only @sortedset",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"zcount", zcountCommand, 4,
	// 	"read-only fast @sortedset",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"zlexcount", zlexcountCommand, 4,
	// 	"read-only fast @sortedset",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"zrevrange", zrevrangeCommand, -4,
	// 	"read-only @sortedset",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"zcard", zcardCommand, 2,
	// 	"read-only fast @sortedset",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"zscore", zscoreCommand, 3,
	// 	"read-only fast @sortedset",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"zrank", zrankCommand, 3,
	// 	"read-only fast @sortedset",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"zrevrank", zrevrankCommand, 3,
	// 	"read-only fast @sortedset",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"zscan", zscanCommand, -3,
	// 	"read-only random @sortedset",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"zpopmin", zpopminCommand, -2,
	// 	"write fast @sortedset",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"zpopmax", zpopmaxCommand, -2,
	// 	"write fast @sortedset",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"bzpopmin", bzpopminCommand, -3,
	// 	"write no-script fast @sortedset @blocking",
	// 	0, nil, 1, -2, 1, 0, 0, 0},

	// {"bzpopmax", bzpopmaxCommand, -3,
	// 	"write no-script fast @sortedset @blocking",
	// 	0, nil, 1, -2, 1, 0, 0, 0},

	// {"hset", hsetCommand, -4,
	// 	"write use-memory fast @hash",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"hsetnx", hsetnxCommand, 4,
	// 	"write use-memory fast @hash",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"hget", hgetCommand, 3,
	// 	"read-only fast @hash",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"hmset", hsetCommand, -4,
	// 	"write use-memory fast @hash",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"hmget", hmgetCommand, -3,
	// 	"read-only fast @hash",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"hincrby", hincrbyCommand, 4,
	// 	"write use-memory fast @hash",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"hincrbyfloat", hincrbyfloatCommand, 4,
	// 	"write use-memory fast @hash",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"hdel", hdelCommand, -3,
	// 	"write fast @hash",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"hlen", hlenCommand, 2,
	// 	"read-only fast @hash",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"hstrlen", hstrlenCommand, 3,
	// 	"read-only fast @hash",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"hkeys", hkeysCommand, 2,
	// 	"read-only to-sort @hash",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"hvals", hvalsCommand, 2,
	// 	"read-only to-sort @hash",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"hgetall", hgetallCommand, 2,
	// 	"read-only random @hash",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"hexists", hexistsCommand, 3,
	// 	"read-only fast @hash",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"hscan", hscanCommand, -3,
	// 	"read-only random @hash",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"incrby", incrbyCommand, 3,
	// 	"write use-memory fast @string",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"decrby", decrbyCommand, 3,
	// 	"write use-memory fast @string",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"incrbyfloat", incrbyfloatCommand, 3,
	// 	"write use-memory fast @string",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"getset", getsetCommand, 3,
	// 	"write use-memory fast @string",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"mset", msetCommand, -3,
	// 	"write use-memory @string",
	// 	0, nil, 1, -1, 2, 0, 0, 0},

	// {"msetnx", msetnxCommand, -3,
	// 	"write use-memory @string",
	// 	0, nil, 1, -1, 2, 0, 0, 0},

	// {"randomkey", randomkeyCommand, 1,
	// 	"read-only random @keyspace",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"select", selectCommand, 2,
	// 	"ok-loading fast @keyspace",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"swapdb", swapdbCommand, 3,
	// 	"write fast @keyspace @dangerous",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"move", moveCommand, 3,
	// 	"write fast @keyspace",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// /* Like for SET, we can't mark rename as a fast command because
	//  * overwriting the target key may result in an implicit slow DEL. */
	// {"rename", renameCommand, 3,
	// 	"write @keyspace",
	// 	0, nil, 1, 2, 1, 0, 0, 0},

	// {"renamenx", renamenxCommand, 3,
	// 	"write fast @keyspace",
	// 	0, nil, 1, 2, 1, 0, 0, 0},

	// {"expire", expireCommand, 3,
	// 	"write fast @keyspace",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"expireat", expireatCommand, 3,
	// 	"write fast @keyspace",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"pexpire", pexpireCommand, 3,
	// 	"write fast @keyspace",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"pexpireat", pexpireatCommand, 3,
	// 	"write fast @keyspace",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	{"keys", keysCommand, 2,
		"read-only to-sort @keyspace @dangerous",
		0, nil, 0, 0, 0, 0, 0, 0},

	// {"scan", scanCommand, -2,
	// 	"read-only random @keyspace",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"dbsize", dbsizeCommand, 1,
	// 	"read-only fast @keyspace",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"auth", authCommand, -2,
	// 	"no-script ok-loading ok-stale fast no-monitor no-slowlog @connection",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// /* We don't allow PING during loading since in Redis PING is used as
	//  * failure detection, and a loading server is considered to be
	//  * not available. */
	{"ping", pingCommand, -1,
		"ok-stale fast @connection",
		0, nil, 0, 0, 0, 0, 0, 0},

	// {"echo", echoCommand, 2,
	// 	"read-only fast @connection",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"save", saveCommand, 1,
	// 	"admin no-script",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"bgsave", bgsaveCommand, -1,
	// 	"admin no-script",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"bgrewriteaof", bgrewriteaofCommand, 1,
	// 	"admin no-script",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"shutdown", shutdownCommand, -1,
	// 	"admin no-script ok-loading ok-stale",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"lastsave", lastsaveCommand, 1,
	// 	"read-only random fast @admin @dangerous",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"type", typeCommand, 2,
	// 	"read-only fast @keyspace",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"multi", multiCommand, 1,
	// 	"no-script fast @transaction",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"exec", execCommand, 1,
	// 	"no-script no-monitor no-slowlog @transaction",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"discard", discardCommand, 1,
	// 	"no-script fast @transaction",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"sync", syncCommand, 1,
	// 	"admin no-script",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"psync", syncCommand, 3,
	// 	"admin no-script",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"replconf", replconfCommand, -1,
	// 	"admin no-script ok-loading ok-stale",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"flushdb", flushdbCommand, -1,
	// 	"write @keyspace @dangerous",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"flushall", flushallCommand, -1,
	// 	"write @keyspace @dangerous",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"sort", sortCommand, -2,
	// 	"write use-memory @list @set @sortedset @dangerous",
	// 	0, sortGetKeys, 1, 1, 1, 0, 0, 0},

	// {"info", infoCommand, -1,
	// 	"ok-loading ok-stale random @dangerous",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"monitor", monitorCommand, 1,
	// 	"admin no-script",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"ttl", ttlCommand, 2,
	// 	"read-only fast random @keyspace",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"touch", touchCommand, -2,
	// 	"read-only fast @keyspace",
	// 	0, nil, 1, -1, 1, 0, 0, 0},

	// {"pttl", pttlCommand, 2,
	// 	"read-only fast random @keyspace",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"persist", persistCommand, 2,
	// 	"write fast @keyspace",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"slaveof", replicaofCommand, 3,
	// 	"admin no-script ok-stale",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"replicaof", replicaofCommand, 3,
	// 	"admin no-script ok-stale",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"role", roleCommand, 1,
	// 	"ok-loading ok-stale no-script fast read-only @dangerous",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"debug", debugCommand, -2,
	// 	"admin no-script",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"config", configCommand, -2,
	// 	"admin ok-loading ok-stale no-script",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"subscribe", subscribeCommand, -2,
	// 	"pub-sub no-script ok-loading ok-stale",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"unsubscribe", unsubscribeCommand, -1,
	// 	"pub-sub no-script ok-loading ok-stale",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"psubscribe", psubscribeCommand, -2,
	// 	"pub-sub no-script ok-loading ok-stale",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"punsubscribe", punsubscribeCommand, -1,
	// 	"pub-sub no-script ok-loading ok-stale",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"publish", publishCommand, 3,
	// 	"pub-sub ok-loading ok-stale fast",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"pubsub", pubsubCommand, -2,
	// 	"pub-sub ok-loading ok-stale random",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"watch", watchCommand, -2,
	// 	"no-script fast @transaction",
	// 	0, nil, 1, -1, 1, 0, 0, 0},

	// {"unwatch", unwatchCommand, 1,
	// 	"no-script fast @transaction",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"cluster", clusterCommand, -2,
	// 	"admin ok-stale random",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"restore", restoreCommand, -4,
	// 	"write use-memory @keyspace @dangerous",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"restore-asking", restoreCommand, -4,
	// 	"write use-memory cluster-asking @keyspace @dangerous",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"migrate", migrateCommand, -6,
	// 	"write random @keyspace @dangerous",
	// 	0, migrateGetKeys, 0, 0, 0, 0, 0, 0},

	// {"asking", askingCommand, 1,
	// 	"fast @keyspace",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"readonly", readonlyCommand, 1,
	// 	"fast @keyspace",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"readwrite", readwriteCommand, 1,
	// 	"fast @keyspace",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"dump", dumpCommand, 2,
	// 	"read-only random @keyspace",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"object", objectCommand, -2,
	// 	"read-only random @keyspace",
	// 	0, nil, 2, 2, 1, 0, 0, 0},

	// {"memory", memoryCommand, -2,
	// 	"random read-only",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"client", clientCommand, -2,
	// 	"admin no-script random @connection",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"hello", helloCommand, -2,
	// 	"no-script fast no-monitor no-slowlog @connection",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// /* EVAL can modify the dataset, however it is not flagged as a write
	//  * command since we do the check while running commands from Lua. */
	// {"eval", evalCommand, -3,
	// 	"no-script @scripting",
	// 	0, evalGetKeys, 0, 0, 0, 0, 0, 0},

	// {"evalsha", evalShaCommand, -3,
	// 	"no-script @scripting",
	// 	0, evalGetKeys, 0, 0, 0, 0, 0, 0},

	// {"slowlog", slowlogCommand, -2,
	// 	"admin random",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"script", scriptCommand, -2,
	// 	"no-script @scripting",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"time", timeCommand, 1,
	// 	"read-only random fast",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"bitop", bitopCommand, -4,
	// 	"write use-memory @bitmap",
	// 	0, nil, 2, -1, 1, 0, 0, 0},

	// {"bitcount", bitcountCommand, -2,
	// 	"read-only @bitmap",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"bitpos", bitposCommand, -3,
	// 	"read-only @bitmap",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"wait", waitCommand, 3,
	// 	"no-script @keyspace",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"command", commandCommand, -1,
	// 	"ok-loading ok-stale random @connection",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"geoadd", geoaddCommand, -5,
	// 	"write use-memory @geo",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// /* GEORADIUS has store options that may write. */
	// {"georadius", georadiusCommand, -6,
	// 	"write @geo",
	// 	0, georadiusGetKeys, 1, 1, 1, 0, 0, 0},

	// {"georadius_ro", georadiusroCommand, -6,
	// 	"read-only @geo",
	// 	0, georadiusGetKeys, 1, 1, 1, 0, 0, 0},

	// {"georadiusbymember", georadiusbymemberCommand, -5,
	// 	"write @geo",
	// 	0, georadiusGetKeys, 1, 1, 1, 0, 0, 0},

	// {"georadiusbymember_ro", georadiusbymemberroCommand, -5,
	// 	"read-only @geo",
	// 	0, georadiusGetKeys, 1, 1, 1, 0, 0, 0},

	// {"geohash", geohashCommand, -2,
	// 	"read-only @geo",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"geopos", geoposCommand, -2,
	// 	"read-only @geo",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"geodist", geodistCommand, -4,
	// 	"read-only @geo",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"pfselftest", pfselftestCommand, 1,
	// 	"admin @hyperloglog",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"pfadd", pfaddCommand, -2,
	// 	"write use-memory fast @hyperloglog",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// /* Technically speaking PFCOUNT may change the key since it changes the
	//  * final bytes in the HyperLogLog representation. However in this case
	//  * we claim that the representation, even if accessible, is an internal
	//  * affair, and the command is semantically read only. */
	// {"pfcount", pfcountCommand, -2,
	// 	"read-only @hyperloglog",
	// 	0, nil, 1, -1, 1, 0, 0, 0},

	// {"pfmerge", pfmergeCommand, -2,
	// 	"write use-memory @hyperloglog",
	// 	0, nil, 1, -1, 1, 0, 0, 0},

	// {"pfdebug", pfdebugCommand, -3,
	// 	"admin write",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"xadd", xaddCommand, -5,
	// 	"write use-memory fast random @stream",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"xrange", xrangeCommand, -4,
	// 	"read-only @stream",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"xrevrange", xrevrangeCommand, -4,
	// 	"read-only @stream",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"xlen", xlenCommand, 2,
	// 	"read-only fast @stream",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"xread", xreadCommand, -4,
	// 	"read-only no-script @stream @blocking",
	// 	0, xreadGetKeys, 1, 1, 1, 0, 0, 0},

	// {"xreadgroup", xreadCommand, -7,
	// 	"write no-script @stream @blocking",
	// 	0, xreadGetKeys, 1, 1, 1, 0, 0, 0},

	// {"xgroup", xgroupCommand, -2,
	// 	"write use-memory @stream",
	// 	0, nil, 2, 2, 1, 0, 0, 0},

	// {"xsetid", xsetidCommand, 3,
	// 	"write use-memory fast @stream",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"xack", xackCommand, -4,
	// 	"write fast random @stream",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"xpending", xpendingCommand, -3,
	// 	"read-only random @stream",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"xclaim", xclaimCommand, -6,
	// 	"write random fast @stream",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"xinfo", xinfoCommand, -2,
	// 	"read-only random @stream",
	// 	0, nil, 2, 2, 1, 0, 0, 0},

	// {"xdel", xdelCommand, -3,
	// 	"write fast @stream",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"xtrim", xtrimCommand, -2,
	// 	"write random @stream",
	// 	0, nil, 1, 1, 1, 0, 0, 0},

	// {"post", securityWarningCommand, -1,
	// 	"ok-loading ok-stale read-only",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"host:", securityWarningCommand, -1,
	// 	"ok-loading ok-stale read-only",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"latency", latencyCommand, -2,
	// 	"admin no-script ok-loading ok-stale",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"lolwut", lolwutCommand, -1,
	// 	"read-only fast",
	// 	0, nil, 0, 0, 0, 0, 0, 0},

	// {"acl", aclCommand, -2,
	// 	"admin no-script ok-loading ok-stale",
	// 	0, nil, 0, 0, 0, 0, 0, 0},
}

func LoadFromFile() {
	filename := "redis-commands.json"
	plan, _ := ioutil.ReadFile(filename)
	var data interface{}
	err := json.Unmarshal(plan, &data)
	if nil != err {
		fmt.Println(err)
		panic("Unable to load commands")
	}
	fmt.Println(data.(map[string]interface{})["keys"])
}

var CMD_WRITE uint64 = (1 << 0)           /* "write" flag */
var CMD_READONLY uint64 = (1 << 1)        /* "read-only" flag */
var CMD_DENYOOM uint64 = (1 << 2)         /* "use-memory" flag */
var CMD_MODULE uint64 = (1 << 3)          /* Command exported by module. */
var CMD_ADMIN uint64 = (1 << 4)           /* "admin" flag */
var CMD_PUBSUB uint64 = (1 << 5)          /* "pub-sub" flag */
var CMD_NOSCRIPT uint64 = (1 << 6)        /* "no-script" flag */
var CMD_RANDOM uint64 = (1 << 7)          /* "random" flag */
var CMD_SORT_FOR_SCRIPT uint64 = (1 << 8) /* "to-sort" flag */
var CMD_LOADING uint64 = (1 << 9)         /* "ok-loading" flag */
var CMD_STALE uint64 = (1 << 10)          /* "ok-stale" flag */
var CMD_SKIP_MONITOR uint64 = (1 << 11)   /* "no-monitor" flag */
var CMD_SKIP_SLOWLOG uint64 = (1 << 12)   /* "no-slowlog" flag */
var CMD_ASKING uint64 = (1 << 13)         /* "cluster-asking" flag */
var CMD_FAST uint64 = (1 << 14)           /* "fast" flag */

var CMD_SLOW uint64 = (1 << 15) /* slow flag*/

func populateCommandTableParseFlags(c *RedisCommand, flags string) error {
	/* Split the line into arguments for processing. */
	argv := strings.Split(flags, " ")
	if len(argv) < 1 {
		return errors.New("invalid flag")
	}

	for j := 0; j < len(argv); j++ {
		flag := argv[j]
		if strings.EqualFold(flag, "write") {
			c.flags |= CMD_WRITE
		} else if strings.EqualFold(flag, "read-only") {
			c.flags |= CMD_READONLY
		} else if strings.EqualFold(flag, "use-memory") {
			c.flags |= CMD_DENYOOM
		} else if strings.EqualFold(flag, "admin") {
			c.flags |= CMD_ADMIN
		} else if strings.EqualFold(flag, "pub-sub") {
			c.flags |= CMD_PUBSUB
		} else if strings.EqualFold(flag, "no-script") {
			c.flags |= CMD_NOSCRIPT
		} else if strings.EqualFold(flag, "random") {
			c.flags |= CMD_RANDOM
		} else if strings.EqualFold(flag, "to-sort") {
			c.flags |= CMD_SORT_FOR_SCRIPT
		} else if strings.EqualFold(flag, "ok-loading") {
			c.flags |= CMD_LOADING
		} else if strings.EqualFold(flag, "ok-stale") {
			c.flags |= CMD_STALE
		} else if strings.EqualFold(flag, "no-monitor") {
			c.flags |= CMD_SKIP_MONITOR
		} else if strings.EqualFold(flag, "no-slowlog") {
			c.flags |= CMD_SKIP_SLOWLOG
		} else if strings.EqualFold(flag, "cluster-asking") {
			c.flags |= CMD_ASKING
		} else if strings.EqualFold(flag, "fast") {
			c.flags |= CMD_FAST
		}
	}
	/* If it's not @fast is @slow in this binary world. */
	if 0 == (c.flags & CMD_FAST) {
		c.flags |= CMD_SLOW
	}
	return nil
}

func PopulateCommandTable() map[string]*RedisCommand {
	commandMap := make(map[string]*RedisCommand)
	numcommands := len(redisCommandTable)

	for j := 0; j < numcommands; j++ {
		c := redisCommandTable[j]
		if populateCommandTableParseFlags(c, c.sflags) != nil {
			panic("Unsupported command flag")
		}
		commandMap[c.name] = c
	}
	return commandMap
}

func (cmd *RedisCommand) Writable() bool {
	return CMD_WRITE == (cmd.flags & CMD_WRITE)
}
