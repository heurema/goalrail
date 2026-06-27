# goalrail-client

Python client SDK for the [Goalrail](https://github.com/heurema/goalrail)
server API.

`goalrail-client` is a typed client for driving Goalrail sessions over the
server's HTTP + SSE API — creating sessions, sending turns, and streaming
responses. It shares the `StreamEvent` / `SessionStreamEventType` types that the
server emits, so streamed envelopes are validated against a single source of
truth.

It is released in lockstep with the core `goalrail` package at a matching
version while the Python distribution names remain in their compatibility
namespace:

```bash
pip install goalrail-client
```

See the [Goalrail repository](https://github.com/heurema/goalrail) for full
documentation.
