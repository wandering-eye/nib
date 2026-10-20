package nib

// main task data structures

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type TaskSet struct {
	tasks []*Task

	// indices
	tasksByID   map[int]*Task
	tasksByUUID map[string]*Task

	// program metadata
	// nextKey is the committed high water mark: keys are handed out from here
	// and never reissued.
	nextKey  int
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
	ts.tasksByID = make(map[int]*Task)

	ts.repoPath = repoPath

	// Construct our options struct by calling our passed-in TaskSetOpt functions.
	ts.nextKey = loadNextKey(repoPath)

	// Keys live in the records now, not in an untracked cache.

	var statuses []string

	if includeResolved {
		// expensive to load -- resolved tasks are unbounded
		statuses = ALL_STATUSES
	} else {
		// non-resolved tasks are bounded, so it's OK to load them even if
		// some are redundant due to query. It's also important to load all
		// non-resolved tasks at once for consistent IDs in case
		// SavePendingChanges is not called...!
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

			if _, err := ts.LoadTask(t); err != nil {
				return nil, fmt.Errorf("%s: %w", finfo.Name(), err)
			}
		}
	}

	// Records written by earlier versions carry no key. Assign them here, after every
	// record is loaded, so the keys already on disk are known and cannot be
	// handed out twice.
	assigned, err := ts.assignMissingKeys()
	if err != nil {
		return nil, err
	}

	// Write them immediately. Holding them in memory only meant the next
	// command recomputed them over whatever it happened to load, so a key
	// could name a different record each run -- the very thing a durable key
	// is for. This happens once, on the first load of such a store.
	if assigned {
		ts.SavePendingChanges()

		// A no-op here means the keys written match what is already committed,
		// which is nothing to record rather than something to report.
		err := GitCommit(repoPath, "Assign durable keys to existing records")
		var commandErr CommandError
		if err != nil && (!errors.As(err, &commandErr) || commandErr.Code != EXIT_NO_OP) {
			return nil, err
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

func (ts *TaskSet) SortByCreated(dir SortByDirection) {
	switch dir {
	case Ascending:
		// Oldest first
		sort.SliceStable(
			ts.tasks,
			func(i, j int) bool { return ts.tasks[i].Created.Before(ts.tasks[j].Created) },
		)
	case Descending:
		// Newest first
		sort.SliceStable(
			ts.tasks,
			func(i, j int) bool { return ts.tasks[i].Created.After(ts.tasks[j].Created) },
		)
	}
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
	switch dir {
	case Ascending:
		// Oldest resolved first
		sort.SliceStable(
			ts.tasks,
			func(i, j int) bool { return ts.tasks[i].Resolved.Before(ts.tasks[j].Resolved) },
		)
	case Descending:
		// Newest resolved first
		sort.SliceStable(
			ts.tasks,
			func(i, j int) bool { return ts.tasks[i].Resolved.After(ts.tasks[j].Resolved) },
		)
	}
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
	// A task arriving without a UUID is being created rather than read off
	// disk, which is the moment its key is allocated.
	isNew := task.UUID == ""

	task.Normalise()

	if task.UUID == "" {
		task.UUID = MustGetUUID4String()
	}

	if err := task.Validate(); err != nil {
		return Task{}, err
	}

	if isNew && task.ID == 0 {
		task.ID = ts.nextKey
		ts.nextKey++
		task.WritePending = true
	}

	if ts.tasksByUUID[task.UUID] != nil {
		// load tasks, do not overwrite
		// TODO ??? (maybe return a nil pointer instead?)
		return Task{}, nil
	}

	// A key claimed twice means two records disagree about who owns it, which
	// no amount of guessing here can resolve. Say so rather than quietly
	// renumbering one of them, which is what recycling used to do.
	if task.ID > 0 && ts.tasksByID[task.ID] != nil {
		return Task{}, fmt.Errorf(
			"key %d is claimed by both %s and %s; keys must be unique",
			task.ID, ts.tasksByID[task.ID].UUID, task.UUID,
		)
	}

	if task.Created.IsZero() {
		task.Created = time.Now().UTC()
		task.WritePending = true
	}

	ts.tasks = append(ts.tasks, &task)
	ts.tasksByUUID[task.UUID] = &task
	ts.tasksByID[task.ID] = &task

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

func (ts *TaskSet) Filter(query Query) {
	for _, task := range ts.tasks {
		if !task.MatchesFilter(query) {
			task.filtered = true
		}
	}
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

func (ts *TaskSet) MustGetByID(id int) Task {
	task, err := ts.GetByID(id)
	if err != nil {
		ExitError(err)
	}

	return task
}

func (ts *TaskSet) GetByID(id int) (Task, error) {
	if ts.tasksByID[id] == nil {
		// A ref naming nothing is its own outcome, distinct from a filter that
		// matched nothing. Every command that resolves a ref reports it the same
		// way rather than flattening it to a generic failure.
		return Task{}, NoMatchError("no open task with id %v exists", id)
	}

	return *ts.tasksByID[id], nil
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

	// The high water mark is committed alongside the records, so a key that has
	// been issued is never issued again -- not after a resolve, not in a fresh
	// clone. Earlier versions kept ids in an untracked cache specifically to avoid merge
	// conflicts between clones. That trade is reversed here: a conflict on this
	// file is visible and fixable, whereas renumbering the whole store because
	// a cache file went missing was neither.
	ts.mustSaveNextKey()
}

// nextKeyPath is the committed high water mark.
func nextKeyPath(repoPath string) string {
	return filepath.Join(repoPath, "next-key")
}

// loadNextKey reads the high water mark. A store that has never had one starts
// at 1, which is also the case for a store written by an earlier version.
func loadNextKey(repoPath string) int {
	data, err := os.ReadFile(nextKeyPath(repoPath))
	if err != nil {
		return 1
	}

	key, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || key < 1 {
		return 1
	}

	return key
}

func (ts *TaskSet) mustSaveNextKey() {
	next := ts.nextKey
	for _, task := range ts.tasks {
		if task.ID >= next {
			next = task.ID + 1
		}
	}

	if err := os.WriteFile(nextKeyPath(ts.repoPath), []byte(strconv.Itoa(next)+"\n"), 0o600); err != nil {
		ExitFail("could not save the key high water mark: %v", err)
	}
}

type SortByDirection string

const (
	Ascending  SortByDirection = "ascending"
	Descending SortByDirection = "descending"
)

// assignMissingKeys gives a key to every record that has none. The order comes
// from the whole store rather than from whatever this command loaded, because a
// key that depends on which statuses were read is the defect this replaced: the
// same record would answer to a different number depending on the command.
func (ts *TaskSet) assignMissingKeys() (bool, error) {
	var keyless bool

	next := ts.nextKey

	for _, task := range ts.tasks {
		if task.ID > 0 {
			if task.ID >= next {
				next = task.ID + 1
			}

			continue
		}

		keyless = true
	}

	ts.nextKey = next

	if !keyless {
		return false, nil
	}

	order, highest, err := keylessOrderAcrossStore(ts.repoPath)
	if err != nil {
		return false, err
	}

	if highest >= ts.nextKey {
		ts.nextKey = highest + 1
	}

	for _, task := range ts.tasks {
		if task.ID > 0 {
			continue
		}

		position, known := order[task.UUID]
		if !known {
			return false, fmt.Errorf("%s has no key and was not found in the store", task.UUID)
		}

		task.ID = ts.nextKey + position
		task.WritePending = true
		ts.tasksByID[task.ID] = task
	}

	ts.nextKey += len(order)

	return true, nil
}

// keylessOrderAcrossStore walks every status directory and returns the position
// each keyless record takes in creation order, plus the highest key already in
// use. Reading the whole store is the price of a numbering that does not depend
// on the command; it only happens while records still lack keys.
func keylessOrderAcrossStore(repoPath string) (map[string]int, int, error) {
	type record struct {
		uuid    string
		created time.Time
	}

	var (
		keyless []record
		highest int
	)

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

			task, err := unmarshalTask(path, entry, status)
			if err != nil {
				return nil, 0, fmt.Errorf("%s: %w", entry.Name(), err)
			}

			if task.ID > 0 {
				if task.ID > highest {
					highest = task.ID
				}

				continue
			}

			keyless = append(keyless, record{uuid: task.UUID, created: task.Created.UTC()})
		}
	}

	sort.SliceStable(keyless, func(first, second int) bool {
		if !keyless[first].created.Equal(keyless[second].created) {
			return keyless[first].created.Before(keyless[second].created)
		}

		return keyless[first].uuid < keyless[second].uuid
	})

	order := make(map[string]int, len(keyless))
	for position, entry := range keyless {
		order[entry.uuid] = position
	}

	return order, highest, nil
}
