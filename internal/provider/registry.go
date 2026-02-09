package provider

import (
	"fmt"
	"sort"
	"sync"
)

var (
	mu        sync.RWMutex
	providers = make(map[string]VMProvider)
)

func Register(name string, vmProvider VMProvider) {
	mu.Lock()
	defer mu.Unlock()
	providers[name] = vmProvider
}

func Get(name string) (VMProvider, error) {
	mu.RLock()
	defer mu.RUnlock()
	vmProvider, exists := providers[name]
	if !exists {
		return nil, fmt.Errorf("unknown provider %q: use 'multipass' or 'aws'", name)
	}
	return vmProvider, nil
}

func List() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func Reset() {
	mu.Lock()
	defer mu.Unlock()
	providers = make(map[string]VMProvider)
}
