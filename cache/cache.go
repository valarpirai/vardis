package cache

import (
	"regexp"
)

var OBJ_STRING uint8 = 0 /* String object. */
var OBJ_LIST uint8 = 1   /* List object. */
var OBJ_SET uint8 = 2    /* Set object. */
var OBJ_ZSET uint8 = 3   /* Sorted set object. */
var OBJ_HASH uint8 = 4   /* Hash object. */

type CacheStorage struct {
	store map[string]*CacheData
}
type CacheData struct {
	val      interface{}
	exp      int64
	dataType uint8
}
type Cache interface {
	Set(string, string) string
	Get(string) string
	Exists(string) string
}

type ICacheStorage interface {
	Store() map[string]*CacheData
}

type ICacheData interface {
	Value() interface{}
	SetValue(interface{})
	Expires() uint64
	Type() uint8
}

// New Initialize in-memory cache store
func NewCache() (ca *CacheStorage) {
	ca = new(CacheStorage)
	ca.store = make(map[string]*CacheData)
	return ca
}

func (c *CacheStorage) Store() map[string]*CacheData {
	return c.store
}

func (c *CacheStorage) Set(key string, val string) string {
	c.store[key] = &CacheData{
		val:      val,
		exp:      0,
		dataType: OBJ_STRING,
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

func (c *CacheData) Value() interface{} {
	return c.val
}
func (c *CacheData) SetValue(val interface{}) {
	c.val = val
}
func (c *CacheData) Expires() int64 {
	return c.exp
}
func (c *CacheData) SetExpires(exp int64) {
	c.exp = exp
}
func (c *CacheData) Type() uint8 {
	return c.dataType
}
func (c *CacheData) SetType(dataType uint8) {
	c.dataType = dataType
}
