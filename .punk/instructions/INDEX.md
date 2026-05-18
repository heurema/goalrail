# Punk Instructions

This is a thin local instruction index for this Punk project.

Use it to find the focused instruction page you need. Do not copy every rule
into this file.

## Start here

- [Getting started](pages/getting-started.md)
- [Project layout](pages/layout.md)
- [Init behavior](pages/init.md)
- [Modules](pages/modules.md)
- [Authority and generated views](pages/authority.md)

## Module instructions

Module-specific instruction trees live under `modules/<module-id>/` when a
module is explicitly added later.

No module is active just because this directory exists.

## Page index view

A future derived page index may live at:

```text
.punk/views/instructions/page-index.json
```

That view is rebuildable and advisory. The source instruction pages remain the
thing to inspect.
