package store_test

import (
	"bytes"
	"fmt"
	"math/rand"

	"testing"

	"github.com/aertje/sparse-store/store"
	"github.com/stretchr/testify/assert"
)

type entry struct {
	offset int
	data   []byte
}

func TestStoreSet(t *testing.T) {
	for _, topt := range []struct {
		name string
		opt  store.Option[byte]
	}{
		{
			name: "default",
			opt:  store.WithMinContiguous[byte](16 << 10), // 16 KiB
		},
		{
			name: "never merge",
			opt:  store.WithMinContiguous[byte](1),
		},
	} {
		{
			for _, tc := range []struct {
				name              string
				content           []entry
				expectedLength    int
				expectedOccupancy int
				expectedContent   []byte
			}{
				{
					name:              "empty",
					content:           []entry{},
					expectedLength:    0,
					expectedOccupancy: 0,
					expectedContent:   []byte{0, 0},
				},
				{
					name: "continuous",
					content: []entry{
						{offset: 0, data: []byte{0}},
						{offset: 1, data: []byte{1}},
					},
					expectedLength:    2,
					expectedOccupancy: 2,
					expectedContent:   []byte{0, 1},
				},
				{
					name: "continuous other order",
					content: []entry{
						{offset: 1, data: []byte{1}},
						{offset: 0, data: []byte{0}},
					},
					expectedLength:    2,
					expectedOccupancy: 2,
					expectedContent:   []byte{0, 1},
				},
				{
					name: "overlapping",
					content: []entry{
						{offset: 0, data: []byte{0, 1}},
						{offset: 1, data: []byte{10, 2}},
					},
					expectedLength:    3,
					expectedOccupancy: 3,
					expectedContent:   []byte{0, 10, 2},
				},
				{
					name: "overlapping other order",
					content: []entry{
						{offset: 1, data: []byte{1, 2}},
						{offset: 0, data: []byte{0, 10}},
					},
					expectedLength:    3,
					expectedOccupancy: 3,
					expectedContent:   []byte{0, 10, 2},
				},
				{
					name: "inside",
					content: []entry{
						{offset: 0, data: []byte{0, 1, 2}},
						{offset: 1, data: []byte{10}},
					},
					expectedLength:    3,
					expectedOccupancy: 3,
					expectedContent:   []byte{0, 10, 2},
				},
				{
					name: "gap",
					content: []entry{
						{offset: 0, data: []byte{0, 1, 2}},
						{offset: 4, data: []byte{4}},
					},
					expectedLength:    5,
					expectedOccupancy: 4,
					expectedContent:   []byte{0, 1, 2, 0, 4},
				},
				{
					name: "gap other order",
					content: []entry{
						{offset: 4, data: []byte{4}},
						{offset: 0, data: []byte{0, 1, 2}},
					},
					expectedLength:    5,
					expectedOccupancy: 4,
					expectedContent:   []byte{0, 1, 2, 0, 4},
				},
				{
					name: "filled gap",
					content: []entry{
						{offset: 0, data: []byte{0, 1, 2}},
						{offset: 4, data: []byte{4}},
						{offset: 3, data: []byte{3}},
					},
					expectedLength:    5,
					expectedOccupancy: 5,
					expectedContent:   []byte{0, 1, 2, 3, 4},
				},
				{
					name: "filled gap other order",
					content: []entry{
						{offset: 3, data: []byte{3}},
						{offset: 0, data: []byte{0, 1, 2}},
						{offset: 4, data: []byte{4}},
					},
					expectedLength:    5,
					expectedOccupancy: 5,
					expectedContent:   []byte{0, 1, 2, 3, 4},
				},
				{
					name: "double gap encompassed",
					content: []entry{
						{offset: 1, data: []byte{1}},
						{offset: 3, data: []byte{3}},
						{offset: 0, data: []byte{0, 10, 2, 30, 4}},
					},
					expectedLength:    5,
					expectedOccupancy: 5,
					expectedContent:   []byte{0, 10, 2, 30, 4},
				},
				{
					name: "double gap filled",
					content: []entry{
						{offset: 1, data: []byte{1}},
						{offset: 3, data: []byte{3}},
						{offset: 1, data: []byte{10, 2, 30}},
					},
					expectedLength:    4,
					expectedOccupancy: 3,
					expectedContent:   []byte{0, 10, 2, 30},
				},
				{
					name: "double gap not filled",
					content: []entry{
						{offset: 1, data: []byte{1}},
						{offset: 3, data: []byte{3}},
						{offset: 3, data: []byte{30, 4}},
					},
					expectedLength:    5,
					expectedOccupancy: 3,
					expectedContent:   []byte{0, 1, 0, 30, 4},
				},
			} {
				t.Run(fmt.Sprintf("%v %v", topt.name, tc.name), func(t *testing.T) {
					s := store.NewStore(topt.opt)

					for _, entry := range tc.content {
						s.Set(entry.offset, entry.data)
					}

					assert.Equal(t, tc.expectedLength, s.Length())
					assert.Equal(t, tc.expectedOccupancy, s.Occupancy())
					data := make([]byte, len(tc.expectedContent))
					s.Get(0, data)
					assert.Equal(t, tc.expectedContent, data)
				})
			}
		}
	}
}

