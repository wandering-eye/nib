package nib

import (
	"fmt"
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
		return Task{}, NoMatchError("no task matches %q", reference)
	case 1:
		return *matches[0], nil
	}

	sort.Slice(matches, func(first, second int) bool {
		return matches[first].UUID < matches[second].UUID
	})

	length := distinguishingLength(matches)

	candidates := make([]string, 0, len(matches))
	for _, task := range matches {
		candidates = append(candidates, fmt.Sprintf("  %s: %s", task.UUID[:length], task.Summary))
	}

	return Task{}, AmbiguousError(
		"%q matches %d tasks; give more of the id:\n%s",
		reference, len(matches), strings.Join(candidates, "\n"),
	)
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
