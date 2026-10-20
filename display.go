package nib

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// sgr returns an ANSI select-graphic-rendition escape, or nothing when colour
// is disabled. Every escape this program emits is built here, so NO_COLOR and
// --color have a single place to take effect.
func sgr(params string) string {
	if !COLOUR_ENABLED {
		return ""
	}

	return "\033[" + params + "m"
}

// colour wraps text in an SGR escape, resetting afterwards.
func colour(params, text string) string {
	return sgr(params) + text + sgr("0")
}

// DisplayByNext renders the TaskSet's array of tasks. The caller asks for the
// rendering it wants; what stdout is attached to is never consulted.
func (ts *TaskSet) DisplayByNext(ctx Query, truncate, asJSON bool) error {
	ts.SortByCreated(Ascending)  // older tasks first (from top) like a FIFO queue
	ts.SortByPriority(Ascending) // high priority tasks first, of course

	if asJSON {
		return renderJSON(ts.Tasks())
	}

	ctx.PrintContextDescription()

	err := ts.renderTable(truncate)
	if err != nil {
		return err
	}

	var critical int

	var totalCritical int

	for _, t := range ts.Tasks() {
		if t.Priority == PRIORITY_CRITICAL {
			critical++
		}
	}

	// search outside current filter in taskset
	for _, t := range ts.AllTasks() {
		if t.Priority == PRIORITY_CRITICAL && !StrSliceContains(HIDDEN_STATUSES, t.Status) {
			totalCritical++
		}
	}

	if critical < totalCritical {
		fmt.Println(colour(
			fmt.Sprintf("38;5;%d", FG_PRIORITY_CRITICAL),
			fmt.Sprintf(
				"%v critical task(s) outside this context! Use `nib -- P0` to see them.",
				totalCritical-critical,
			),
		))
	}

	return nil
}

// renderJSON writes any of the structures this program reports as indented
// JSON. It is reached only when the caller asked for JSON with --json.
func renderJSON(value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}

	return WriteStdout(data)
}

func (ts *TaskSet) renderTable(truncate bool) error {
	tasks := ts.Tasks()
	total := len(tasks)

	if ts.NumTotal() == 0 {
		fmt.Println("No tasks found. Run `nib help` for instructions.")
	} else if len(tasks) == 0 {
		// an empty result is not a failure -- the same query renders [] with
		// --json, and the exit code must not depend on the rendering
		fmt.Println("No matching tasks in given context or filter.")
	} else if len(tasks) == 1 {
		task := tasks[0]
		task.Display()

		if task.Notes != "" {
			fmt.Printf("\nNotes on task %s:\n%s\n\n", task.ShortUUID(), colour("38;5;245", task.Notes))
		}

		return nil
	} else {
		maxTasks := MAX_TASKS_SHOWN

		if truncate && maxTasks < len(tasks) {
			tasks = tasks[:maxTasks]
		}

		table := NewTable(
			TerminalWidth(),
			"ID",
			"Priority",
			"Tags",
			"Due",
			"Project",
			"Summary",
		)

		for _, t := range tasks {
			style := t.Style()

			table.AddRow(
				[]string{
					t.ShortUUID(),
					t.Priority,
					strings.Join(t.Tags, " "),
					t.ParseDueDateToStr(),
					t.Project,
					t.LongSummary(),
				},
				style,
			)
		}

		table.Render()

		if truncate && maxTasks < total {
			fmt.Printf("\n%v/%v tasks shown.\n", maxTasks, total)
		} else {
			fmt.Printf("\n%v tasks.\n", total)
		}
	}

	return nil
}

func (task *Task) Display() {
	table := NewTable(
		TerminalWidth(),
		"Name",
		"Value",
	)

	table.AddRow([]string{"ID", task.ShortUUID()}, RowStyle{})
	table.AddRow([]string{"Priority", task.Priority}, RowStyle{})
	table.AddRow([]string{"Summary", task.Summary}, RowStyle{})
	table.AddRow([]string{"Status", task.Status}, RowStyle{})
	table.AddRow([]string{"Project", task.Project}, RowStyle{})
	table.AddRow([]string{"Tags", strings.Join(task.Tags, ", ")}, RowStyle{})
	table.AddRow([]string{"UUID", task.UUID}, RowStyle{})
	table.AddRow([]string{"Created", task.Created.String()}, RowStyle{})

	if !task.Resolved.IsZero() {
		table.AddRow([]string{"Resolved", task.Resolved.String()}, RowStyle{})
	}

	if !task.Due.IsZero() {
		table.AddRow([]string{"Due", task.Due.String()}, RowStyle{})
	}

	table.Render()
}

