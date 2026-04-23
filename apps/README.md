# Apps

Application surfaces for Goalrail implementation.

Current product shape reserves three app-level surfaces:

- Web — intent, contract review, oversight
- CLI — delivery runtime
- Integrations — repo, tracker, runtime settings

Current repo reality:

- shared web rules live in `apps/web/`
- runnable frontend apps live in `apps/web/<resource>`
- `apps/web/demo-change-packet` is the current local React + Vite + Mantine change-packet demo prototype
- the demo is still a prototype and does not imply a finished Goalrail web product surface
