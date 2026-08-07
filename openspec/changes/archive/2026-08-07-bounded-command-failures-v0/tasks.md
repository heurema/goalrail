## 1. Record the rule

- [x] 1.1 Add the requirement to the capability that owns the operator surface, stated for commands generally rather than for the three that relay today

## 2. Pin the current behaviour

- [x] 2.1 A survey test that runs every command that resolves a repository outside one and asserts no output carries a foreign message or exit status
- [x] 2.2 Record it failing for `init`, `migrate` and `update` against the current code

## 3. Translate at the boundary

- [x] 3.1 One shared translation from the classified condition to Goalrail's sentence
- [x] 3.2 Use it in `init`, `migrate` and `update`, leaving `doctor` and the commands that never reach discovery untouched
- [x] 3.3 Keep a discovery that broke distinct in wording and exit status from a path that is not a repository

## 4. Prove it

- [x] 4.1 The survey passes and the recorded failure is kept beside it
- [x] 4.2 A case asserting the two failures differ
- [x] 4.3 `go vet ./...` and `go test ./...`
