package nib

// main task data structures

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"
)

type TaskSet struct {
	tasks []*Task

	tasksByUUID map[string]*Task

	repoPath string
}

type Project struct {
	Name          string `json:"name"`
	Tasks         int    `json:"taskCount"`
	TasksResolved int    `json:"resolvedCount"`
	// if any task is in the active state
	Active bool `json:"active"`
	// first task created
	Created time.Time `json:"created"`
	// last task resolved
	Resolved time.Time `json:"resolved"`

	// highest non-resolved priority within project
	Priority string `json:"priority"`
}

// LoadTaskSet constructs a TaskSet from a repo path..
func LoadTaskSet(repoPath string, includeResolved bool) (*TaskSet, error) {
	// Initialise an empty TaskSet
	var ts TaskSet
	ts.tasksByUUID = make(map[string]*Task)

	ts.repoPath = repoPath

	var statuses []string

	var keyed bool

	if includeResolved {
		// expensive to load -- resolved tasks are unbounded
		statuses = ALL_STATUSES
	} else {
		// non-resolved tasks are bounded, so it's OK to load them even if
		// some are redundant due to query.
		statuses = NON_RESOLVED_STATUSES
	}

	for _, status := range statuses {
		dir := filepath.Join(repoPath, status)

		files, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				// Continuing here is necessary, because we do not guarantee
				// that all status directories exist on program startup.
				continue
			}

			return nil, err
		}

		for _, finfo := range files {
			// Discard hidden files like .gitkeep
			if strings.HasPrefix(finfo.Name(), ".") {
				continue
			}

			path := filepath.Join(dir, finfo.Name())

			// A record that cannot be read is not skipped. Logging and
			// carrying on meant a command listed fewer tasks than the store
			// held and still reported success.
			t, err := unmarshalTask(path, finfo, status)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", finfo.Name(), err)
			}

			if t.LegacyKey != 0 {
				keyed = true
			}

			if _, err := ts.LoadTask(t); err != nil {
				return nil, fmt.Errorf("%s: %w", finfo.Name(), err)
			}
		}
	}

	// The counter is what every store from the numeric-key versions has, so it
	// marks one still to migrate, including one that an older clone has just
	// synced keys back into.
	if keyed || fileExists(filepath.Join(repoPath, LEGACY_KEY_FILE)) {
		if err := removeNumericKeys(repoPath); err != nil {
			fmt.Fprintf(os.Stderr, "could not finish removing numeric task keys: %v\n", err)
		}
	}

	// hide some tasks by default. This is useful for things like templates and
	// recurring tasks which are shown either directly or with show- commands
	for _, task := range ts.tasks {
		if StrSliceContains(HIDDEN_STATUSES, task.Status) {
			task.filtered = true
		}
	}

	return &ts, nil
}

func (ts *TaskSet) UnHide() {
	for _, task := range ts.tasks {
		if StrSliceContains(HIDDEN_STATUSES, task.Status) {
			task.filtered = false
		}
	}
}

// SortByCreated orders by creation time, then UUID, so tasks created in the
// same instant still come out the same way whichever directory they were read
// from.
func (ts *TaskSet) SortByCreated(dir SortByDirection) {
	ts.sortByTime(dir, func(task *Task) time.Time { return task.Created })
}

func (ts *TaskSet) SortByPriority(dir SortByDirection) {
	switch dir {
	case Ascending:
		// P1 first
		sort.SliceStable(
			ts.tasks,
			func(i, j int) bool { return ts.tasks[i].Priority < ts.tasks[j].Priority },
		)
	case Descending:
		// P1 last
		sort.SliceStable(
			ts.tasks,
			func(i, j int) bool { return ts.tasks[i].Priority > ts.tasks[j].Priority },
		)
	}
}

func (ts *TaskSet) SortByResolved(dir SortByDirection) {
	ts.sortByTime(dir, func(task *Task) time.Time { return task.Resolved })
}

func (ts *TaskSet) sortByTime(dir SortByDirection, timeOf func(*Task) time.Time) {
	sort.SliceStable(ts.tasks, func(first, second int) bool {
		firstTime, secondTime := timeOf(ts.tasks[first]), timeOf(ts.tasks[second])

		if !firstTime.Equal(secondTime) {
			if dir == Descending {
				return firstTime.After(secondTime)
			}

			return firstTime.Before(secondTime)
		}

		return ts.tasks[first].UUID < ts.tasks[second].UUID
	})
}

