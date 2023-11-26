# Sparse store

This data structure efficiently stores sparse data (slices) in memory. It supports both `set` and `get` operations for specific segments, treating the data as a contiguous slice. The underlying structure dynamically allocates and stores only the bits that are set, optimizing memory usage.

It merges small contiguous chunks (16384 entries by default, configurable with `WithMinContiguous`) into a larger slice for more efficient storage and retrieval speed.

## Usage

```go
// Create a byte store that will merge chunks of up to 1kiB.
s := store.NewStore[byte](store.WithMinContiguous[byte](1 << 10))

// Create an input buffer of 1MiB.
in := bytes.Repeat([]byte{1, 2, 3, 4}, 1<<20/4)
// Populate the values at an offset of 1GiB minus 1MiB.
s.Set(1<<30-1<<20, in)

// Get the occupancy (how many values are populated).
occupancy := s.Occupancy() // -> 1MiB
// Get the length (how much data the structure represents).
length := s.Length() // -> 1GiB

fmt.Printf("Occupancy: %v, length: %v\n", occupancy, length)
// -> Occupancy: 1048576, length: 1073741824

// Create an output buffer of 1kiB.
out := make([]byte, 1<<10)
// Check if the data would be complete at an offset of 1GiB minus 1MiB
s.Has(1<<30-1<<20, len(out)) // -> true
// Retrieve data at an offset of 1GiB.
has := s.Get(1<<30-1<<20, out) // -> true

fmt.Printf("For 1KiB at 1GiB-1MiB (complete = %v): %v\n", has, out[:4])
// -> For 1kiB at 1GiB-1MiB (complete = true): [1 2 3 4]

// Create a new output buffer of 1kiB.
out = make([]byte, 1<<10)
// Check if the data would be complete at an offset of 1GiB.
s.Has(1<<30, len(out)) // -> false
// Retrieve data at an offset of 1GiB.
has = s.Get(1<<30, out) // -> false

fmt.Printf("For 1KiB at 1GiB (complete = %v): %v\n", has, out[:4])
// -> For 1kiB at 1GiB (complete = false): [0 0 0 0] 
```
