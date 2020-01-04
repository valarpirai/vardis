package cache

import (
	"regexp"
)

type CacheStorage struct {
	store map[string]CacheData
}
type CacheData struct {
	val interface{}
	exp uint64
}
type Cache interface {
	Set(string, string) string
	Get(string) string
	Exists(string) string
}

// New Initialize in-memory cache store
func NewCache() (ca *CacheStorage) {
	ca = new(CacheStorage)
	ca.store = make(map[string]CacheData)
	return ca
}

func (c *CacheStorage) Set(key string, val string) string {
	c.store[key] = CacheData{
		val: val,
		exp: 0,
	}
	return "OK"
}

func (c *CacheStorage) Get(key string) (string, bool) {
	if data, ok := c.store[key]; ok {
		return data.val.(string), true
	}
	return "", false
}

func (c *CacheStorage) Exists(key string) int {
	if _, ok := c.store[key]; ok {
		return 1
	}
	return 0
}

func (c *CacheStorage) Keys(pattern string) []string {
	r, _ := regexp.Compile(pattern)
	keys := make([]string, 0, len(c.store))
	for k := range c.store {
		if r.MatchString(k) {
			keys = append(keys, k)
		}
	}
	return keys
}

// func (cache *CacheStorage) ProcessCommands(req proto.RequestInterface) (result interface{}) {
// 	log.Debug("Command Length: " + strconv.Itoa(req.CommandLength()))
// 	log.Debugf("COMMAND -> %#v", req.Command())
// 	switch req.Command() {
// 	case "PING":
// 		result = "PONG"
// 	case "SET":
// 		key, val := req.Key(), req.Value()
// 		result = cache.Set(key, val)
// 	case "GET":
// 		key := req.Key()
// 		res, ok := cache.Get(key)
// 		if ok {
// 			result = res
// 		}
// 	case "EXISTS":
// 		key := req.Key()
// 		result = cache.Exists(key)
// 	case "KEYS":
// 		result = cache.Keys(req.Key())
// 	default:
// 		result = nil
// 	}
// 	return
// }
