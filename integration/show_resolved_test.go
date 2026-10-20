package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wandering-eye/nib"
)

func TestShowResolved(t *testing.T) {
	repo, cleanup := makeRepo(t)
	defer cleanup()

	program := testCmd(repo)

	output, exiterr, success := program("add", "+one", "one")
	assertProgramResult(t, output, exiterr, success)

	output, exiterr, success = program("add", "+two", "two")
	assertProgramResult(t, output, exiterr, success)

	output, exiterr, success = program("add", "three")
	assertProgramResult(t, output, exiterr, success)

	output, exiterr, success = program(referenceFor(t, repo, "one"), "done")
	assertProgramResult(t, output, exiterr, success)

	output, exiterr, success = program("show-resolved")
	assertProgramResult(t, output, exiterr, success)

	var tasks []nib.Task

	tasks = unmarshalTaskArray(t, output)
	assert.Equal(t, "one", tasks[0].Summary, "one should be resolved")

	// Test the sorting of resolved tasks
	_, exiterr, success = program(referenceFor(t, repo, "three"), "done")
	assertProgramResult(t, output, exiterr, success)

	_, exiterr, success = program(referenceFor(t, repo, "two"), "done")
	assertProgramResult(t, output, exiterr, success)

	output, exiterr, success = program("show-resolved")
	assertProgramResult(t, output, exiterr, success)

	// sorting is ascending, so the most recently resolved tasks are shown last
	// (visible in terminal)
	tasks = unmarshalTaskArray(t, output)
	assert.Equal(t, "one", tasks[0].Summary, "one should be resolved")

	var zeroValue time.Time

	assert.True(
		t,
		tasks[0].Resolved.After(zeroValue),
		"resolved time should not be 0 value for time.Time",
	)
}
