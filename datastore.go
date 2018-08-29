package tftp

import "sync"

type datastore interface {
	keyExists(key string) bool
	getData(key string) [][]byte
	setData(key string, value [][]byte)
}

type MapDataStore struct {
	mapStore map[string][][]byte
	lock     sync.RWMutex
}

// using a single RWMutex will lock the entire
// map on write.
// https://github.com/orcaman/concurrent-map
// may be a performance improvement.
// var lock = sync.RWMutex{}

func (m *MapDataStore) keyExists(key string) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	_, ok := m.mapStore[key]
	return ok
}

// TODO: move this to interface for dependency injection
func (m *MapDataStore) getData(key string) [][]byte {
	// here we need a thread-safe map of string to 2d
	// array of bytes, whose shape is n x 512
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.mapStore[key]
}

func (m *MapDataStore) setData(key string, value [][]byte) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.mapStore[key] = value
}
