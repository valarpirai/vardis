package commands

import (
	"github.com/valarpirai/vardis/cache"
	"github.com/valarpirai/vardis/proto"
)

// String command implementation
func getCommand(req *proto.Request, cache *cache.CacheStorage) (result interface{}) {
	res, ok := cache.Get(req.Key())
	if ok {
		result = res
	}
	return result
}
func setCommand(req *proto.Request, cache *cache.CacheStorage) (result interface{}) {
	key, val := req.Key(), req.Value()
	result = cache.Set(key, val)
	return result
}
func delCommand(req *proto.Request, cache *cache.CacheStorage) (result interface{}) {
	return
}
func keysCommand(req *proto.Request, cache *cache.CacheStorage) (result interface{}) {
	result = cache.Keys(req.Key())
	return
}
func existsCommand(req *proto.Request, cache *cache.CacheStorage) (result interface{}) {
	key := req.Key()
	result = cache.Exists(key)
	return
}
func pingCommand(req *proto.Request, cache *cache.CacheStorage) (result interface{}) {
	return "PONG"
}
