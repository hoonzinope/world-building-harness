# World Building

Use this skill when the user asks to create, revise, validate, or accept worldbuilding canon.

Rules:
- Treat `content/` Markdown as canon and do not edit it directly.
- Put generated material in `drafts/` through `world_create_draft`, `world_create_update_draft`, or `world_create_deprecate_draft`.
- Stage long query, title, body, reason, and retcon reason text with `world_stage_input` before passing file/hash bindings to other tools.
- Validate every draft with `world_validate_draft`.
- Before accepting, run `world_diff_draft`, show the diff binding, stage a reason, receive explicit user approval for both, create approval attestation, then call `world_accept_draft`.
- If validation reports `error` or `conflict`, revise or reject the draft unless the user explicitly requests the force path and the tool allows it.
- Tool JSON is authoritative. Use `ok`, `command_status`, `data.validation_status`, `data.block_reason`, `issues`, and `available_actions` to decide the next step.
- For story ideas that are not canon yet, create `storylet` drafts. To make a story element canon, create entity/event/place drafts and accept them through the normal flow.

Typical create flow:
1. `world_status`
2. `world_stage_input(kind=query)` and `world_search_docs`
3. draft the title/body
4. `world_stage_input(kind=title)` and `world_stage_input(kind=body)`
5. `world_create_draft`
6. `world_validate_draft`
7. summarize `draft_path`, validation status, and next actions