// MustLoadTask is the same as LoadTask, except it exits on error.
func (ts *TaskSet) MustLoadTask(task Task) Task {
	newTask, err := ts.LoadTask(task)
	if err != nil {
		ExitFail("%s, task %s", err, task.UUID)
	}

	return newTask
}

// LoadTask adds a task to the TaskSet, but only if it has a new uuid or no uuid.
// Return annotated task.
func (ts *TaskSet) LoadTask(task Task) (Task, error) {
	task.Normalise()

	if task.UUID == "" {
		task.UUID = MustGetUUID4String()
	}

	if err := task.Validate(); err != nil {
		return Task{}, err
	}

	if ts.tasksByUUID[task.UUID] != nil {
		// load tasks, do not overwrite
		// TODO ??? (maybe return a nil pointer instead?)
		return Task{}, nil
	}

	if task.Created.IsZero() {
		task.Created = time.Now().UTC()
		task.WritePending = true
	}

	ts.tasks = append(ts.tasks, &task)
	ts.tasksByUUID[task.UUID] = &task

	return task, nil
}

// TODO maybe this is the place to check for invalid state transitions instead
// of the main switch statement. Though, a future 3rdparty sync system could
// need this to work regardless.
func (ts *TaskSet) MustUpdateTask(task Task) {
	if err := ts.UpdateTask(task); err != nil {
		ExitFail(err.Error())
	}
}

func (ts *TaskSet) UpdateTask(task Task) error {
	task.Normalise()

	if err := task.Validate(); err != nil {
		return fmt.Errorf("%w, task %s", err, task.UUID)
	}

	if ts.tasksByUUID[task.UUID] == nil {
		return errors.New("could not find given task to update by UUID")
	}

	if !IsValidPriority(task.Priority) {
		return errors.New("invalid priority specified")
	}

	old := ts.tasksByUUID[task.UUID]

	if old.Status != task.Status && !IsValidStateTransition(old.Status, task.Status) {
		return fmt.Errorf("invalid state transition: %s -> %s", old.Status, task.Status)
	}

	if old.Status != task.Status && task.Status == STATUS_RESOLVED &&
		strings.Contains(task.Notes, "- [ ] ") {
		return errors.New("refusing to resolve task with incomplete tasklist")
	}

	if task.Status == STATUS_RESOLVED && task.Resolved.IsZero() {
		task.Resolved = time.Now().UTC()
	}

	task.WritePending = true
	// existing pointer must point to address of new task copied
	*ts.tasksByUUID[task.UUID] = task

	return nil
}

// Filter hides every task the query does not match. References in the query
// select tasks by identity first; one that resolves to no task, or to several,
// is an error rather than an empty result.
func (ts *TaskSet) Filter(query Query) error {
	referenced := make(map[string]bool, len(query.References))

	for _, reference := range query.References {
		task, err := ts.Resolve(reference)
		if err != nil {
			return err
		}

		referenced[task.UUID] = true
	}

	for _, task := range ts.tasks {
		if len(referenced) > 0 && !referenced[task.UUID] || !task.MatchesFilter(query) {
			task.filtered = true
		}
	}

	return nil
}

func (ts *TaskSet) FilterByStatus(status string) {
	for _, task := range ts.tasks {
		if task.Status != status {
			task.filtered = true
		}
	}
}

func (ts *TaskSet) FilterOrganised() {
	for _, task := range ts.tasks {
		if len(task.Tags) > 0 || task.Project != "" {
			task.filtered = true
		}
	}
}

func (ts *TaskSet) Tasks() []Task {
	tasks := make([]Task, 0, len(ts.tasks))
	for _, task := range ts.tasks {
		if !task.filtered {
			tasks = append(tasks, *task)
		}
	}

	return tasks
}

func (ts *TaskSet) AllTasks() []Task {
	tasks := make([]Task, 0, len(ts.tasks))
	for _, task := range ts.tasks {
		tasks = append(tasks, *task)
	}

	return tasks
}

