package nib

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Tasks created in the same instant must sort the same way whatever order they
// were read in.
func TestSortByCreatedBreaksTiesByUUID(t *testing.T) {
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	ts := TaskSet{tasks: []*Task{
		{UUID: "cccc", Created: created},
		{UUID: "aaaa", Created: created},
		{UUID: "bbbb", Created: created.Add(-time.Second)},
		{UUID: "dddd", Created: created},
	}}

	ts.SortByCreated(Ascending)

	var order []string
	for _, task := range ts.tasks {
		order = append(order, task.UUID)
	}

	assert.Equal(t, []string{"bbbb", "aaaa", "cccc", "dddd"}, order)

	ts.SortByCreated(Descending)

	order = order[:0]
	for _, task := range ts.tasks {
		order = append(order, task.UUID)
	}

	assert.Equal(t, []string{"aaaa", "cccc", "dddd", "bbbb"}, order)
}
