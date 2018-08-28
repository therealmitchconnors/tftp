package tftp

import "sync"

var mapStore = make(map[string][][]byte)

// using a single RWMutex will lock the entire
// map on write.
// https://github.com/orcaman/concurrent-map
// may be a performance improvement.
var lock = sync.RWMutex{}

func keyExists(key string) bool {
	lock.RLock()
	defer lock.RUnlock()
	_, ok := mapStore[key]
	return ok
}

// TODO: move this to interface for dependency injection
func getData(key string) [][]byte {
	// here we need a thread-safe map of string to 2d
	// array of bytes, whose shape is n x 512
	lock.RLock()
	defer lock.RUnlock()
	return mapStore[key]
}

func setData(key string, value [][]byte) {
	lock.Lock()
	defer lock.Unlock()
	mapStore[key] = value
}
