package hyphae

import (
	"sync/atomic"
)

// Its value is number of all existing hyphae. NonEmptyHypha mutators are expected to manipulate the value. It is concurrent-safe.
var count atomic.Int32

// Count how many hyphae there are. This is a O(1), the number of hyphae is stored in memory.
func Count() int {
	return int(count.Load())
}

// incrementCount increments the value of the hyphae counter. Use when creating new hyphae or loading hyphae from disk.
func incrementCount() {
	count.Add(1)
}

// decrementCount decrements the value of the hyphae counter. Use when deleting existing hyphae.
func decrementCount() {
	count.Add(-1)
}

func addCount(value int) {
	count.Add(int32(value))
}

func setCount(value int) {
	count.Store(int32(value))
}
