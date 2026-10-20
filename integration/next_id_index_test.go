package integration

import (
	"testing"

	"gotest.tools/assert"
)

func TestNextByReference(t *testing.T) {
	repo, cleanup := makeRepo(t)
	defer cleanup()

	program := testCmd(repo)

	output, exiterr, success := program("add", "one")
	assertProgramResult(t, output, exiterr, success)

	output, exiterr, success = program("add", "two", "+two")
	assertProgramResult(t, output, exiterr, success)

	output, exiterr, success = program(referenceFor(t, repo, "one"))
	assertProgramResult(t, output, exiterr, success)

	tasks := unmarshalTaskArray(t, output)
	assert.Equal(t, "one", tasks[0].Summary, "find task one by reference")
}

func TestNextByReferenceOutsideContext(t *testing.T) {
	repo, cleanup := makeRepo(t)
	defer cleanup()

	program := testCmd(repo)

	output, exiterr, success := program("add", "one", "+one")
	assertProgramResult(t, output, exiterr, success)

	output, exiterr, success = program("add", "two", "+two")
	assertProgramResult(t, output, exiterr, success)

	output, exiterr, success = program("context", "+one")
	assertProgramResult(t, output, exiterr, success)

	output, exiterr, success = program(referenceFor(t, repo, "two"))
	assertProgramResult(t, output, exiterr, success)

	tasks := unmarshalTaskArray(t, output)
	assert.Equal(
		t,
		"two",
		tasks[0].Summary,
		"find task two by reference (context ignored when addressing a task directly)",
	)

	output, exiterr, success = program("next")
	assertProgramResult(t, output, exiterr, success)

	tasks = unmarshalTaskArray(t, output)
	assert.Equal(t, "one", tasks[0].Summary, "one is the only task in our current context")
}
