# Project Layout

Current local Punk layout:

- `.punk/README.md` - thin project entrypoint.
- `.punk/project.toml` - setup metadata, not runtime authority.
- `.punk/instructions/` - local human and agent instructions.
- `.punk/memory/` - tracked durable Level 0 project memory.

Runtime and derived stores are not created by init unless an explicit later
slice activates them.

Generated views, if present later, are rebuildable views over source artifacts.
