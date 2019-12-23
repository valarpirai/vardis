package cache

import (
	"bufio"
	"regexp"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/valarpirai/vardis/proto"
)

type CacheStorage struct {
	store map[string]interface{}
}

type Cache interface {
	Set(string, string) string
	Get(string) string
	Exists(string) string
}

// New Initialize in-memory cache store
func NewCache() (ca *CacheStorage) {
	ca = new(CacheStorage)
	ca.store = make(map[string]interface{})
	return ca
}

func (c *CacheStorage) LoadFromDisk(p *Persistance) {
	log.SetLevel(log.WarnLevel)
	defer log.SetLevel(log.DebugLevel)
	reader := bufio.NewReader(p.aofFile)
	for {
		netData, _, err := proto.Decode(reader)
		if err != nil {
			log.Error(err)
			break
		}
		request := proto.ParseCommand(netData)
		c.ProcessCommands(request)
	}
	log.Warn("Data Loaded successfully")
}

func (c *CacheStorage) Set(key string, val string) string {
	c.store[key] = val
	return "OK"
}

func (c *CacheStorage) Get(key string) (string, bool) {
	if val, ok := c.store[key]; ok {
		return val.(string), true
	}
	return "", false
}

func (c *CacheStorage) Exists(key string) int {
	if _, ok := c.store[key]; ok {
		return 1
	}
	return 0
}

func (cache *CacheStorage) ProcessCommands(req proto.RequestInterface) (result interface{}) {
	log.Debug("Command Length: " + strconv.Itoa(req.CommandLength()))
	log.Debugf("COMMAND -> %#v", req.Command())
	switch req.Command() {
	case "PING":
		result = "PONG"
	case "SET":
		key, val := req.Key(), req.Value()
		result = cache.Set(key, val)
	case "GET":
		key := req.Key()
		res, ok := cache.Get(key)
		if ok {
			result = res
		}
	case "EXISTS":
		key := req.Key()
		result = cache.Exists(key)
	case "KEYS":
		r, _ := regexp.Compile(req.Key())
		keys := make([]string, 0, len(cache.store))
		for k := range cache.store {
			if r.MatchString(k) {
				keys = append(keys, k)
			}
		}
		result = keys
	default:
		result = nil
	}
	return
}
