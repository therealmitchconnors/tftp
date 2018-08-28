package tftp

import "testing"

func TestSetData(t *testing.T) {
	m := MapDataStore{mapStore: make(map[string][][]byte)}
	value := make([][]byte, 10)
	var cur byte
	for i := 0; i < 10; i++ {
		value[i] = make([]byte, 512)
		for j := 0; j < 512; j++ {
			value[i][j] = cur
			cur++
		}
	}
	key := "fhqwgads"
	m.setData(key, value)
	if !m.keyExists(key) {
		t.Error("key fhqwgads was set, exists returns false")
	}
	value2 := m.getData(key)
	if len(value) != len(value2) {
		t.Errorf("We gave an array of len %d but got back an array of len %d", len(value), len(value2))
	}
	if len(value[0]) != len(value2[0]) {
		t.Errorf("Input array and 2nd dimension of %d while output array had %d", len(value[0]), len(value2[0]))
	}
	for i := range value {
		for j := range value[i] {
			if value[i][j] != value2[i][j] {
				t.Errorf("input array [%d][%d] had value %d, while output array had %d", i, j, value[i][j], value2[i][j])
			}
		}
	}
}
