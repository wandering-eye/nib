package integration

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wandering-eye/nib"
)

func TestSettingTagContext(t *testing.T) {
	repo, cleanup := makeRepo(t)
	defer cleanup()

	program := testCmd(repo)

	output, exiterr, success := program("add", "+one", "one")
	assertProgramResult(t, output, exiterr, success)

	output, exiterr, success = program("add", "+one", "one")
	assertProgramResult(t, output, exiterr, success)

	output, exiterr, success = program("add", "+two", "two")
	assertProgramResult(t, output, exiterr, success)

	output, exiterr, success = program("context", "+two")
	assertProgramResult(t, output, exiterr, success)

	output, exiterr, success = program("next")
	assertProgramResult(t, output, exiterr, success)

	var tasks []nib.Task

	tasks = unmarshalTaskArray(t, output)
	assert.Equal(t, "two", tasks[0].Summary, "setting +two as a context")

	output, exiterr, success = program("context", "-one")
	assertProgramResult(t, output, exiterr, success)

	output, exiterr, success = program("next")
	assertProgramResult(t, output, exiterr, success)

	tasks = unmarshalTaskArray(t, output)
	assert.Equal(t, "two", tasks[0].Summary, "setting -one as a context")
}

func TestSettingTagAndProjectContext(t *testing.T) {
	repo, cleanup := makeRepo(t)
	defer cleanup()

	program := testCmd(repo)

	output, exiterr, success := program("add", "+one", "+alpha", "one")
	assertProgramResult(t, output, exiterr, success)

	output, exiterr, success = program("add", "project:beta", "+two", "two")
	assertProgramResult(t, output, exiterr, success)

	output, exiterr, success = program("context", "project:beta")
	assertProgramResult(t, output, exiterr, success)

	output, exiterr, success = program("next")
	assertProgramResult(t, output, exiterr, success)

	var tasks []nib.Task

	tasks = unmarshalTaskArray(t, output)
	assert.Equal(t, "two", tasks[0].Summary, "setting project:beta as a context")

	output, exiterr, success = program("context", "project:beta", "+one")
	assertProgramResult(t, output, exiterr, success)

	output, exiterr, success = program("next")
	assertProgramResult(t, output, exiterr, success)

	tasks = unmarshalTaskArray(t, output)
	assert.Empty(t, tasks, "no tasks within context project:beta +one")
}

func TestContextFromEnvVar(t *testing.T) {
	repo, cleanup := makeRepo(t)
	defer cleanup()

	program := testCmd(repo)

	output, exiterr, success := program("add", "+one", "+alpha", "one")
	assertProgramResult(t, output, exiterr, success)

	output, exiterr, success = program("add", "project:beta", "+two", "two")
	assertProgramResult(t, output, exiterr, success)

	output, exiterr, success = program("context", "project:beta")
	assertProgramResult(t, output, exiterr, success)

	// override context with an env var
	unsetEnv := setEnv("NIB_CONTEXT", "+one +alpha")

	t.Logf("NIB_CONTEXT=%s", os.Getenv("NIB_CONTEXT"))

	output, exiterr, success = program("next")
	assertProgramResult(t, output, exiterr, success)

	var tasks []nib.Task

	tasks = unmarshalTaskArray(t, output)
	assert.Equal(t, "one", tasks[0].Summary, "'+one +alpha' context set by NIB_CONTEXT ")

	// unset the context override, so we expect to use the on-disk context
	unsetEnv()

	output, exiterr, success = program("next")
	assertProgramResult(t, output, exiterr, success)

	tasks = unmarshalTaskArray(t, output)
	assert.Equal(t, "two", tasks[0].Summary, "project:beta is on-disk context")
}
