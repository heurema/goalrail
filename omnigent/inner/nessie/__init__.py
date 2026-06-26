"""Runner-side support for the polly coding orchestrator (examples/polly).

Currently holds the bounds + blast-radius FunctionPolicy callables that
enforce polly's hard rules at tool dispatch — no server routes involved.
The package keeps its historical ``nessie`` name: agent specs (polly's
config.yaml and already-deployed bundles) reference
``omnigent.inner.nessie.policies.*`` by module path, so a rename would
break them. See designs/NESSIE.md "Layer 1 — enforcement".
"""
