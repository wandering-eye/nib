package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wandering-eye/nib"
)

func TestShowActive(t *testing.T) {
	repo, cleanup := makeRepo(t)
	defer cleanup()

	program := testCmd(repo)

	output, exiterr, success := program("add", "+one", "one")
	assertProgramResult(t, output, exiterr, success)

	output, exiterr, success = program("start", referenceFor(t, repo, "one"))
	assertProgramResult(t, output, exiterr, success)

	output, exiterr, success = program("show-active")
	assertProgramResult(t, output, exiterr, success)

	var tasks []nib.Task

	tasks = unmarshalTaskArray(t, output)
	assert.Equal(t, "one", tasks[0].Summary, "one should be started")

	output, exiterr, success = program("stop", referenceFor(t, repo, "one"))
	assertProgramResult(t, output, exiterr, success)

	output, exiterr, success = program("show-active")
	assertProgramResult(t, output, exiterr, success)

	tasks = unmarshalTaskArray(t, output)
	assert.Empty(t, tasks, "no tasks should be active")
}
