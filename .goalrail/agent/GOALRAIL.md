# Goalrail Agent Pack v0

You are in a Goalrail-initialized repository.

Use the Goalrail CLI as the machine interface. Prefer `--format json` for commands that support it.

Never invent Goalrail state. The Goalrail server owns canonical Intake, Goal, readiness, clarification, Contract, event, gate, proof, and verification state.

For user requests like "start Goalrail work" or pasted Jira/Linear task text, call:

```bash
goalrail work start --title <title> --body-file - --format json
```

Pass pasted task text through stdin as the work body.

If a Goalrail JSON response contains `next_action.available=false`, do not call `next_action.command`. Treat it as a planned future command and explain the current available next step to the user.

If a Goalrail JSON response contains `next_action.available=true` and a `next_action.command`, you may call that command to continue the current Goalrail flow.

If a Goalrail JSON response contains `next_action.kind=ask_user`, render the returned questions to the user. Submit answers with `goalrail work answer` using question_id-bound structured JSON. Do not submit free-form answers without mapping them to returned `question_id` values.

If a Goalrail JSON response contains `next_action.kind=draft_contract` and `next_action.available=true`, call `goalrail contract draft` with the returned Goal ID. The command returns a server Contract handle and a local repository receipt. Do not upload raw source or draft contract fields outside returned Goalrail commands.

If a Goalrail JSON response contains `next_action.kind=update_contract` and `next_action.available=true`, read only the local files needed for the draft and submit structured proposed fields with `goalrail contract update`. Use `question_id`- and field-bound JSON, include local receipt refs when useful, and do not upload raw source bodies.

If a Goalrail JSON response contains `next_action.kind=review_contract`, show the changed draft contract fields to the user for review. Do not submit, approve, plan, run, verify, or create proof unless Goalrail returns an available command for that later state.

If the user explicitly accepts the reviewed draft and Goalrail exposes `goalrail contract submit` as an available command, submit the Contract for approval. This is not approval.

Only call `goalrail contract approve --confirm-user-approval` after the user explicitly approves the submitted Contract. Never infer approval from silence or from a generic continuation request.

If a Goalrail JSON response contains `next_action.kind=plan_work` and `next_action.available=true`, call `goalrail work plan` with the returned Contract ID. This only creates or returns a server WorkItemPlan; newly created plans start queued. It does not acquire a lease, produce a proposal, create WorkItems, run code, verify, or create proof.

If a Goalrail JSON response contains `next_action.kind=review_plan_proposal` and `next_action.available=true`, call `goalrail work plan status` with the returned Plan ID. Show the proposed tasks to the user.

Only call `goalrail work proposal accept --confirm-user-acceptance` after the user explicitly accepts the submitted WorkItemPlanProposal. Never infer plan acceptance from silence or from a generic continuation request.

If a Goalrail JSON response contains `next_action.kind=prepare_checkout` and `next_action.available=true`, call `goalrail work checkout prepare` with the returned WorkItem ID. This creates or returns a server-owned checkout job and checkout instruction only. It does not assign, claim, execute commands, create Run, verify, gate, or create proof.

If a Goalrail JSON response contains `next_action.kind=runner_checkout_required`, explain that a runner process must submit a workspace receipt before execution preparation can use a `checkout_receipt_id`. Do not perform checkout by chat, do not run arbitrary commands as proof, and do not claim execution.

If a Goalrail JSON response or runner handoff includes a `task_id` and `checkout_receipt_id`, and the user asks to prepare execution, call `goalrail work execution prepare`. This creates or returns a server-owned ExecutionJob only. It does not start Run, execute commands, create execution receipt, gate, or proof.

If a Goalrail JSON response contains `next_action.kind=runner_execution_required`, explain that a runner must lease the ExecutionJob, explicitly start a Run, and may submit only a metadata-only no-command ExecutionReceipt. Do not run commands by chat and do not claim command execution, gate, or proof.

After the command returns, show a concise human summary with:

- `intake_id`
- `goal_id`
- `goal_state`

Do not create branches, run agents, run tests, create proof, or claim verification unless Goalrail returns those states.

This Agent Pack is provider-neutral. It is not a Codex, Claude, Gemini, Cursor, Windsurf, or Gravity adapter, plugin, skill, slash command, or provider setting.
