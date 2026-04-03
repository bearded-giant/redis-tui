package cmd

import (
	"sync"
)

var (
	mu           sync.RWMutex
	scanSize     int64 = 1000
	includeTypes bool  = true
)

func GetScanSize() int64 {
	mu.RLock()
	defer mu.RUnlock()
	return scanSize
}

func SetScanSize(s int64) {
	mu.Lock()
	defer mu.Unlock()
	scanSize = s
}

func GetIncludeTypes() bool {
	mu.RLock()
	defer mu.RUnlock()
	return includeTypes
}

func SetIncludeTypes(v bool) {
	mu.Lock()
	defer mu.Unlock()
	includeTypes = v
}