func (t *Task) Style() RowStyle {
	now := time.Now()
	style := RowStyle{}
	active := t.Status == STATUS_ACTIVE
	paused := t.Status == STATUS_PAUSED
	resolved := t.Status == STATUS_RESOLVED

	getFg := func(normalColor, activeColor int) int {
		if active {
			return activeColor
		}
		return normalColor
	}

	if t.Priority == PRIORITY_CRITICAL {
		style.Fg = getFg(FG_PRIORITY_CRITICAL, FG_ACTIVE_PRIORITY_CRITICAL)
	} else if !t.Due.IsZero() && t.Due.Before(now) && !resolved {
		style.Fg = getFg(FG_PRIORITY_HIGH, FG_ACTIVE_PRIORITY_HIGH)
	} else if t.Priority == PRIORITY_HIGH {
		style.Fg = getFg(FG_PRIORITY_HIGH, FG_ACTIVE_PRIORITY_HIGH)
	} else if t.Priority == PRIORITY_LOW {
		style.Fg = getFg(FG_PRIORITY_LOW, FG_ACTIVE_PRIORITY_LOW)
	} else {
		style.Fg = getFg(FG_DEFAULT, FG_ACTIVE)
	}

	if active {
		style.Bg = BG_ACTIVE
	} else if paused {
		style.Bg = BG_PAUSED
	}

	return style
}

// TODO combine with previous with interface, plus computed Project status.
func (p *Project) Style() RowStyle {
	style := RowStyle{}

	if p.Active {
		style.Fg = FG_ACTIVE
		style.Bg = BG_ACTIVE
	} else if p.Priority == PRIORITY_CRITICAL {
		style.Fg = FG_PRIORITY_CRITICAL
	} else if p.Priority == PRIORITY_HIGH {
		style.Fg = FG_PRIORITY_HIGH
	} else if p.Priority == PRIORITY_LOW {
		style.Fg = FG_PRIORITY_LOW
	}

	return style
}

func (ts TaskSet) DisplayByWeek(asJSON bool) error {
	ts.SortByResolved(Ascending)

	if asJSON {
		return renderJSON(ts.Tasks())
	}

	var table *Table

	var lastWeek int

	tasks := ts.Tasks()

	for _, t := range tasks {
		_, week := t.Resolved.ISOWeek()

		// guaranteed true for first iteration, ISOweek starts with 1.
		if week != lastWeek {
			if table != nil && len(table.Rows) > 0 {
				table.Render()
			}
			// insert gap
			fmt.Printf(
				"\n\n> Week %d, starting %s\n\n",
				week,
				t.Resolved.Format("Mon 2 Jan 2006"),
			)

			table = NewTable(
				TerminalWidth(),
				"Resolved",
				"Priority",
				"Tags",
				"Due",
				"Project",
				"Summary",
			)
		}

		table.AddRow(
			[]string{
				t.Resolved.Format("Mon 2"),
				t.Priority,
				strings.Join(t.Tags, " "),
				t.ParseDueDateToStr(),
				t.Project,
				t.LongSummary(),
			},
			t.Style(),
		)

		_, lastWeek = t.Resolved.ISOWeek()
	}

	if table != nil {
		table.Render()
	}

	fmt.Printf("%v tasks.\n", len(tasks))

	return nil
}

func (ts TaskSet) DisplayProjects(asJSON bool) error {
	if asJSON {
		return renderJSON(ts.GetProjects())
	}

	ts.renderProjectsTable()

	return nil
}

func (ts TaskSet) renderProjectsTable() {
	projects := ts.GetProjects()
	table := NewTable(
		TerminalWidth(),
		"Name",
		"Progress",
		"Created",
	)

	for _, project := range projects {
		if project.TasksResolved < project.Tasks {
			table.AddRow(
				[]string{
					project.Name,
					fmt.Sprintf("%d/%d", project.TasksResolved, project.Tasks),
					project.Created.Format("Mon 2 Jan 2006"),
				},
				project.Style(),
			)
		}
	}

	table.Render()
}
