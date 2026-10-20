# nib database format

The default database location is `~/.nib/`, but can be configured by the
environment variable `NIB_GIT_REPO`.

The format on disk stores the tasks in a directory according to the task
status, with each task stored under a yaml file with a UUID4 as the filename.
UUIDs are used to avoid conflicts when synchronising. The yaml schema is
defined by the `Task` struct in [task.go](../task.go).

This way only non-resolved tasks are actually loaded for most commands, so
performance is stable even with a large task history.

A task's identity is its UUID; nothing else is allocated, so two clones that
add tasks independently never collide. Commands accept any leading part of the
UUID of at least 4 characters.

Earlier versions also wrote a numeric `key` into each record and kept a
`next-key` counter at the top of the store. The first load by a later version
removes both and commits that as "Remove numeric task keys".

Machine-local state (the current context) lives in `.git/nib/state.bin` and is
not synchronised.

TODO elaborate with examples.
