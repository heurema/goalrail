# A confined root follows a symbolic link whose target stays inside it

The round-5 fix descended from the home directory and the design claimed
no path segment could leave the bundle. A confined root refuses a link
that escapes it — including any absolute link — but permits one whose
target stays inside. The claim was therefore stronger than the code held.

Standalone probe, source retained beside this file as
`symlink-inside-home-probe.go.txt`. Run it with `go run`:

```
relative symlink inside the root followed: true (err: <nil>)
bytes read from the planted tree: "PLANTED-TREE"
```

The planted tree is read through the redirected version directory. Every
path segment is now inspected with lstat and a symbolic link at any of
them is refused, so root confinement is no longer the only guarantee.
