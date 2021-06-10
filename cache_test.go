package cache

import "testing"

// test is a helper that accepts an slice of operations (e.g. [["Put", "foo", "bar"], ["Get", "foo", "bar"]]) and test the behavior
func test(t *testing.T, cache *Cache, operations [][]interface{}) {
	for i := 0; i < len(operations); i++ {
		operation := operations[i]
		kind := operation[0]
		if kind == "Put" {
			cache.Put(CacheKey(operation[1].(string)), operation[2].(string))
		} else { // "Get"
			value, err := cache.Get(CacheKey(operation[1].(string)))
			if operation[2] == nil { // value should be evicted
				if value != nil {
					t.Errorf("key = %s, value should be nil, but got %s", operation[1].(string), *value)
				}
			} else {
				if err == nil && *value != operation[2].(string) {
					t.Errorf("key = %s, value should be %s, but we got %s", operation[1].(string), operation[2].(string), *value)
				}
			}
		}
	}
}

func TestFIFOPolicy(t *testing.T) {
	testCase := [][]interface{}{
		{"Put", "1", "1"},
		{"Put", "2", "2"},
		{"Put", "3", "3"},
		{"Put", "4", "4"},
		{"Put", "5", "5"},
		{"Get", "1", "1"},
		{"Get", "2", "2"},
		{"Get", "3", "3"},
		{"Get", "4", "4"},
		{"Get", "5", "5"},
		{"Put", "6", "6"}, // 1 is evicted
		{"Get", "1", nil},
		{"Get", "6", "6"},
		{"Put", "7", "7"}, // 2 is evicted
		{"Get", "1", nil},
		{"Get", "2", nil},
		{"Get", "7", "7"},
	}

	cache := NewCache(5, FIFO)
	test(t, cache, testCase)
}

func TestLRUPolicy(t *testing.T) {
	testCase := [][]interface{}{
		{"Put", "1", "1"},
		{"Put", "2", "2"},
		{"Put", "3", "3"},
		{"Put", "4", "4"},
		{"Put", "5", "5"},
		{"Get", "1", "1"},
		{"Get", "2", "2"},
		{"Put", "6", "6"}, // 3 is evicted
		{"Get", "3", nil},
		{"Get", "1", "1"},
		{"Get", "6", "6"},
		{"Put", "7", "7"}, // 4 is evicted
		{"Get", "4", nil},
		{"Get", "7", "7"},
	}

	cache := NewCache(5, LRU)
	test(t, cache, testCase)
}
