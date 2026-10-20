package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoteAppendTextWithoutTTY(t *testing.T) {
	repo, cleanup := makeRepo(t)
	defer cleanup()

	program := testCmd(repo)

	output, exiterr, success := program("add", "test task")
	assertProgramResult(t, output, exiterr, success)

	_, exiterr, success = program("note", "1", "first line")
	assertProgramResult(t, nil, exiterr, success)

	output, exiterr, success = program("show-open")
	assertProgramResult(t, output, exiterr, success)

	tasks := unmarshalTaskArray(t, output)
	require.Len(t, tasks, 1)
	assert.Equal(t, "first line", tasks[0].Notes)

	_, exiterr, success = program("note", "1", "second line")
	assertProgramResult(t, nil, exiterr, success)

	output, exiterr, success = program("show-open")
	assertProgramResult(t, output, exiterr, success)

	tasks = unmarshalTaskArray(t, output)
	require.Len(t, tasks, 1)
	assert.Equal(t, "first line\nsecond line", tasks[0].Notes)
}
