package nib

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeReferenceTaskSet(t *testing.T) *TaskSet {
	t.Helper()

	ts := &TaskSet{tasksByUUID: map[string]*Task{}}

	for uuid, summary := range map[string]string{
		"aaaa1111-0000-4000-8000-000000000001": "first",
		"aaaa1111-0000-4000-8000-000000000002": "second",
		"bbbb2222-0000-4000-8000-000000000003": "third",
	} {
		_, err := ts.LoadTask(Task{UUID: uuid, Status: STATUS_PENDING, Summary: summary})
		require.NoError(t, err)
	}

	return ts
}

func exitCodeOf(t *testing.T, err error) int {
	t.Helper()

	var commandErr CommandError
	require.True(t, errors.As(err, &commandErr), "%v carries no exit code", err)

	return commandErr.Code
}

func TestResolveAUniquePrefix(t *testing.T) {
	ts := makeReferenceTaskSet(t)

	for _, reference := range []string{
		"bbbb",
		"bbbb2222",
		"BBBB2222",
		"bbbb2222-0000",
		"bbbb22220000",
		"bbbb2222-0000-4000-8000-000000000003",
	} {
		task, err := ts.Resolve(reference)
		require.NoError(t, err, reference)
		assert.Equal(t, "third", task.Summary, reference)
	}

	task, err := ts.Resolve("aaaa1111-0000-4000-8000-000000000002")
	require.NoError(t, err)
	assert.Equal(t, "second", task.Summary, "a full UUID is never ambiguous")
}

func TestResolveNothingIsNoMatch(t *testing.T) {
	ts := makeReferenceTaskSet(t)

	_, err := ts.Resolve("cccc")
	require.Error(t, err)
	assert.Equal(t, EXIT_NO_MATCH, exitCodeOf(t, err))
}

func TestResolveAnAmbiguousPrefixListsCandidates(t *testing.T) {
	ts := makeReferenceTaskSet(t)

	_, err := ts.Resolve("aaaa1111")
	require.Error(t, err)
	assert.Equal(t, EXIT_AMBIGUOUS, exitCodeOf(t, err))

	// Eight characters would print both candidates alike, so the listing
	// grows until they differ.
	assert.Contains(t, err.Error(), "aaaa1111-0000-4000-8000-000000000001: first")
	assert.Contains(t, err.Error(), "aaaa1111-0000-4000-8000-000000000002: second")
	assert.NotContains(t, err.Error(), "third")
}

func TestResolveATooShortPrefixIsUsage(t *testing.T) {
	ts := makeReferenceTaskSet(t)

	for _, reference := range []string{"b", "bbb", "b-b-b"} {
		_, err := ts.Resolve(reference)
		require.Error(t, err, reference)
		assert.Equal(t, EXIT_USAGE, exitCodeOf(t, err), reference)
	}
}
