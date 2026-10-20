package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Earlier versions kept the id map in an untracked file under .git, assigned at load
// by scanning for the lowest free integer. Deleting that one file renumbered
// the whole store, and a clone started from nothing. The key belongs to the
// record.
func TestKeysArePersistedInTheRecord(test *testing.T) {
	repo, cleanup := makeRepo(test)
	defer cleanup()

	program := testCmd(repo)
	output, exiterr, success := program("add", "durable")
	assertProgramResult(test, output, exiterr, success)

	matches, err := filepath.Glob(filepath.Join(repo, "pending", "*.yml"))
	assert.NoError(test, err)
	assert.Len(test, matches, 1)

	stored, err := os.ReadFile(matches[0])
	assert.NoError(test, err)
	assert.Contains(test, string(stored), "key: 1", "the key must be on disk, not in a cache")
}

func TestKeysSurviveAClone(test *testing.T) {
	repo, cleanup := makeRepo(test)
	defer cleanup()

	program := testCmd(repo)

	for _, summary := range []string{"one", "two", "three"} {
		output, exiterr, success := program("add", summary)
		assertProgramResult(test, output, exiterr, success)
	}

	output, _, _ := program("next")
	before := unmarshalTaskArray(test, output)

	clone, err := os.MkdirTemp("", "nib-clone-")
	assert.NoError(test, err)

	defer func() { _ = os.RemoveAll(clone) }()

	into := filepath.Join(clone, "copy")
	assert.NoError(test, exec.Command("git", "clone", "--quiet", repo, into).Run())

	output, _, _ = testCmd(into)("next")
	after := unmarshalTaskArray(test, output)

	assert.Len(test, after, len(before))

	for index := range before {
		assert.Equal(test, before[index].ID, after[index].ID,
			"key changed in the clone for %q", before[index].Summary)
	}
}

// Resolving used to blank the id and release it to the next task created.
func TestAResolvedKeyIsNeverReissued(test *testing.T) {
	repo, cleanup := makeRepo(test)
	defer cleanup()

	program := testCmd(repo)

	for _, summary := range []string{"first", "second", "third"} {
		output, exiterr, success := program("add", summary)
		assertProgramResult(test, output, exiterr, success)
	}

	output, exiterr, success := program("done", "2")
	assertProgramResult(test, output, exiterr, success)

	output, exiterr, success = program("add", "fourth")
	assertProgramResult(test, output, exiterr, success)

	output, _, _ = program("next")

	for _, task := range unmarshalTaskArray(test, output) {
		if task.Summary == "fourth" {
			assert.NotEqual(test, 2, task.ID, "key 2 belonged to a resolved task")
			assert.Equal(test, 4, task.ID, "keys continue past the high water mark")
		}
	}

	output, _, _ = program("show-resolved")
	resolved := unmarshalTaskArray(test, output)
	assert.Len(test, resolved, 1)
	assert.Equal(test, 2, resolved[0].ID, "a resolved task keeps the key it was given")
}

// Two records claiming one key cannot be reconciled by guessing, so it is
// reported rather than silently renumbered.
func TestADuplicateKeyIsRefused(test *testing.T) {
	repo, cleanup := makeRepo(test)
	defer cleanup()

	program := testCmd(repo)
	output, exiterr, success := program("add", "original")
	assertProgramResult(test, output, exiterr, success)

	matches, err := filepath.Glob(filepath.Join(repo, "pending", "*.yml"))
	assert.NoError(test, err)
	assert.Len(test, matches, 1)

	stored, err := os.ReadFile(matches[0])
	assert.NoError(test, err)

	twin := filepath.Join(repo, "pending", "aaaaaaaa-1111-2222-3333-444444444444.yml")
	assert.NoError(test, os.WriteFile(twin, stored, 0o600))

	_, exiterr, _ = program("next")
	assert.NotNil(test, exiterr, "a duplicate key must not load")
	assert.NotEqual(test, 0, exiterr.ExitCode())
}

