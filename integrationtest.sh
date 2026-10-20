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

# initialse git repo
git -C $NIB_GIT_REPO init -b master
git -C $NIB_GIT_REPO config user.email "you@example.com"
git -C $NIB_GIT_REPO config user.name "Test user"
git -C $UPSTREAM_BARE_REPO init --bare -b master

# general task state management and commands
./nib help
./nib add test task +foo project:bar
./nib start 1
./nib stop 1
echo y | ./nib remove 1
./nib add re-add add test task +foo project:bar
./nib add another task
./nib +foo
./nib -foo
./nib -project:foo
./nib 3 done
./nib undo
./nib log something
./nib +foo
./nib note 2 this is a note
./nib context +foo
./nib next
./nib 2 done
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
./nib add a tasklist task -- "- [ ] incomplete task"
! ./nib 7 done

# test template functions
./nib add task to copy
./nib add template:8 +copiedTask
./nib template 9
# Task 9 should now be a template
./nib show-templates +copiedTask
# copy Template with some modifications
./nib add template:9 -copiedTask +copiedTemplate
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
./nib add eggs test task +foo project:bar
./nib sync
./nib git reset --hard HEAD~1
./nib add bacon test task +foo project:bar
./nib sync

# Known failure, https://github.com/wandering-eye/nib/issues/1: both clones
# allocate the same key and the merge is clean, so the store stops loading.
# Passing fails the suite so the fix is noticed and this wrapper removed.
if diverged_output=$(./nib next 2>&1); then
    echo "issue #1 check passed unexpectedly; remove its known-failure wrapper" >&2
    exit 1
fi
grep -q "keys must be unique" <<<"$diverged_output"
echo "known failure, issue #1: diverged clones share a key, failed as expected"

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
