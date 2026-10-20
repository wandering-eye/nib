#!/bin/bash
# Just a quick smoke test to check nothing major is broken as a start of CI

# This script should only test commands work without crashing. Behavioural
# tests should use the go test system.

# exit on error and print commands
set -x
set -e -o pipefail

# isolated db locations (repo2 is used for a sync target). NIB_GIT_REPO is
# overridden so it cannot leak in from the caller's shell, and assigned apart
# from export so a failed mktemp stops the script instead of leaving it empty,
# which would fall back to the real store.
NIB_GIT_REPO=$(mktemp --directory)
UPSTREAM_BARE_REPO=$(mktemp --directory)
export NIB_GIT_REPO UPSTREAM_BARE_REPO

if [[ -e nib ]]; then
    rm nib
fi

cleanup() {
    set +x
    set +e
    rm -rf $NIB_GIT_REPO
    rm -rf $UPSTREAM_BARE_REPO
    rm -f nib
}

trap cleanup EXIT

go build -o nib ./cmd/nib

# run a creating command and print the short id it reports, e.g. "Added 1a2b3c4d: ..."
created() {
    local id
    id=$(./nib "$@" | sed -n 's/^[A-Za-z ]*\([0-9a-f]\{8\}\): .*/\1/p')
    test -n "$id"
    echo "$id"
}

# initialse git repo
git -C $NIB_GIT_REPO init -b master
git -C $NIB_GIT_REPO config user.email "you@example.com"
git -C $NIB_GIT_REPO config user.name "Test user"
git -C $UPSTREAM_BARE_REPO init --bare -b master

# general task state management and commands
./nib help
first=$(created add test task +foo project:bar)
./nib start "$first"
./nib stop "$first"
echo y | ./nib remove "$first"
readded=$(created add re-add add test task +foo project:bar)
another=$(created add another task)
./nib +foo
./nib -foo
./nib -project:foo
./nib "$another" done
./nib undo
./nib log something
./nib +foo
./nib note "$readded" this is a note
./nib context +foo
./nib next
./nib "$readded" done
./nib show-resolved

./nib show-projects

# -- to remove current context which is +foo
./nib add -- unorganised task
./nib show-unorganised

# we are in context project:bar, adding with another project should fail
./nib context project:bar
! ./nib add project:baz test
# ... however, bypassing the context with -- should work
./nib add project:cheese test --

# test context listing with a context and no context
./nib context
./nib context none
./nib context

# try to resolve a task with an incomplete tasklist: should fail
tasklist=$(created add a tasklist task -- "- [ ] incomplete task")
! ./nib "$tasklist" done

# test template functions
original=$(created add task to copy)
copied=$(created add template:"$original" +copiedTask)
./nib template "$copied"
# the copy should now be a template
./nib show-templates +copiedTask
# copy Template with some modifications
./nib add template:"$copied" -copiedTask +copiedTemplate
./nib show-open +copiedTemplate
# Create new template from CMD line.
./nib template give me some things to do P1 +uniqueTag
./nib show-templates +uniqueTag

# test git command pass through
./nib git status

# regression test: nil pointer dereference when showing by-week tables of zero length
./nib show-resolved project:doesnotexist

# set the bare repository as upstream origin to test sync against, and then push to it
git -C $NIB_GIT_REPO remote add origin $UPSTREAM_BARE_REPO
git -C $NIB_GIT_REPO push -u origin master:master
./nib sync

# cause independent changes (could be from separate downstream repositories,
# but its easier and equivalent to simulate this with a hard reset)
eggs=$(created add eggs test task +foo project:bar)
./nib sync
./nib git reset --hard HEAD~1
bacon=$(created add bacon test task +foo project:bar)
./nib sync

# both clones added a task before syncing (issue #1); after the merge every
# command still works and both tasks are there
./nib next
open_tasks=$(./nib show-open)
grep -q "eggs test task" <<<"$open_tasks"
grep -q "bacon test task" <<<"$open_tasks"
./nib "$eggs"
./nib "$bacon"
./nib "$eggs" done
./nib "$bacon" done
resolved_tasks=$(./nib show-resolved)
grep -q "eggs test task" <<<"$resolved_tasks"
grep -q "bacon test task" <<<"$resolved_tasks"

# there should be no staged changes
git -C $NIB_GIT_REPO diff-index --quiet --cached HEAD --

# there should be no un-staged changes
git -C $NIB_GIT_REPO diff-files --quiet

# there should be no untracked files changes (the lock file is untracked by design)
test -z "$(git -C $NIB_GIT_REPO ls-files --others --exclude=.nib.lock)"

# nib should be able to create a git repo if it does not exist before
# executing command. Also help should work before initialisation (and should
# not init)
NIB_GIT_REPO=$(mktemp --directory --dry-run)
./nib help
echo y | ./nib add test