// A store written before keys were recorded carries none. Whatever numbering it is given
// must not let one key name two different records, or `done 8` is ambiguous
// depending on which command the user looked at first.
func TestKeylessStoreNumbersEveryRecordUniquely(test *testing.T) {
	repo, cleanup := makeRepo(test)
	defer cleanup()

	program := testCmd(repo)

	// Interleaved on purpose: a resolved record created before a pending one is
	// what exposes a numbering derived from only part of the store.
	for _, summary := range []string{"r1", "p1", "r2", "p2", "r3", "p3"} {
		output, exiterr, success := program("add", summary)
		assertProgramResult(test, output, exiterr, success)
	}

	for _, key := range []string{"1", "3", "5"} {
		output, exiterr, success := program("done", key)
		assertProgramResult(test, output, exiterr, success)
	}

	// Strip the keys, leaving the store as an earlier version would have written it.
	for _, status := range []string{"pending", "resolved"} {
		matches, err := filepath.Glob(filepath.Join(repo, status, "*.yml"))
		assert.NoError(test, err)

		for _, match := range matches {
			stored, err := os.ReadFile(match)
			assert.NoError(test, err)

			var kept []string

			for _, line := range strings.Split(string(stored), "\n") {
				if !strings.HasPrefix(line, "key:") {
					kept = append(kept, line)
				}
			}

			assert.NoError(test, os.WriteFile(match, []byte(strings.Join(kept, "\n")), 0o600))
		}
	}

	// Such a store has no counter either, so remove it: with one present the
	// numbering starts high enough to hide an ordering mistake.
	assert.NoError(test, os.Remove(filepath.Join(repo, "next-key")))

	owner := map[int]string{}

	for _, command := range []string{"next", "show-resolved"} {
		output, _, _ := program(command)

		for _, task := range unmarshalTaskArray(test, output) {
			if existing, seen := owner[task.ID]; seen && existing != task.UUID {
				test.Fatalf("key %d names both %s and %s", task.ID, existing, task.UUID)
			}

			owner[task.ID] = task.UUID
		}
	}

	assert.Len(test, owner, 6, "every record must hold a distinct key")
}

// Assigning keys in memory was not enough: read commands never save, so the
// next command recomputed them and a record could answer to a different
// number each run. The first load of a keyless store writes them down.
func TestMigratingLoadPersistsKeys(test *testing.T) {
	repo, cleanup := makeRepo(test)
	defer cleanup()

	program := testCmd(repo)

	for _, summary := range []string{"r1", "p1", "r2", "p2"} {
		output, exiterr, success := program("add", summary)
		assertProgramResult(test, output, exiterr, success)
	}

	for _, key := range []string{"1", "3"} {
		output, exiterr, success := program("done", key)
		assertProgramResult(test, output, exiterr, success)
	}

	// Rewrite the store the way an earlier version left it: no keys, no counter, and the
	// summaries changed so re-derived keys cannot coincide with the old ones.
	for _, status := range []string{"pending", "resolved"} {
		matches, err := filepath.Glob(filepath.Join(repo, status, "*.yml"))
		assert.NoError(test, err)

		for _, match := range matches {
			stored, err := os.ReadFile(match)
			assert.NoError(test, err)

			var kept []string

			for _, line := range strings.Split(string(stored), "\n") {
				if !strings.HasPrefix(line, "key:") {
					kept = append(kept, line)
				}
			}

			assert.NoError(test, os.WriteFile(match, []byte(strings.Join(kept, "\n")), 0o600))
		}
	}

	assert.NoError(test, os.Remove(filepath.Join(repo, "next-key")))

	keysFor := func(command string) map[string]int {
		output, _, _ := program(command)
		found := map[string]int{}

		for _, task := range unmarshalTaskArray(test, output) {
			found[task.UUID] = task.ID
		}

		return found
	}

	// Two reads and a write; the numbering has to hold across all three.
	first := keysFor("next")
	assert.NotEmpty(test, first)

	onDisk, err := filepath.Glob(filepath.Join(repo, "pending", "*.yml"))
	assert.NoError(test, err)

	for _, match := range onDisk {
		stored, err := os.ReadFile(match)
		assert.NoError(test, err)
		assert.Contains(test, string(stored), "key:", "%s must have its key written down", match)
	}

	output, exiterr, success := program("add", "forces a save")
	assertProgramResult(test, output, exiterr, success)

	after := keysFor("next")

	for uuid, key := range first {
		assert.Equal(test, key, after[uuid], "key for %s changed after a mutation", uuid)
	}
}
