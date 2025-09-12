package hyphae

import (
	"sync/atomic"
)

// Its value is number of all existing hyphae. NonEmptyHypha mutators are expected to manipulate the value. It is concurrent-safe.
var count atomic.Int32

// ResetCount sets the value of hyphae count to zero. Use when reloading hyphae.
func ResetCount() {
	count.Store(0)
}

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