func TestStoreGetAndHas(t *testing.T) {
	for _, tc := range []struct {
		name            string
		content         []entry
		offset          int
		expectedContent []byte
		expectHas       bool
	}{
		{
			name:            "empty, nothing requested",
			content:         []entry{},
			offset:          0,
			expectedContent: []byte{},
			expectHas:       true,
		},
		{
			name:            "empty with overfetch",
			content:         []entry{},
			offset:          1,
			expectedContent: []byte{0, 0},
			expectHas:       false,
		},
		{
			name: "continuous",
			content: []entry{
				{offset: 1, data: []byte{1, 2}},
			},
			offset:          1,
			expectedContent: []byte{1, 2},
			expectHas:       true,
		},
		{
			name: "continuous with inside fetch",
			content: []entry{
				{offset: 1, data: []byte{1, 2, 3}},
			},
			offset:          2,
			expectedContent: []byte{2, 3},
			expectHas:       true,
		},
		{
			name: "continuous with underfetch miss",
			content: []entry{
				{offset: 1, data: []byte{1, 2}},
			},
			offset:          0,
			expectedContent: []byte{0},
			expectHas:       false,
		},
		{
			name: "continuous with overfetch miss",
			content: []entry{
				{offset: 1, data: []byte{1, 2}},
			},
			offset:          3,
			expectedContent: []byte{0},
			expectHas:       false,
		},
		{
			name: "continuous with underfetch hit",
			content: []entry{
				{offset: 1, data: []byte{1, 2}},
			},
			offset:          0,
			expectedContent: []byte{0, 1},
			expectHas:       false,
		},
		{
			name: "continuous with overfetch hit",
			content: []entry{
				{offset: 1, data: []byte{1, 2}},
			},
			offset:          2,
			expectedContent: []byte{2, 0},
			expectHas:       false,
		},
		{
			name: "continuous with under- and overfetch",
			content: []entry{
				{offset: 1, data: []byte{1, 2}},
			},
			offset:          0,
			expectedContent: []byte{0, 1, 2, 0},
			expectHas:       false,
		},
		{
			name: "gap with offset",
			content: []entry{
				{offset: 1, data: []byte{1}},
				{offset: 3, data: []byte{3}},
			},
			offset:          1,
			expectedContent: []byte{1, 0, 3},
			expectHas:       false,
		},
		{
			name: "gap with fetch from gap hit",
			content: []entry{
				{offset: 1, data: []byte{1}},
				{offset: 3, data: []byte{3}},
			},
			offset:          2,
			expectedContent: []byte{0, 3},
			expectHas:       false,
		},
		{
			name: "gap with fetch from gap miss",
			content: []entry{
				{offset: 1, data: []byte{1}},
				{offset: 3, data: []byte{3}},
			},
			offset:          2,
			expectedContent: []byte{0},
			expectHas:       false,
		},
		{
			name: "gap with fetch after gap",
			content: []entry{
				{offset: 1, data: []byte{1}},
				{offset: 3, data: []byte{3}},
			},
			offset:          3,
			expectedContent: []byte{3},
			expectHas:       true,
		},
		{
			name: "gap with fetch before gap",
			content: []entry{
				{offset: 1, data: []byte{1}},
				{offset: 3, data: []byte{3}},
			},
			offset:          1,
			expectedContent: []byte{1},
			expectHas:       true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := store.NewStore[byte]()

			for _, entry := range tc.content {
				s.Set(entry.offset, entry.data)
			}

			data := make([]byte, len(tc.expectedContent))
			complete := s.Get(tc.offset, data)
			assert.Equal(t, tc.expectedContent, data)
			assert.Equal(t, tc.expectHas, complete)

			has := s.Has(tc.offset, len(tc.expectedContent))
			assert.Equal(t, tc.expectHas, has)
		})
	}
}

func BenchmarkStoreSet(b *testing.B) {
	s := store.NewStore[byte]()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := make([]byte, rand.Intn(1<<20)) // 1MiB
		b.StartTimer()
		s.Set(rand.Intn(1<<30), buf) // 1GiB
		b.StopTimer()
	}
}

func TestComplete(t *testing.T) {
	s := store.NewStore[byte](store.WithMinContiguous[byte](1))

	in := []byte{1, 2}

	s.Set(1, in)

	has := s.Has(2, 2)

	assert.False(t, has)
}

func TestSample(t *testing.T) {
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

	// Create an output buffer of 1kiB.
	out := make([]byte, 1<<10)
	// Check if the data would be complete at an offset of 1GiB minus 1MiB
	s.Has(1<<30-1<<20, len(out)) // -> true
	// Retrieve data at an offset of 1GiB.
	has := s.Get(1<<30-1<<20, out) // -> true
	fmt.Printf("For 1KiB at 1GiB-1MiB (complete = %v): %v\n", has, out[:4])

	// Create a new output buffer of 1kiB.
	out = make([]byte, 1<<10)
	// Retrieve data at an offset of 1GiB.
	has = s.Get(1<<30, out) // -> false
	fmt.Printf("For 1KiB at 1GiB (complete = %v): %v\n", has, out[:4])

	assert.Fail(t, "fail")
}
