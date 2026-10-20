# Performance

nib loads and parses every non-resolved task, each task being a single file.
This may sound wasteful, but it allows git to track history natively and is
performant thanks to modern OS disk caches and SSDs.

If it slows down as the number of non-resolved tasks grows, indexing or
archiving very old tasks are the options to look at. The number of open tasks
is (hopefully) bounded, so this should not be necessary.

TODO add `hyperfine` benchmarks.
