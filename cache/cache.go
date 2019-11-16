package cache

type cache struct {
	store map[string]string
}

type Cache interface {
	Set(string, string) string
	Get(string) string
	Exists(string) string
}

// New Initialize inmemory cache store
func NewCache() (ca *cache) {
	ca = new(cache)
	ca.store = make(map[string]string)
	return ca
}

func (c *cache) Set(key string, val string) string {
	c.store[key] = val
	return "OK"
}

func (c *cache) Get(key string) (string, bool) {
	if val, ok := c.store[key]; ok {
		return val, true
	}
	return "", false
}

func (c *cache) Exists(key string) bool {
	if _, ok := c.store[key]; ok {
		return true
	}
	return false
}
