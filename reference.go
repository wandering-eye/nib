package nib

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// isReferenceShaped reports whether a word on the command line could be a
// leading part of a task UUID: hex digits, with hyphens allowed after the first
// so a UUID can be pasted as is. A leading hyphen is an anti-tag, and "--"
// ignores the context, so neither is ever a reference.
func isReferenceShaped(word string) bool {
	if word == "" || !isHexDigit(rune(word[0])) {
		return false
	}

	for _, character := range word {
		if character != '-' && !isHexDigit(character) {
			return false
		}
	}

	return true
}

func isHexDigit(character rune) bool {
	return ('0' <= character && character <= '9') ||
		('a' <= character && character <= 'f') ||
		('A' <= character && character <= 'F')
}

// stripReference reduces a reference or UUID to the hex digits that are
// compared, so hyphens and case never decide a match.
func stripReference(reference string) string {
	return strings.ToLower(strings.ReplaceAll(reference, "-", ""))
}

// Resolve finds the one loaded task whose UUID starts with reference. It never
// guesses: a reference matching several tasks is refused with every candidate
// listed, because acting on the wrong task is worse than asking again.
func (ts *TaskSet) Resolve(reference string) (Task, error) {
	digits := stripReference(reference)

	if len(digits) < REFERENCE_MIN_LENGTH {
		return Task{}, UsageError(
			"task reference %q is too short: give at least %d characters of the task's id",
			reference, REFERENCE_MIN_LENGTH,
		)
	}

	var matches []*Task

	for _, task := range ts.tasks {
		if strings.HasPrefix(stripReference(task.UUID), digits) {
			matches = append(matches, task)
		}
	}

	switch len(matches) {
	case 0:
		if !ts.resolvedLoaded {
			if err := ts.refuseResolvedMatches(reference, digits); err != nil {
				return Task{}, err
			}
		}

		return Task{}, NoMatchError("no task matches %q", reference)
	case 1:
		return *matches[0], nil
	}

	return Task{}, AmbiguousError(
		"%q matches %d tasks; give more of the id:\n%s",
		reference, len(matches), listCandidates(matches),
	)
}

// refuseResolvedMatches explains a reference that matched no loaded task by
// looking for it among the resolved ones, which were not loaded. Naming the
// resolved task keeps "no task matches" from reading as a mistyped id. It is
// still a reference that matched nothing usable, so the exit code stays
// EXIT_NO_MATCH.
func (ts *TaskSet) refuseResolvedMatches(reference, digits string) error {
	if ts.repoPath == "" {
		return nil
	}

	directory := filepath.Join(ts.repoPath, STATUS_RESOLVED)

	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil
	}

	var matches []*Task

	for _, entry := range entries {
		name := entry.Name()
		if len(name) != TASK_FILENAME_LEN ||
			!strings.HasPrefix(stripReference(name[:36]), digits) {
			continue
		}

		task, err := unmarshalTask(filepath.Join(directory, name), entry, STATUS_RESOLVED)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}

		matches = append(matches, &task)
	}

	switch len(matches) {
	case 0:
		return nil
	case 1:
		return NoMatchError(
			"task %s %q is resolved; this command only applies to open tasks",
			matches[0].ShortUUID(), matches[0].Summary,
		)
	}

	return NoMatchError(
		"%q matches no open task, only %d resolved ones; this command only applies to open tasks:\n%s",
		reference,
		len(matches),
		listCandidates(matches),
	)
}

// listCandidates prints one line per task, sorted by UUID and shown long
// enough that no two lines start alike.
func listCandidates(tasks []*Task) string {
	sort.Slice(tasks, func(first, second int) bool {
		return tasks[first].UUID < tasks[second].UUID
	})

	length := distinguishingLength(tasks)

	candidates := make([]string, 0, len(tasks))
	for _, task := range tasks {
		candidate := fmt.Sprintf("  %s: %s", task.UUID[:length], task.Summary)
		if task.Status == STATUS_RESOLVED {
			candidate += " (resolved)"
		}

		candidates = append(candidates, candidate)
	}

	return strings.Join(candidates, "\n")
}

// MustResolve is Resolve, exiting with the error's code on failure.
func (ts *TaskSet) MustResolve(reference string) Task {
	task, err := ts.Resolve(reference)
	if err != nil {
		ExitError(err)
	}

	return task
}

// distinguishingLength is how much of each UUID to show so no two of the
// sorted tasks print alike: the usual short form, or longer when the reference
// was already that long.
func distinguishingLength(sortedTasks []*Task) int {
	length := SHORT_UUID_LENGTH

	for index := 1; index < len(sortedTasks); index++ {
		previous, current := sortedTasks[index-1].UUID, sortedTasks[index].UUID

		for length < len(current) && previous[:length] == current[:length] {
			length++
		}
	}

	return length
}
