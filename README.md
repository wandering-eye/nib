<p align="center">
<img align="center" src="etc/icon.png" alt="icon" height="64" />
</p>

<h1 align="center">nib</h1>

<p align="center">
<i>Single binary terminal-based TODO manager with git-based sync + markdown notes per task</i>
</p>

nib is a personal task tracker designed to help you focus. It is similar to
[Taskwarrior](https://taskwarrior.org/) but uses git to synchronise instead of
a special protocol.

If you're at home in the terminal, and with solving the occasional merge
conflict, then nib is for you!

Features:

- Powerful context system (automatically applies filter/tags to queries and new tasks)
- **Git powered sync**/undo/resolve ([passwordstore.org](https://www.passwordstore.org/) style) which means no need to set up a sync server, and syncing between devices is easy!
- Task listing won't break with long task text
- `note` command -- edit a **full markdown note** for each task. **Checklists are useful here.**
- `open` command -- **open URLs found in specified task** (including notes) in the browser
- zsh/bash completion (including tags and projects in current context) for speed; PowerShell completion on Windows
- A single statically-linked binary

Non-features:

- Collaboration. This is a personal task tracker. Use another system for
  projects that involve multiple people. Note that it can still be beneficial
  to use nib to track what you are working on in the context of a
  multi-person project tracked elsewhere.

Requirements:

- Git
- A 256-color capable terminal

# Installation

On macOS, install with Homebrew:

```sh
brew install wandering-eye/tap/nib
```

Otherwise:

1. Copy the executable (from the [releases page][releases]) to somewhere in your PATH.
   - Linux/macOS: name it `nib` and mark it executable (e.g. `/usr/local/bin/`).
   - Windows: use `nib.exe` and place it in a directory on `PATH` (e.g. `%USERPROFILE%\bin`).
1. Enable shell completions:
   - Bash: add `source <(nib bash-completion)` to your `.bashrc`.
   - Zsh: add `nib zsh-completion > /usr/local/share/zsh/site-functions/_nib` or source `completions/zsh.sh` in your `~/.zshrc`.
   - Nushell: run `nib nushell-completion | save -f ~/.config/nushell/completions/nib.nu` then add `source ~/.config/nushell/completions/nib.nu` to your `config.nu`.
   - PowerShell (Windows): see section "PowerShell completion" below.
1. Set up an alias in your `.bashrc`: `alias task=nib` or `alias t=nib` to make task management slightly faster.
1. Create or clone a `~/.nib` git repository for the data, if you haven't already:
   - Linux/macOS: `mkdir ~/.nib && git -C ~/.nib init`
   - Windows: `mkdir %USERPROFILE%\.nib` then `git -C %USERPROFILE%\.nib init`

## Windows notes

- Default data location: `%USERPROFILE%\.nib` (can be overridden via `NIB_GIT_REPO`).
- On first run, nib may prompt to create the repository if it doesn't exist; answer `y`/`yes` to proceed.
- Terminal: use Windows Terminal or PowerShell with a 256-color capable profile for best results.
- Shell completions: PowerShell completion is supported; see next section.

### PowerShell completion

To enable PowerShell tab completion for `nib` on Windows, use the embedded script in `completions/powershell.ps1`.

One-off for the current session:

```powershell
. ./completions/powershell.ps1
```

Enable permanently by dot-sourcing from your PowerShell profile:

```powershell
# Create a profile if it doesn't exist
if (!(Test-Path -LiteralPath $PROFILE)) { New-Item -ItemType File -Path $PROFILE -Force | Out-Null }

# Open your profile for editing
notepad $PROFILE

# Add the following line (adjust path if needed):
. "$PSScriptRoot/completions/powershell.ps1"
```

Alternatively, hardcode the absolute path to the repository if `PSScriptRoot` is not suitable for your setup, e.g.:

```powershell
. "C:\\Users\\<you>\\source\\repos\\nib\\completions\\powershell.ps1"
```

# Moving from Taskwarrior

We have a [migration guide](doc/taskwarrior-migration.md) to make the transition from taskwarrior to nib a simple process.

# Usage

```
Usage: nib [id...] <cmd> [task summary/filter]

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
```

# Syntax

## Priority

| Symbol | Name     | Note                                                                |
| ------ | -------- | ------------------------------------------------------------------- |
| `P0`   | Critical | Must be resolved immediately. May appear in all contexts in future. |
| `P1`   | High     | Need to be resolved in near future, display in highlighted          |
| `P2`   | Normal   | Default priority                                                    |
| `P3`   | Low      | Shown at bottom and faded.                                          |

## Operators

| Symbol          | Syntax                   | Description                                                      | Example                                       |
| --------------- | ------------------------ | ---------------------------------------------------------------- | --------------------------------------------- |
| `+`             | `+<tag>`                 | Include tag. Filter/context, or when adding task.                | `nib add fix server +work`                 |
| `-`             | `-<tag>`                 | Exclude tag. Filter/context only.                                | `nib next -feature`                        |
| `--`            | `--`                     | Ignore context. When listing or adding tasks.                    | `nib --`, `task add -- +home do guttering` |
| `/`             | `/`                      | When adding a task, everything after will be a note.             | `nib add check out ipfs / https://ipfs.io` |
| `project:`      | `project:<project>`      | Set project. Filter/context, or when adding task.                | `nib context project:nib`               |
| `-project:`     | `-project:<project>`     | Exclude project, filter/context only.                            | `nib next -project:nib -work`           |
| `template:`     | `template:<id>`          | Base new task on a template.                                     | `nib add template:1a2b3c4d`                |
| `due:`          | `due:<date>`             | Set or filter by due date.                                       | `nib add task due:friday`                  |
| `due.[filter]`  | `due.[filter]:<date>`    | Filter tasks based on due date filter (before, after, in/on).    | `nib next due.before:tomorrow`             |


## State

| State    | Description                                   |
| -------- | --------------------------------------------- |
| Pending  | Tasks that have never been started            |
| Active   | Tasks that have been started                  |
| Paused   | Tasks that have been started but then stopped |
| Resolved | Tasks that have been done/close/completed     |

## Due Dates

Due dates can be specified when adding tasks or filtering existing tasks.

### Date Formats

| Format          | Description                                           | Examples                                                    |
| --------------- | ----------------------------------------------------- | ----------------------------------------------------------- |
| Relative dates  | `yesterday`, `today`, `tomorrow`                      | `due:tomorrow`                                              |
| Weekdays        | Full names or 3-letter abbreviations                  | `due:monday`, `due:wed`, `due:this-fri`, `due:next-tuesday` |
| Full dates      | YYYY-MM-DD format                                     | `due:2024-12-25`                                            |
| Month-day       | MM-DD format (sets current year automatically)        | `due:12-25`                                                 |
| Day of month    | DD format (sets current month and year automatically) | `due:15`                                                    |

### Adding Tasks with Due Dates

```bash
nib add task with due date due:next-monday
nib add finish report due:friday
nib add pay bills due:15  # 15th of current month
nib add halloween party due:2025-10-31
```

### Filtering Tasks by Due Date

```bash
nib next due.before:friday        # Tasks due before Friday
nib next due.after:tomorrow       # Tasks due after tomorrow  
nib next due.on:tue               # Tasks due on Tuesday
nib next due:overdue              # Overdue tasks
nib next due.before:2025-12-31    # Tasks due before end of year
```

# Contexts

When nib runs, a context can be set to filter the task output. Run `nib help context`
for more examples. There are two ways to set a context.

1. The `context` command, which sets a global context on disk.
1. The `NIB_CONTEXT` environment variable. Contexts set by this environment
   variable override the global context on disk.

Use the `context` to set a context that will apply by default, no matter what
terminal window you're using.

Use the `NIB_CONTEXT` environment variable to override context in specific
uses. For instance, a [direnv](https://direnv.net/) config can set a context for
particular directories.

Context is not synchronised between machines.

# Dealing with merge conflicts

nib is written in such a way that merge conflicts should not happen, unless
a task is edited independently on 2 or more machines without synchronising. In
practice this happens rarely; however when it does happen nib will fail to
commit and warn you. You'll then need to go to the underlying `~/.nib` git
repository and resolve manually before committing and running `nib sync`.
Two clones that each add a task while out of sync is not a conflict: tasks
are identified by UUID, so both simply appear after the next sync.

# Exit codes

| Code | Meaning                                                                   |
| ---- | ------------------------------------------------------------------------- |
| 0    | Success, including a filter that matched nothing                          |
| 1    | Failure                                                                   |
| 2    | Usage error, including an ID shorter than 4 characters                    |
| 3    | An ID matched no task, or only resolved ones where an open task is needed |
| 4    | Nothing changed, or a confirmation was declined                           |
| 5    | An ID matched more than one task; candidates go to stderr                 |

# JSON output

`--json` renders tasks as an array of objects. A task's identity is its
`uuid` field, the full UUID; any prefix of it of at least 4 characters is a
valid ID on the command line. Earlier versions also emitted a numeric
`id` field, which no longer exists.

# Performance

See [etc/PERFORMANCE.md](etc/PERFORMANCE.md)

# General tips

- Overwhelmed by tasks? Try focussing by prioritising (set priorities) or narrowing the context. The `show-tags` and `show-projects` commands are useful for creating a context.
- Use nib to track things you might forget, rather than everything. SNR is important. Don't track tasks for the sake of it, and don't track _ideas_. Track ideas separately.
- Spend regular time reviewing tasks. You'll probably find some you've already resolved, and many you've forgotten. The `show-unorganised` command is good for this.
- Try to work through tasks from the top of the list. nib sorts by priority then creation date -- the most important tasks are at the top.
- Use `start`/`stop` to mark what you're genuinely working on right now; it makes resuming work faster. Paused tasks will be slightly highlighted, so you won't lose track of them. `show-paused` helps if they start to pile up.
- Keep a [github-style check list](https://help.github.com/en/articles/about-task-lists) in the markdown note of complex or procedural tasks
- Failing to get started working? Start with the smallest task
- Record only required tasks. Track ideas separately, else your task list will grow unboundedly! I keep an `ideas.md` for various projects for this reason.
- set `NIB_CONTEXT` in a `.envrc` per-project repository. With direnv, this allows you to automatically switch context

# Database

See [etc/DATABASE_FORMAT.md](etc/DATABASE_FORMAT.md)

The default database location is:

- Linux/macOS: `~/.nib/`
- Windows: `%USERPROFILE%\\.nib`

It can be configured by the environment variable `NIB_GIT_REPO`.

# Integration of fuzzy finders

Instead of passing task IDs to commands, a fuzzy finder like [*fzf*](https://github.com/junegunn/fzf) can be used to
search tasks by their summary instead. This is not part of nib itself, but can be achieved by defining a shell
wrapper like this in your `.bashrc` or `.zshrc`:

```bash
function nib () {
    local has_double_dash subcommand

    if (( $# == 1 || $# == 2 )); then
        has_double_dash="$( [[ "$1" == "--" || "$2" == "--" ]] && echo "1" || echo "0" )"
        subcommand="$( [[ "$1" == "--" ]] && echo "$2" || echo "$1" )"
        if (( $# == 1 || has_double_dash )) && \
            [[ "${subcommand}" =~ ^(done|edit|note|open|remove|start|stop)$ ]]
        then
            command nib show-open --json $( (( has_double_dash )) && echo "--" ) \
                | jq -r '.[] | [.uuid, .summary] | @tsv' \
                | fzf --delimiter='\t' --nth=2 --accept-nth=1 --preview='nib {1} --' \
                | xargs -n1 nib "${subcommand}"
            return
        fi
    fi

    command nib "$@"
}
```

The wrapper function checks if nib was called with a subcommand missing the task ID, and if so uses the `show-open`
command to get a list of tasks to choose from. `--` can be used to ignore the current context for the fuzzy search. All
other commands are passed directly to nib.

A more advanced version, which adds additional support for the nib `modify` command by inserting the selected task ID
directly into the shell prompt, can be found in [etc/fzf_integration.zsh](etc/fzf_integration.zsh) (zsh only).

# Alternatives

Alternatives listed must be capable of running in the terminal.

- [TaskLite](https://github.com/ad-si/TaskLite) -- The CLI task manager for power users, written in Haskell
- [Taskwarrior](https://taskwarrior.org/) -- the closest analogue
- [Taskbook](https://github.com/klaussinani/taskbook) -- board metaphor, note support
- [todo.txt-cli](https://github.com/todotxt/todo.txt-cli)
- [etm](https://github.com/dagraham/etm-dgraham) -- event and task manager
- [t](https://github.com/sjl/t) -- a minimal todo list with an amusing philosophy

# FAQ

> Does nib encrypt tasks?

Encryption is not a design goal of nib. If you want to have your remote
repository encrypted, you may consider
[git-remote-gcrypt](https://spwhitton.name/tech/code/git-remote-gcrypt/) or
[git-crypt](https://github.com/AGWA/git-crypt). Note that nib has not been
tested with these tools, nor can any claims be made about the security of the
tools themselves.

> Is it possible to modify more than one task at once with a filter?

Yes.

1. Set a context:
2. Run a modify command without an ID
3. Hit y to confirm to modify all tasks in context

This means it's natural to review the tasks that would be modified before
modifying by listing all tasks in the current context first, instead of
potentially operating blindly by matching tags.

You can also specify multiple task IDs at one time, as with any other command.

[releases]: https://github.com/wandering-eye/nib/releases
