# nib database format

The default database location is `~/.nib/`, but can be configured by the
environment variable `NIB_GIT_REPO`.

The format on disk stores the tasks in a directory according to the task
status, with each task stored under a yaml file with a UUID4 as the filename.
UUIDs are used to avoid conflicts when synchronising. The yaml schema is
defined by the `Task` struct in [task.go](../task.go).

This way only non-resolved tasks are actually loaded for most commands, so
performance is stable even with a large task history.

Each task records its ID in the yaml as `key`. Keys are allocated at creation
from the committed `next-key` high water mark and are never reissued, not after
a task is resolved and not in a fresh clone.

Machine-local state (the current context) lives in `.git/nib/state.bin` and is
not synchronised.

TODO elaborate with examples.
