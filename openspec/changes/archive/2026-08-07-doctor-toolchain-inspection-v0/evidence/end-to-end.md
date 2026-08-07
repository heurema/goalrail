# End-to-end reproduction, before and after

Decoy `node` and `openspec` first on PATH, each appending to a marker.
Command: gr doctor --repo . in a gr-initialized scratch repository.

## Before (23e30a5)
```
EXECUTED node --version
EXECUTED openspec --version
```

## After (this branch)
```
(marker empty — nothing executed)
```

Report line after:
```
planning: ready=false, bundle=incompatible, runtime=missing, compiler=missing
```
