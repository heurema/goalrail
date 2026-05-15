# Init

`punk init <project-id>` initializes the current directory in place.

It does not create a new subdirectory named `<project-id>`.

Default greenfield init writes compact Level 0 project memory and thin
instruction entrypoints.

`punk init <project-id> --mode brownfield` writes an advisory brownfield entry
scaffold. It does not scan the repository, reconstruct project truth, generate
contracts, write gate decisions, or create proof.
