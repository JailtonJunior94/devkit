package driverreg

import "sync"

var (
	mu         sync.RWMutex
	registered = map[string]struct{}{}
)

func Register(name string) {
	if name == "" {
		panic("database: register driver with empty name")
	}

	mu.Lock()
	registered[name] = struct{}{}
	mu.Unlock()
}

func IsRegistered(name string) bool {
	mu.RLock()
	defer mu.RUnlock()

	_, ok := registered[name]
	return ok
}
