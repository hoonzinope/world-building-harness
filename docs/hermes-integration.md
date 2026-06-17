# Hermes Integration

## 1. Role
Hermes is the conversation, automation, and multi-channel orchestration layer for world work. It is not the source of truth for canon, validation, diff generation, or approval binding.

`world-tool` remains the authoritative executor for:
- canon storage and validation
- draft creation and updates
- diff generation
- approval attestation
- accept/reject
- run/audit/recovery

Hermes may draft text, coordinate channels, and sequence tool calls, but it must not be given a general shell surface for world operations.

## 2. Minimal MCP surface
To keep Hermes constrained, expose only the smallest `world-tool` wrapper set needed for the world-building workflow.

Required tools:

- `world_list`
- `world_status`
- `world_stage_input`
- `world_search_docs`
- `world_read_doc`
- `world_create_draft`
- `world_create_update_draft`
- `world_create_deprecate_draft`
- `world_read_draft`
- `world_validate_draft`
- `world_diff_draft`
- `world_create_approval_attestation`
- `world_accept_draft`
- `world_force_accept_draft`
- `world_reject_draft`
- `world_validate_content`
- `world_list_runs`
- `world_get_run`
- `world_get_run_artifact`
- `world_recover_run`

Do not expose a generic shell tool, filesystem write tool, or any broader command runner to Hermes for world content work.

## 3. Data flow rules
- Long query, title, body, reason, and retcon reason text must be staged through `world_stage_input` first.
- Hermes must pass only file/hash bindings to downstream tools.
- Approval provenance must be created with `world_create_approval_attestation`.
- The attestation must bind the exact downstream action: `world_accept_draft` or `world_force_accept_draft`.
- Hermes must treat `world-tool` JSON as authoritative and should not infer hidden state from shell output or prompt text.

## 4. Operational boundary
Hermes can orchestrate across chat, task queues, and external channels, but any state mutation affecting canon must go through `world-tool`.

The intended chain is:

`Hermes -> limited MCP tools -> world-tool -> world root`

Not:

`Hermes -> shell -> world root`

## 5. Recommended usage
Use Hermes for:
- gathering user intent
- drafting candidate text
- coordinating approvals
- preparing channel-specific responses
- triggering world tool calls in order

Use `world-tool` for:
- validating draft content
- comparing diffs
- recording approval attestations
- accepting or rejecting drafts
- recovering partial transactions

## 6. Related contracts
- [docs/system-design.md](system-design.md)
- [docs/opencrabs-integration.md](opencrabs-integration.md)
- [docs/commands.md](commands.md)
- [opencrabs/tools/world-tools.toml](../opencrabs/tools/world-tools.toml)