func (ts *TaskSet) GetTags() map[string]bool {
	tagset := make(map[string]bool)

	for _, task := range ts.Tasks() {
		for _, tag := range task.Tags {
			tagset[tag] = true
		}
	}

	return tagset
}

func (ts *TaskSet) GetProjects() []*Project {
	projectsMap := make(map[string]*Project)

	for _, task := range ts.AllTasks() {
		name := task.Project

		if name == "" {
			continue
		}

		if projectsMap[name] == nil {
			projectsMap[name] = &Project{
				Name:     name,
				Priority: PRIORITY_LOW,
			}
		}

		project := projectsMap[name]

		project.Tasks++

		if project.Created.IsZero() || task.Created.Before(project.Created) {
			project.Created = task.Created
		}

		if task.Resolved.After(project.Resolved) {
			project.Resolved = task.Resolved
		}

		if task.Status == STATUS_RESOLVED {
			project.TasksResolved++
		}

		if task.Status == STATUS_ACTIVE {
			project.Active = true
		}

		if task.Status != STATUS_RESOLVED && task.Priority < project.Priority {
			project.Priority = task.Priority
		}
	}

	// collect keys to produce ordered output (rather than randomised)
	names := make([]string, 0, len(projectsMap))
	projects := make([]*Project, 0, len(projectsMap))

	for name := range projectsMap {
		names = append(names, name)
	}

	sort.Strings(names)

	for _, name := range names {
		projects = append(projects, projectsMap[name])
	}

	return projects
}

func (ts *TaskSet) NumTotal() int {
	return len(ts.tasks)
}

// save pending changes to disk
// TODO return files that have been added/deleted/modified/renamed so they can
// be passed to git add for performance, instead of doing git add .
func (ts *TaskSet) SavePendingChanges() {
	for _, task := range ts.tasks {
		if task.WritePending {
			task.SaveToDisk(ts.repoPath)
		}
	}
}

type SortByDirection string

const (
	Ascending  SortByDirection = "ascending"
	Descending SortByDirection = "descending"
)

// legacyKeyLine is the top-level key entry earlier versions wrote. Only a
// top-level entry can start at column 0, so this never matches inside notes.
var legacyKeyLine = regexp.MustCompile(`(?m)^key:[ \t]*[0-9]+[ \t]*(\r?\n|\z)`)

// removeNumericKeys rewrites every record that still carries a numeric key
// without it, deletes the counter, and commits both as one change. Every
// status is swept, not only the ones this command loads, so resolved records
// lose their keys too. Each record is rewritten where it lies and nothing else
// is touched, so running it again on a migrated store changes nothing.
func removeNumericKeys(repoPath string) error {
	for _, status := range ALL_STATUSES {
		entries, err := os.ReadDir(filepath.Join(repoPath, status))
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			path := filepath.Join(repoPath, status, entry.Name())

			// A record that cannot be read is reported by the load that needs
			// it; it is no reason to leave the others keyed.
			task, err := unmarshalTask(path, entry, status)
			if err != nil || task.LegacyKey == 0 {
				continue
			}

			if err := removeKeyFromRecord(path, task); err != nil {
				return fmt.Errorf("%s: %w", entry.Name(), err)
			}
		}
	}

	if err := os.Remove(filepath.Join(repoPath, LEGACY_KEY_FILE)); err != nil &&
		!os.IsNotExist(err) {
		return err
	}

	// A no-op means the records already matched what is committed, which is
	// nothing to record rather than something to report.
	err := GitCommit(repoPath, "Remove numeric task keys")

	var commandErr CommandError
	if err != nil && (!errors.As(err, &commandErr) || commandErr.Code != EXIT_NO_OP) {
		return err
	}

	return nil
}

// removeKeyFromRecord deletes the key line and leaves every other byte as it
// was. Re-serialising the whole record would also restyle notes written by
// other tools, turning a one-line change into a rewrite. A key in some other
// form is left to the serialiser.
func removeKeyFromRecord(path string, task Task) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	stripped := legacyKeyLine.ReplaceAll(data, nil)

	var remaining Task
	if err := yaml.Unmarshal(stripped, &remaining); err != nil || remaining.LegacyKey != 0 {
		task.LegacyKey = 0

		return writeRecord(path, task)
	}

	return os.WriteFile(path, stripped, 0o600)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}
