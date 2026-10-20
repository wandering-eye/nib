package nib

import (
	"errors"
	"os"
	"path/filepath"
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

// makeStoreWithResolvedTwin writes one open and one resolved task sharing the
// prefix aaaa1111, and a resolved task alone under dddd4444.
func makeStoreWithResolvedTwin(t *testing.T) string {
	t.Helper()

	repo := t.TempDir()

	for _, record := range []struct{ status, uuid, summary string }{
		{STATUS_PENDING, "aaaa1111-0000-4000-8000-000000000001", "open twin"},
		{STATUS_RESOLVED, "aaaa1111-0000-4000-8000-000000000002", "resolved twin"},
		{STATUS_RESOLVED, "dddd4444-0000-4000-8000-000000000004", "finished work"},
	} {
		require.NoError(t, os.MkdirAll(filepath.Join(repo, record.status), 0o700))
		require.NoError(t, writeRecord(
			filepath.Join(repo, record.status, record.uuid+".yml"),
			Task{UUID: record.uuid, Summary: record.summary, Priority: PRIORITY_NORMAL},
		))
	}

	return repo
}

func TestResolveNamesAResolvedTaskWhenOnlyOpenTasksAreLoaded(t *testing.T) {
	ts, err := LoadTaskSet(makeStoreWithResolvedTwin(t), false)
	require.NoError(t, err)

	_, err = ts.Resolve("dddd4444")
	require.Error(t, err)
	assert.Equal(t, EXIT_NO_MATCH, exitCodeOf(t, err))
	assert.Equal(
		t,
		`task dddd4444 "finished work" is resolved; this command only applies to open tasks`,
		err.Error(),
	)

	task, err := ts.Resolve("aaaa1111")
	require.NoError(t, err, "a resolved task that was not loaded never makes a match ambiguous")
	assert.Equal(t, "open twin", task.Summary)

	_, err = ts.Resolve("eeee")
	require.Error(t, err)
	assert.Equal(t, EXIT_NO_MATCH, exitCodeOf(t, err))
	assert.Equal(t, `no task matches "eeee"`, err.Error())
}

func TestResolveAcrossEveryStatusMarksResolvedCandidates(t *testing.T) {
	ts, err := LoadTaskSet(makeStoreWithResolvedTwin(t), true)
	require.NoError(t, err)

	task, err := ts.Resolve("dddd4444")
	require.NoError(t, err)
	assert.Equal(t, STATUS_RESOLVED, task.Status)

	_, err = ts.Resolve("aaaa1111")
	require.Error(t, err)
	assert.Equal(t, EXIT_AMBIGUOUS, exitCodeOf(t, err))
	assert.Contains(t, err.Error(), "aaaa1111-0000-4000-8000-000000000001: open twin\n")
	assert.Contains(
		t,
		err.Error(),
		"aaaa1111-0000-4000-8000-000000000002: resolved twin (resolved)",
	)
}

func TestResolveListsEveryResolvedTaskAPrefixMatches(t *testing.T) {
	repo := makeStoreWithResolvedTwin(t)
	require.NoError(t, writeRecord(
		filepath.Join(repo, STATUS_RESOLVED, "dddd4444-0000-4000-8000-000000000005.yml"),
		Task{
			UUID:     "dddd4444-0000-4000-8000-000000000005",
			Summary:  "also finished",
			Priority: PRIORITY_NORMAL,
		},
	))

	ts, err := LoadTaskSet(repo, false)
	require.NoError(t, err)

	_, err = ts.Resolve("dddd")
	require.Error(t, err)
	assert.Equal(t, EXIT_NO_MATCH, exitCodeOf(t, err))
	assert.Contains(t, err.Error(), "only 2 resolved ones")
	assert.Contains(
		t,
		err.Error(),
		"dddd4444-0000-4000-8000-000000000004: finished work (resolved)",
	)
	assert.Contains(
		t,
		err.Error(),
		"dddd4444-0000-4000-8000-000000000005: also finished (resolved)",
	)
}
