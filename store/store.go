package store

import (
	"sort"

	"slices"
)

const defaultMinContiguous = 16 << 10 // 16 Ki

type entry[T any] struct {
	order  int
	offset int
	data   []T
}

type entries[T any] []entry[T]

func (e entries[T]) Search(x int) int {
	return sort.Search(len(e), func(i int) bool {
		return e[i].offset >= x
	})
}

type Store[T any] struct {
	minContiguous int

	entries     entries[T]
	insertCount int
	occupancy   int
	length      int
}

type Option[T any] func(*Store[T])

func WithMinContiguous[T any](minContiguous int) Option[T] {
	return func(c *Store[T]) {
		c.minContiguous = minContiguous
	}
}

func NewStore[T any](opts ...Option[T]) *Store[T] {
	cache := &Store[T]{
		minContiguous: defaultMinContiguous,
	}

	for _, opt := range opts {
		opt(cache)
	}

	return cache
}

func (c *Store[T]) Occupancy() int {
	return c.occupancy
}

func (c *Store[T]) Length() int {
	return c.length
}

// Has returns true if the cache contains data at `offset` with length
// `length`.
func (c *Store[T]) Has(offset int, length int) bool {
	if len(c.entries) == 0 && length > 0 {
		return false
	}

	completeTo := offset
	for _, entry := range c.entries {
		// If the entry is before the requested range, skip it.
		if entry.offset+len(entry.data) < offset {
			continue
		}
		// If the entry starts after the requested range, or if there
		// is a gap between the previous entry and this one, we're done.
		if entry.offset > offset+length || completeTo < entry.offset {
			break
		}

		completeTo = entry.offset + len(entry.data)
	}

	// If the cache contains the complete range, return true.
	return completeTo >= offset+length
}

// Get populates `p` with the data at `offset`. If the cache does not contain the
// complete data for this range, Get returns false.
func (c *Store[T]) Get(offset int, p []T) bool {
	if len(c.entries) == 0 && len(p) > 0 {
		return false
	}

	// The logic for completeTo is the same as in Has, but we have to continue
	// iterating over the entries to populate `p`.
	completeTo := offset
	complete := true
	for _, entry := range c.entries {
		if entry.offset+len(entry.data) < offset {
			continue
		}
		if entry.offset > offset+len(p) {
			break
		}

		if completeTo < entry.offset {
			complete = false
		}

		offsetDelta := entry.offset - offset
		if offsetDelta < 0 {
			copy(p, entry.data[-offsetDelta:])
		} else {
			copy(p[offsetDelta:], entry.data)
		}

		completeTo = entry.offset + len(entry.data)
	}

	return complete && completeTo >= offset+len(p)
}

// Set sets the cache data at `offset` to `p`. If the cache already contains
// data at `offset`, it is overwritten.
func (c *Store[T]) Set(offset int, p []T) {
	i := c.entries.Search(offset)
	c.entries = slices.Insert(c.entries, i, entry[T]{c.insertCount, offset, p})
	c.insertCount++

	// If the length increased, update it.
	if c.length < offset+len(p) {
		c.length = offset + len(p)
	}

	// Update the occupancy optimistically. If the entry is compacted, the
	// occupancy will be updated again.
	c.occupancy += len(p)

	c.compact()
}

// compact compacts the cache by merging adjacent entries and removing
// overlapping entries.
func (c *Store[T]) compact() {
	for i := 0; i < len(c.entries)-1; i++ {
		// We use references here as we want to update the entries in place
		// when reslicing.
		current := &c.entries[i]
		next := &c.entries[i+1]

		currentMin := current.offset
		currentMax := current.offset + len(current.data)
		nextMin := next.offset
		nextMax := next.offset + len(next.data)

		if nextMin < currentMax {
			// If the current entry encompasses the next entry, copy if needed.
			if nextMax <= currentMax {
				// If the next entry has a higher order, copy.
				if current.order < next.order {
					copy(current.data[nextMin-currentMin:], next.data)
				}

				c.entries = append(c.entries[:i+1], c.entries[i+2:]...)
				c.occupancy -= len(next.data)
				i--
				continue
			} else {
				// If the entries overlap reslice so that they become contiguous.
				c.occupancy -= currentMax - nextMin
				if current.order < next.order {
					current.data = current.data[:nextMin-currentMin]
					currentMax = nextMin
				} else {
					next.data = next.data[currentMax-nextMin:]
					next.offset = currentMax
					nextMin = currentMax
				}
			}
		}

		// If the entries are contiguous and small enough, combine them.
		if currentMax == nextMin && nextMax-currentMin <= c.minContiguous {
			newData := make([]T, nextMax-currentMin)
			copy(newData, current.data)
			copy(newData[currentMax-currentMin:], next.data)
			c.entries[i] = entry[T]{current.order, currentMin, newData}
			c.entries = append(c.entries[:i+1], c.entries[i+2:]...)
			i--
		}
	}
}
