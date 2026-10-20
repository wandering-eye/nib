package nib

import (
	"fmt"
	"os"
)

func Help(cmd string) {
	var helpStr string

	showKey := false

	switch cmd {
	case CMD_NEXT:
		helpStr = `Usage: nib next [filter] [--]
Usage: nib [filter] [--]
Example: nib +work +bug --

Display list of non-resolved tasks in the current context, most recent last,
optional filter. It is the default command, so "next" is unnecessary.

Bypass the current context with --.

Colour key:
`
		showKey = true
	case CMD_ADD:
		helpStr = `Usage: nib add [template:<id>] [task summary] [--]
Example: nib add Fix main web page 500 error +bug P1 project:website

Add a task, printing the ID used later to reference the task, as in
"Added 1a2b3c4d: Fix main web page 500 error".

Tags, project and priority can be added anywhere within the task summary.

Add -- to ignore the current context. / can be used when adding tasks to note
any words after.

A copy of an existing task can be made by including "template:<id>". See
"nib help template" for more information on templates.

`
	case CMD_TEMPLATE:
		helpStr = `Usage nib template <id> [task summary] [--]
Example: nib template Fix main web page 500 error +bug P1 project:website
Example: nib template 1a2b3c4d project:

If valid task ID is supplied, a copy of the task is created as a template. If
no ID is given, a new task template is created.

Tags, project and priority can be added anywhere within the task summary.

Add -- to ignore the current context. / can be used when adding tasks to note
any words after

Template tasks are not displayed with "show-open" or "show-next" commands.
Their intent is to act as a readily available task template for commonly used
or repeated tasks.

To create a new task from a template use the command:
"nib add template:<id> [task summary] [--]"
The template task <id> remains unchanged, but a new task is created as a copy
with any modifications made in the task summary.

Github-style task lists (checklists) are recommended for templates, useful for
performing procedures. Example:

- [ ] buy bananas
- [ ] eat bananas
- [ ] make coffee

`
	case CMD_RM, CMD_REMOVE:
		helpStr = `Usage: nib remove <id...>
Example: nib 1a2b3c4d remove

Remove a task.

The task is deleted from the filesystem, and the change is committed.

`

	case CMD_LOG:
		helpStr = `Usage: nib log [task summary] [--]
Example: nib log Fix main web page 500 error +bug P1 project:website

Add an immediately resolved task. Syntax identical to add command.

Tags, project and priority can be added anywhere within the task summary.

Add -- to ignore the current context.

`
	case CMD_START:
		helpStr = `Usage: nib <id...> start
Usage: nib start [task summary] [--]
Example: nib 1a2b3c4d start
Example: nib start Fix main web page 500 error +bug P1 project:website

Mark a task as active, meaning you're currently at work on the task.

Alternatively, "start" can add a task and start it immediately with the same
syntax is the "add" command.  Tags, project and priority can be added anywhere
within the task summary.

Add -- to ignore the current context.
`
	case CMD_NOTE:
		fallthrough
	case CMD_NOTES:
		helpStr = `Usage: nib note <id>
Usage: nib note <id> <text>
Example: nib note 1a2b3c4d problem is faulty hardware

Edit or append text to the markdown notes attached to a particular task.
Resolved tasks can be noted too; their status is left as it is.
`
	case CMD_STOP:
		helpStr = `Usage: nib <id...> stop [text]
Example: nib 1a2b3c4d stop
Example: nib 1a2b3c4d stop replaced some hardware

Set a task as inactive, meaning you've stopped work on the task. Optional text
may be added, which will be appended to the note.
`
	case CMD_RESOLVE:
		fallthrough
	case CMD_DONE:
		helpStr = `Usage: nib <id...> done [closing note]
Example: nib 1a2b3c4d done
Example: nib 1a2b done replaced some hardware

Resolve a task. Optional text may be added, which will be appended to the note.
`
	case CMD_CONTEXT:
		helpStr = `Usage: nib context <filter>
Example: nib context +work -bug
Example: nib context none

Set a global filter consisting of a project, tags or antitags. Subsequent new
tasks and most commands will then have this filter applied automatically.

For example, if you were to run "task add fix the webserver," the given task
would then have the tag "work" applied automatically.

To reset to no context, run: nib context none

Context can also be set with the environment variable NIB_CONTEXT. If set, 
this context string will override the context stored on disk.
`
	case CMD_MODIFY:
		helpStr = `Usage: nib <id...> modify <filter>
Usage: nib modify <filter>
Example: nib 1a2b3c4d modify -work +home project:workbench -project:website

Modify the attributes of the given tasks, specified by ID. If no ID is given,
the operation will be performed to all tasks in the current context subject to
confirmation.

Modifiable attributes: tags, project and priority.
`
	case CMD_EDIT:
		helpStr = `Usage: nib <id...> edit

Edit a task in your text editor.
`
	case CMD_UNDO:
		helpStr = `Usage: nib undo
Usage: nib undo <n>

Undo the last <n> commits on the repository. Default is 1. Use

	nib git log

To see commit history. For more complicated history manipulation it may be best
to revert/rebase/merge on the nib repository itself. The nib repository
is at ~/.nib by default.
`
	case CMD_SYNC:
		helpStr = `Usage: nib sync

Synchronise with the remote git server. Runs git pull then git push. If there
are conflicts that cannot be automatically resolved, it is necessary to
manually resolve them in  ~/.nib or with the "task git" command.
`
	case CMD_GIT:
		helpStr = `Usage: nib git <args...>
Example: nib git status

Run the given git command inside ~/.nib
`
	case CMD_SHOW_RESOLVED:
		helpStr = `Usage: nib resolved

Show a report of last 1000 resolved tasks.
`
	case CMD_SHOW_TEMPLATES:
		helpStr = `Usage: dtask show-templates [filter] [--]

Show a report of stored template tasks with an optional filter.

Bypass the current context with --`
	case CMD_OPEN:
		helpStr = `Usage: nib <id...> open

Open all URLs found within the task summary and notes. If you commonly have
dozens of tabs open to later action, convert them into tasks to open later with
this command.
`
	case CMD_SHOW_PROJECTS:
		helpStr = `Usage: nib show-projects

Show a breakdown of projects with progress information
`

	case CMD_PRINT_BASH_COMPLETION,
		CMD_PRINT_ZSH_COMPLETION,
		CMD_PRINT_FISH_COMPLETION,
		CMD_PRINT_NU_COMPLETION:
		helpStr = `Usage: nib [bash|zsh|fish|nushell]-completion

Print a shell completion script to stdout. The script can be sourced from
a user's .bashrc, .zshrc, fish.config, or nushell config.nu like so:

    source <(nib bash-completion)
    source <(nib zsh-completion)
    nib fish-completion | source
    nib nushell-completion | save -f ~/.config/nushell/completions/nib.nu
    # then add ` + "`source ~/.config/nushell/completions/nib.nu`" + ` to your config.nu
`
	default:
		helpStr = `Usage: nib [id...] <cmd> [task summary/filter]

Where [task summary] is text with tags/project/priority specified. Tags are
specified with + (or - for filtering) eg: +work. The project is specified with
a project:g prefix eg: project:nib -- no quotes. Priorities run from P3
(low), P2 (default) to P1 (high) and P0 (critical). Text can also be specified
for a substring search of description and notes.

Cmd and IDs can be swapped, multiple IDs can be specified for batch
operations.

A task's ID is a leading part of its UUID. The first 8 characters are shown
wherever tasks are listed; any prefix of at least 4 characters works, as does
the full UUID, with or without hyphens. A prefix shared by more than one task
is refused with the candidates listed rather than guessed.

run "nib help <cmd>" for command specific help.

Add -- to ignore the current context. / can be used when adding tasks to note
any words after.

Available commands:

next                : Show most important tasks (priority, creation date -- truncated and default)
add                 : Add a task
template            : Add a task template
log                 : Log a task (already resolved)
start               : Change task status to active
note                : Append to or edit note for a task
stop                : Change task status to pending
done                : Resolve a task
context             : Set global context for task list and new tasks (use "none" to set no context)
modify              : Change task attributes specified on command line
edit                : Edit task with text editor
undo                : Undo last n commits
sync                : Pull then push to git repository, automatic merge commit.
open                : Open all URLs found in summary/annotations
git                 : Pass a command to git in the repository. Used for push/pull.
remove              : Remove a task (use to remove tasks added by mistake)
show-projects       : List projects with completion status
show-tags           : List tags in use
show-active         : Show tasks that have been started
show-paused         : Show tasks that have been started then stopped
show-open           : Show all non-resolved tasks (without truncation)
show-resolved       : Show resolved tasks
show-templates      : Show task templates
show-unorganised    : Show untagged tasks with no projects (global context)
bash-completion     : Print bash completion script to stdout
fish-completion     : Print fish completion script to stdout
zsh-completion      : Print zsh completion script to stdout
nushell-completion  : Print nushell completion script to stdout
help                : Get help on any command or show this message
version             : Show nib version information

Exit codes:

0  success, including a filter that matched nothing
1  failure
2  usage error, including an ID shorter than 4 characters
3  an ID matched no task, or only resolved ones where an open task is needed
4  nothing changed, or a confirmation was declined
5  an ID matched more than one task

Colour Key:
`

		showKey = true
	}

	fmt.Fprint(os.Stderr, helpStr)

	if showKey {
		colourPrintln(0, FG_PRIORITY_CRITICAL, BG_DEFAULT_2, "Critical priority")
		colourPrintln(0, FG_PRIORITY_HIGH, BG_DEFAULT_2, "High priority")
		colourPrintln(0, FG_DEFAULT, BG_DEFAULT_1, "Normal priority")
		colourPrintln(0, FG_PRIORITY_LOW, BG_DEFAULT_2, "Low priority")
		colourPrintln(0, FG_ACTIVE, BG_ACTIVE, "Active")
		colourPrintln(0, FG_DEFAULT, BG_PAUSED, "Paused")
	}

	os.Exit(0)
}

func colourPrintln(mode, fg, bg int, line string) {
	line = FixStr(line, 25)
	fmt.Fprintf(os.Stderr, "\033[%d;38;5;%d;48;5;%dm%s\033[0m\n", mode, fg, bg, line)
}
