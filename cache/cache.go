package cache

type CacheStorage struct {
	store map[string]string
}

type Cache interface {
	Set(string, string) string
	Get(string) string
	Exists(string) string
}

// New Initialize inmemory cache store
func NewCache() (ca *CacheStorage) {
	ca = new(CacheStorage)
	ca.store = make(map[string]string)
	return ca
}

func (c *CacheStorage) Set(key string, val string) string {
	c.store[key] = val
	return "OK"
}

func (c *CacheStorage) Get(key string) (string, bool) {
	if val, ok := c.store[key]; ok {
		return val, true
	}
	return "", false
}

func (c *CacheStorage) Exists(key string) int {
	if _, ok := c.store[key]; ok {
		return 1
	}
	return 0
}
