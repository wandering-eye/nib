package nib

// main task data structures

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"slices"
	"sort"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"
)

type SubTask struct {
	Summary  string
	Resolved bool
}

// Task is our representation of tasks added at the command line and serialized
// to the task database on disk. It is rendered in multiple ways by the TaskSet
// to which it belongs.
type Task struct {
	// not stored in file -- rather filename and directory
	UUID   string `json:"uuid"   yaml:"-"` // TODO: use actual uuid.UUID type here
	Status string `json:"status" yaml:",omitempty"`
	// is new or has changed. Need to write to disk.
	WritePending bool `json:"-"      yaml:"-"`

	// LegacyKey is the numeric key earlier versions wrote into every record.
	// It is read only so the load can tell a record still carries one; it is
	// cleared before anything is written, so it never reaches disk again.
	LegacyKey int `json:"-" yaml:"key,omitempty"`

	// Deleted, if true, marks this task for deletion
	Deleted bool `json:"-" yaml:"-"`

	// concise representation of task
	Summary string `json:"summary"`
	// more detail, or information to remember to complete the task
	Notes   string   `json:"notes"`
	Tags    []string `json:"tags"`
	Project string   `json:"project"`
	// see const.go for PRIORITY_ strings
	Priority    string    `json:"priority"`
	DelegatedTo string    `json:"-"`
	Subtasks    []SubTask `json:"-"`
	// uuids of tasks that this task depends on
	// blocked status can be derived.
	// TODO possible filter: :blocked. Also, :overdue
	Dependencies []string `json:"-"`

	Created  time.Time `json:"created"`
	Resolved time.Time `json:"resolved"`
	Due      time.Time `json:"due"`

	// TaskSet uses this to indicate if a given task is excluded by a filter
	// (context etc)
	filtered bool `json:"-"`
}

// Equals returns whether t2 equals task.
// for equality, we only consider "core properties", we ignore WritePending, LegacyKey, Deleted and filtered.
func (t Task) Equals(t2 Task) bool {
	if t2.UUID != t.UUID {
		return false
	}

	if t2.Status != t.Status {
		return false
	}

	if t2.Summary != t.Summary {
		return false
	}

	if t2.Notes != t.Notes {
		return false
	}

	if !reflect.DeepEqual(t.Tags, t2.Tags) {
		return false
	}

	if t2.Project != t.Project {
		return false
	}

	if t2.Priority != t.Priority {
		return false
	}

	if t2.DelegatedTo != t.DelegatedTo {
		return false
	}

	if !reflect.DeepEqual(t.Subtasks, t2.Subtasks) {
		return false
	}

	if !reflect.DeepEqual(t.Dependencies, t2.Dependencies) {
		return false
	}

	if !t2.Created.Equal(t.Created) || !t2.Resolved.Equal(t.Resolved) || !t2.Due.Equal(t.Due) {
		return false
	}

	return true
}

// Unmarshal a Task from disk. We explicitly pass status, because the caller
// already knows the status, and can override the status declared in yaml.
func unmarshalTask(path string, finfo os.DirEntry, status string) (Task, error) {
	if len(finfo.Name()) != TASK_FILENAME_LEN {
		return Task{}, fmt.Errorf("filename does not encode UUID %s (wrong length)", finfo.Name())
	}

	uuid := finfo.Name()[0:36]
	if !IsValidUUID4String(uuid) {
		return Task{}, fmt.Errorf("filename does not encode UUID %s", finfo.Name())
	}

	t := Task{
		UUID:   uuid,
		Status: status,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Task{}, fmt.Errorf("failed to read %s", finfo.Name())
	}

	err = yaml.Unmarshal(data, &t)
	if err != nil {
		return Task{}, fmt.Errorf("failed to unmarshal %s", finfo.Name())
	}

	t.Status = status

	return t, nil
}

func (t Task) String() string {
	if t.UUID != "" {
		return fmt.Sprintf("%s: %s", t.ShortUUID(), t.Summary)
	}

	return t.Summary
}

// ShortUUID is the leading part of the UUID shown wherever a task is listed.
// Any prefix of at least REFERENCE_MIN_LENGTH characters refers to the task.
func (t Task) ShortUUID() string {
	if len(t.UUID) < SHORT_UUID_LENGTH {
		return t.UUID
	}

	return t.UUID[:SHORT_UUID_LENGTH]
}

func (t *Task) MatchesFilter(query Query) bool {
	for _, tag := range query.Tags {
		if !StrSliceContains(t.Tags, tag) {
			return false
		}
	}

	for _, tag := range query.AntiTags {
		if StrSliceContains(t.Tags, tag) {
			return false
		}
	}

	if StrSliceContains(query.AntiProjects, t.Project) {
		return false
	}

	if query.Project != "" && t.Project != query.Project {
		return false
	}

	if !query.Due.IsZero() {
		if t.Due.IsZero() {
			return false
		}
		if query.DateFilter == "after" && t.Due.Before(query.Due) {
			return false
		}
		if query.DateFilter == "before" && t.Due.After(query.Due) {
			return false
		}
		if (query.DateFilter == "on" || query.DateFilter == "in") && !t.Due.Equal(query.Due) {
			return false
		}
		if query.DateFilter == "" && !t.Due.Equal(query.Due) {
			return false
		}
	}

	if query.Priority != "" && t.Priority != query.Priority {
		return false
	}

	if query.Text != "" &&
		!strings.Contains(strings.ToLower(t.Summary+t.Notes), strings.ToLower(query.Text)) {
		return false
	}

	return true
}

