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