// Normalise mutates and sorts some of a task object's fields into a consistent
// format. This should make git diffs more useful.
func (t *Task) Normalise() {
	// Stored instants are UTC. Local time is an input and display concern:
	// "tomorrow" is resolved against the user's day, then converted here, so
	// two machines in different zones agree on what is on disk.
	t.Created = t.Created.UTC()
	t.Resolved = t.Resolved.UTC()
	t.Due = t.Due.UTC()

	t.Project = strings.ToLower(t.Project)

	// tags must be lowercase
	for i, tag := range t.Tags {
		t.Tags[i] = strings.ToLower(tag)
	}

	// tags must be sorted
	sort.Strings(t.Tags)

	// tags must be unique
	t.Tags = DeduplicateStrings(t.Tags)

	if t.Priority == "" {
		t.Priority = PRIORITY_NORMAL
	}

	t.LegacyKey = 0
}

// normalise the task before validating!
func (t *Task) Validate() error {
	if !IsValidUUID4String(t.UUID) {
		return errors.New("invalid task UUID4")
	}

	if !IsValidStatus(t.Status) {
		return errors.New("invalid status specified on task")
	}

	if !IsValidPriority(t.Priority) {
		return errors.New("invalid priority specified")
	}

	for _, uuid := range t.Dependencies {
		if !IsValidUUID4String(uuid) {
			return errors.New("invalid dependency UUID4")
		}
	}

	return nil
}

// provides Summary + Last note if available.
func (t *Task) LongSummary() string {
	notes := strings.TrimSpace(t.Notes)
	noteLines := strings.Split(notes, "\n")
	lastNote := noteLines[len(noteLines)-1]

	if len(lastNote) > 0 {
		return t.Summary + " " + NOTE_MODE_KEYWORD + " " + lastNote
	}

	return t.Summary
}

func (t *Task) Modify(query Query) {
	for _, tag := range query.Tags {
		if !StrSliceContains(t.Tags, tag) {
			t.Tags = append(t.Tags, tag)
		}
	}

	for i, tag := range t.Tags {
		if StrSliceContains(query.AntiTags, tag) {
			// delete item
			t.Tags = slices.Delete(t.Tags, i, i+1)
		}
	}

	if query.Project != "" {
		t.Project = query.Project
	}

	if StrSliceContains(query.AntiProjects, t.Project) {
		t.Project = ""
	}

	if query.Priority != "" {
		t.Priority = query.Priority
	}

	if !query.Due.IsZero() {
		t.Due = query.Due
	}

	if t.Notes != "" {
		t.Notes += "\n"
	}

	t.Notes += query.Note
}

func (t *Task) SaveToDisk(repoPath string) {
	// save should be idempotent
	t.WritePending = false

	filepath := MustGetRepoPath(repoPath, t.Status, t.UUID+".yml")

	if t.Deleted {
		// Task is marked deleted. Delete from its current status directory.
		if err := os.Remove(filepath); err != nil {
			ExitFail("Could not remove task %s: %v", filepath, err)
		}
	} else if err := writeRecord(filepath, *t); err != nil {
		ExitFail("Failed to write task %s: %v", t, err)
	}

	// Delete task from other status directories. Only one copy should exist, at most.
	for _, st := range ALL_STATUSES {
		if st == t.Status {
			continue
		}

		filepath := MustGetRepoPath(repoPath, st, t.UUID+".yml")

		if _, err := os.Stat(filepath); !os.IsNotExist(err) {
			err := os.Remove(filepath)
			if err != nil {
				ExitFail("Could not remove task %s: %v", filepath, err)
			}
		}
	}
}

// writeRecord serialises a task to the given path. The status is the
// directory the record lives in, so it is left out of the file.
func writeRecord(path string, task Task) error {
	task.Status = ""

	data, err := yaml.Marshal(&task)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

// ParseDueDateToStr renders a due date the way a person reads it. The stored
// instant is UTC, so it is converted back to local first: "tomorrow" is a
// property of the reader's day, and comparing a UTC instant against a local
// now puts every due date a day out for anyone east of UTC.
func (t *Task) ParseDueDateToStr() string {
	due := t.Due.Local()
	now := time.Now()

	if due.IsZero() {
		return ""
	}

	if due.YearDay() == now.YearDay() && due.Year() == now.Year() {
		return "today"
	}

	tomorrow := now.AddDate(0, 0, 1)
	if due.YearDay() == tomorrow.YearDay() && due.Year() == tomorrow.Year() {
		return "tomorrow"
	}

	yesterday := now.AddDate(0, 0, -1)
	if due.YearDay() == yesterday.YearDay() && due.Year() == yesterday.Year() {
		return "yesterday"
	}

	return formatDueDate(due)
}

func formatDueDate(due time.Time) string {
	due = due.Local()
	now := time.Now()
	year, _, _ := now.Date()
	dueYear, _, _ := due.Date()

	switch {
	case due.After(now) && due.Sub(startOfDay(now)) < 7*24*time.Hour:
		return due.Format("Mon _2")
	case dueYear == year:
		return due.Format("_2 Jan")
	default:
		return due.Format("_2 Jan 2006")
	}
}
