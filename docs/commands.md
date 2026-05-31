# commands.md

# world-tool Commands

## 1. 개요
`world-tool`은 OpenCrabs dynamic tools가 호출하는 Go 단일 바이너리다. OpenCrabs가 판단과 대화를 담당하고, `world-tool`은 파일/설정 변경을 안전하게 수행한다.

모든 명령은 `--json` 출력을 지원해야 한다. OpenCrabs dynamic tool은 stdout JSON만 해석한다.

## 2. 기본 구조
Canonical CLI grammar는 아래와 같다.

```bash
world-tool [global flags] <resource> <action> [command flags]
```

공통 플래그:

```bash
--world <id>            OpenCrabs/world registry의 world id
--root <path>           직접 world root를 지정할 때 사용
--world-id <id>         --root 모드에서 logical world id를 고정할 때 사용
--registry <path>       world registry 파일 경로
--json                  machine-readable JSON output
--run-id <id>           기존 run에 후속 event/artifact 추가
--dry-run               파일 변경 없이 diff와 계획만 출력
--verbose               상세 로그 출력(stderr only)
```

`registry list`와 `world list`는 `--world`/`--root`를 받지 않는다. `registry add`는 `--world`와 `--root`를 함께 받는다. `world init`은 `--root`를 받으며 새 world 생성 시 `--world-id`를 함께 받는다. 그 외 command는 `--world`와 `--root` 중 정확히 하나가 필수이며, 둘 다 지정하면 실패한다.

`--root` 모드에서는 logical world id를 반드시 고정해야 한다. 새 world를 만드는 `world init`은 `--world-id`를 필수로 받으며, 이미 `harness.yaml`이 있는 root를 여는 command는 `harness.yaml`의 world id를 우선 사용한다. `harness.yaml`이 없는데 `--root`만 주어졌다면 `--world-id`가 없으면 실패한다.

`world init`은 아직 `harness.yaml`이 없는 디렉토리에 사용할 수 있는 유일한 command다. 그 외 command에서 `--root`는 symlink를 해석한 absolute path로 normalize한 뒤 `harness.yaml`이 있는 world root여야 하며, JSON envelope는 이 normalize된 root와 resolved world id를 사용해서 항상 deterministically 작성된다.

Canonical command order:
- `world-tool draft validate`
- `world-tool draft diff`
- `world-tool draft accept`
- `world-tool draft reject`
- `world-tool content validate`

`world-tool validate draft`, `world-tool diff draft`, `world-tool accept draft`, `world-tool reject draft`는 사용하지 않는다.

## 3. JSON 출력 계약
모든 command는 `--json` 모드에서 stdout에 JSON 하나만 출력한다. 로그, progress, debug text는 stdout에 쓰지 않는다. 필요하면 stderr에만 쓴다.

공통 envelope:

```json
{
  "schema_version": "world-tool.v1",
  "ok": true,
  "command_status": "completed",
  "command": "draft.create",
  "world_id": "ashen-continent",
  "registry_root": "/host/worlds/ashen-continent",
  "root": "/workspace/world",
  "run_id": "20260530-001",
  "data": {},
  "issues": [],
  "available_actions": []
}
```

Top-level field 규칙:
- 모든 JSON은 `schema_version`, `ok`, `command_status`, `command`, `data`, `issues`, `available_actions`를 가진다.
- world root를 여는 command는 `world_id`, `registry_root`, `root`, `run_id`를 가진다. `world_id`는 `--world`나 `--root`+`--world-id`, 또는 `harness.yaml`에서 resolved된 값이다. `registry_root`는 registry 또는 명시적 `--root`로 선택된 canonical root이고, `root`는 현재 process가 실제로 접근하는 effective root다. container/bind-mount 환경에서는 둘이 다를 수 있다.
- `registry list`, `world list`, `registry add`처럼 world root를 열지 않는 command는 `world_id`, `registry_root`, `root`, `run_id`를 null로 둘 수 있다.
- `ok: false`인 경우 `error.code`와 `error.message`가 필수다.
- validation severity는 top-level에 두지 않고 `data.validation_status`에만 둔다.
- blocked 결과는 `data.block_reason`에 block code를 담고, 필요하면 `data.validation_status`도 함께 반환한다.

`command_status` 허용값:
- `completed`: command가 정상 완료됨
- `blocked`: command는 실행됐고 domain result를 반환했지만, policy/validation/precondition 때문에 상태 변경을 하지 않음. `ok`는 true다. `blocked`는 domain/policy stop에만 사용한다.
- `failed`: CLI/config/path/I/O/lock/transaction/internal 오류로 command 자체가 실패했으며 `ok`는 false다. `PATH_*`, `LOCK_BUSY`, `IO_ERROR`, `TRANSACTION_INCOMPLETE`는 여기에 속한다.

validation 결과는 top-level `command_status`와 섞지 않고 `data.validation_status`에 둔다.

```json
{
  "schema_version": "world-tool.v1",
  "ok": true,
  "command_status": "completed",
  "command": "draft.validate",
  "world_id": "ashen-continent",
  "registry_root": "/host/worlds/ashen-continent",
  "root": "/workspace/world",
  "run_id": "20260530-002",
  "data": {
    "draft_path": "drafts/nations/nation_northern_empire.md",
    "validation_status": "conflict"
  },
  "issues": [],
  "available_actions": ["world_update_draft", "world_reject_draft"]
}
```

blocked envelope:

```json
{
  "schema_version": "world-tool.v1",
  "ok": true,
  "command_status": "blocked",
  "command": "draft.accept",
  "world_id": "ashen-continent",
  "registry_root": "/host/worlds/ashen-continent",
  "root": "/workspace/world",
  "run_id": "20260530-011",
  "data": {
    "draft_path": "drafts/nations/nation_northern_empire.md",
    "validation_status": "conflict",
    "block_reason": "VALIDATION_BLOCKED"
  },
  "issues": [
    {
      "rule": "VR-220",
      "severity": "conflict",
      "message": "conflict blocks accept until the draft is updated",
      "path": "drafts/nations/nation_northern_empire.md"
    }
  ],
  "available_actions": ["world_update_draft", "world_reject_draft"]
}
```

실패 envelope:

```json
{
  "schema_version": "world-tool.v1",
  "ok": false,
  "command_status": "failed",
  "command": "doc.read",
  "world_id": "ashen-continent",
  "registry_root": "/host/worlds/ashen-continent",
  "root": "/workspace/world",
  "run_id": "20260530-003",
  "data": {},
  "error": {
    "code": "PATH_OUTSIDE_ROOT",
    "message": "path is outside selected world root",
    "details": {
      "path": "../../.ssh/id_rsa"
    }
  },
  "issues": [],
  "available_actions": []
}
```

`issues[]` 항목 형식:

```json
{
  "rule": "VR-101",
  "severity": "conflict",
  "message": "id nation_northern_empire already exists in content",
  "path": "drafts/nations/nation_northern_empire.md",
  "field": "id",
  "recommendation": "use --change-type update with --target-id for retcon/update workflow"
}
```

Exit code 정책:

| Exit code | 의미 |
| --- | --- |
| 0 | command가 실행됐고 stdout JSON이 authoritative result다. completed/blocked 같은 domain result는 0을 사용할 수 있다. |
| 2 | CLI argument, registry, config, path boundary 오류 |
| 3 | 파일 I/O, lock 획득 실패, atomic write 실패 |
| 4 | 내부 오류 또는 panic 복구 |

OpenCrabs는 `ok`, `command_status`, `data.validation_status`, `data.block_reason`, `issues`, `available_actions`, `error.code` 순서로 해석한다.

대표 error/block code:
- `INVALID_ARGUMENT`
- `REGISTRY_NOT_FOUND`
- `WORLD_NOT_FOUND`
- `PATH_OUTSIDE_ROOT`
- `PATH_NOT_MARKDOWN`
- `PATH_SCOPE_DENIED`
- `INPUT_HASH_MISMATCH`
- `AUTH_CONTEXT_MISSING`
- `DRAFT_NOT_ACTIVE`
- `DIFF_BINDING_REQUIRED`
- `DIFF_BINDING_MISMATCH`
- `ID_CONFLICT`
- `TARGET_PATH_CONFLICT`
- `STORYLET_NOT_CANON_TARGET`
- `VALIDATION_BLOCKED`
- `MIGRATION_BLOCKED`
- `FORCE_NOT_ALLOWED`
- `LOCK_BUSY`
- `TRANSACTION_INCOMPLETE`
- `IO_ERROR`
- `INTERNAL_ERROR`

`PATH_*`, `LOCK_BUSY`, `IO_ERROR`, `TRANSACTION_INCOMPLETE`는 `failed`로 반환하고, `VALIDATION_BLOCKED`, `MIGRATION_BLOCKED`, `DIFF_BINDING_REQUIRED`, `DIFF_BINDING_MISMATCH`, `DRAFT_NOT_ACTIVE`, `STORYLET_NOT_CANON_TARGET`, `FORCE_NOT_ALLOWED`처럼 정책/도메인 중단인 경우에만 `blocked`를 사용한다. `TRANSACTION_INCOMPLETE`는 partial mutation 가능성을 뜻하므로 `blocked`가 아니라 `failed`와 함께 recovery metadata를 반환한다.

## 4. Path Scope
문서 path 인자는 world root 기준 상대 경로만 허용한다. command별 추가 allowlist는 다음을 따른다.

| Command | 허용 path |
| --- | --- |
| `doc list/read/search --scope content` | `content/**/*.md` |
| `doc list/read/search --scope drafts` | `drafts/**/*.md` |
| `doc list/read/search --scope active` | `content/**/*.md`, `drafts/**/*.md` |
| `doc search --query-file` | `runs/inbox/**` read/consume only for staged query payloads |
| `draft create/update` | generated draft paths under `drafts/**/*.md` from document-types metadata using explicit type+id; staged title/body/retcon_reason inputs under `runs/inbox/**` read/consume only |
| `draft read/validate/diff/accept/reject` | draft paths under `drafts/**/*.md`; staged reason/approval attestation inputs under `runs/inbox/**` read/consume only where the command defines them |
| `input stage` | `runs/inbox/**` write only |
| `approval attest` | `runs/inbox/**` write only for approval attestation artifacts; trusted auth context is read from a wrapper-owned file outside the world root |
| `run list` | immutable run index/summary files only; `runs/inbox/**` excluded |
| `run get` | default redacted manifest only; explicit repeated `--artifact <basename>` allowlist only; sensitive artifacts require privileged access or a separate command; `runs/inbox/**` excluded |
| `content validate` | `content/**/*.md` |

`doc` command는 `runs/`, `archive/`, `raw/`, `schema/`를 읽지 않는다. archive 조회가 필요하면 향후 별도 `archive` resource를 둔다.
`runs/inbox/`는 staging 전용이며 `run get/list`로 normal browsing 하지 않는다.

## 5. registry
```bash
world-tool registry list --json
world-tool registry add --world ashen-continent --root /host/worlds/ashen-continent --title "잿빛 대륙" --json
world-tool registry remove --world ashen-continent --json
world-tool registry default --world ashen-continent --json
```

동작:
- registry 파일 조회와 갱신
- world id 중복 검사
- root path normalize와 존재 여부 검사
- default world 선택

`--registry` 해석 우선순위:
1. command flag `--registry <path>`
2. 환경변수 `WORLD_TOOL_REGISTRY`
3. `~/.opencrabs/worlds.yaml`
4. `~/.config/world-tool/worlds.yaml`

## 6. world
```bash
world-tool world list --json
world-tool world status --world ashen-continent --json
world-tool world init --root ./worlds/ashen-continent --world-id ashen-continent --json
```

동작:
- world root 기본 구조 생성
- `harness.yaml` 생성
- content/drafts/runs/archive 상태 요약

`world init`은 registry를 자동 수정하지 않는다. registry 등록은 `registry add`로 명시적으로 수행한다.

## 7. input
OpenCrabs가 긴 query/title/body/reason/retcon_reason을 직접 shell argv에 넣지 않도록 `runs/inbox/` staging file을 만드는 resource다.

```bash
world-tool input stage --world ashen-continent --kind query --stdin --json
world-tool input stage --world ashen-continent --kind title --stdin --json
world-tool input stage --world ashen-continent --kind body --stdin --json
world-tool input stage --world ashen-continent --kind reason --stdin --json
world-tool input stage --world ashen-continent --kind retcon_reason --stdin --json
```

결과 예시:

```json
{
  "schema_version": "world-tool.v1",
  "ok": true,
  "command_status": "completed",
  "command": "input.stage",
  "world_id": "ashen-continent",
  "registry_root": "/host/worlds/ashen-continent",
  "root": "/workspace/world",
  "run_id": "20260530-004",
  "data": {
    "kind": "body",
    "input_path": "runs/inbox/20260530-004-body.md",
    "input_hash": "sha256:..."
  },
  "issues": [],
  "available_actions": []
}
```

정책:
- `--kind` 허용값은 `query`, `title`, `body`, `reason`, `retcon_reason`, `note`다.
- staging path는 항상 `runs/inbox/` 아래에 tool이 생성한다.
- symlink는 허용하지 않는다.
- 기본 크기 상한은 입력 하나당 1 MiB다.
- 후속 command가 staging file을 읽으면 해당 run artifact로 복사하고 inbox 원본은 삭제하거나 consumed marker를 남긴다.
- staged input을 후속 command가 소비할 때는 반드시 해당 파일 경로와 대응하는 hash를 함께 넘겨야 한다. `world-tool`은 파일을 다시 해시해서 `input_hash`와 비교하고, 불일치하면 `command_status: "failed"`와 `error.code: "INPUT_HASH_MISMATCH"`를 반환한다.

## 8. doc
```bash
world-tool doc list --world ashen-continent --scope active --json
world-tool doc read --world ashen-continent --path content/nations/nation_ashen_empire.md --json
world-tool doc search --world ashen-continent --scope active --query-file runs/inbox/20260530-004-query.txt --query-hash sha256:... --json
```

동작:
- content/drafts 문서 목록
- path boundary와 scope 검사 후 문서 읽기
- title, tag, id, full-text 기반 검색

`--query "text"`는 local CLI 편의 기능으로만 허용한다. OpenCrabs dynamic tools는 `input stage`로 만든 `--query-file`과 `--query-hash`를 함께 사용한다. `--scope`를 생략하면 local CLI에서는 `active`를 기본으로 쓸 수 있지만, canonical dynamic tool mapping에서는 `--scope active`를 명시한다.

## 9. draft
```bash
world-tool draft create \
  --world ashen-continent \
  --change-type create \
  --type nation \
  --id nation_ashen_empire \
  --title-file runs/inbox/20260530-004-title.txt \
  --title-hash sha256:... \
  --body-file runs/inbox/20260530-005-body.md \
  --body-hash sha256:... \
  --json

world-tool draft create \
  --world ashen-continent \
  --change-type update \
  --target-id nation_ashen_empire \
  --title-file runs/inbox/20260530-006-title.txt \
  --title-hash sha256:... \
  --body-file runs/inbox/20260530-007-body.md \
  --body-hash sha256:... \
  --retcon-reason-file runs/inbox/20260530-007-retcon-reason.txt \
  --retcon-reason-hash sha256:... \
  --json

world-tool draft create \
  --world ashen-continent \
  --change-type deprecate \
  --target-id nation_ashen_empire \
  --title-file runs/inbox/20260530-009-title.txt \
  --title-hash sha256:... \
  --body-file runs/inbox/20260530-009-body.md \
  --body-hash sha256:... \
  --retcon-reason-file runs/inbox/20260530-009-retcon-reason.txt \
  --retcon-reason-hash sha256:... \
  --json

world-tool draft update --world ashen-continent --draft drafts/nations/nation_ashen_empire.md --body-file runs/inbox/20260530-008-body.md --body-hash sha256:... --json
world-tool draft list --world ashen-continent --json
world-tool draft read --world ashen-continent --draft drafts/nations/nation_ashen_empire.md --json
```

`change_type` 허용값:
- `create`: 새 canon 문서 생성
- `update`: 기존 canon 문서 갱신 또는 retcon
- `deprecate`: 기존 canon 문서를 deprecated로 변경

정책:
- `draft create`는 `--change-type`이 필수다. `--change-type`이 없거나 `--change-type`과 나머지 인자 조합이 맞지 않으면 CLI argument 오류로 `command_status: "failed"`와 `error.code: "INVALID_ARGUMENT"`를 반환한다.
- `create`는 `--type`과 `--id`가 필수이고 `--target-id`를 받지 않는다.
- `world-tool`은 document-types metadata를 조회해 `type`에 대한 `id` format/prefix를 검증하고, draft/content path를 `type+id`에서 직접 파생한다. title에서 implicit ID를 생성하지 않는다.
- `create`는 같은 id가 content에 있으면 `ID_CONFLICT`로 blocked다.
- `update`와 `deprecate`는 `--target-id`와 `--retcon-reason-file`이 필수이고, `--target-id`가 content에 존재해야 한다.
- `update`와 `deprecate` draft의 id는 `target_id`와 같아야 한다.
- `update`는 기존 content path를 유지한다. rename은 `draft update`의 책임이 아니며 별도 migration command family로 처리한다.
- `create` 계열에서 `--title-file`, `--body-file`, `--retcon-reason-file`은 각각 `--title-hash`, `--body-hash`, `--retcon-reason-hash`와 짝을 이뤄야 하며, `world-tool`은 파일 내용을 다시 해시해서 검증한다.
- 해시 불일치는 `command_status: "failed"`와 `error.code: "INPUT_HASH_MISMATCH"`로 처리한다.
- `update`와 `deprecate`는 `--retcon-reason-file` 내용을 frontmatter `retcon_reason`으로 기록해야 한다. 본문 `Canon Notes`는 보강 설명용이고 frontmatter를 대체하지 않는다.

## 10. content
```bash
world-tool content validate --world ashen-continent --json
world-tool content migrate --world ashen-continent --dry-run --json
```

동작:
- content 전체 구조와 참조 검증
- storylet canon 존재 여부 차단
- schema migration 필요 문서 보고
- `content migrate`는 이 계약에서 report-only다. partial apply나 `--apply`는 허용하지 않는다.
- `content migrate --dry-run`은 content를 변경하지 않고 `runs/<run-id>/migration.json`, `migration.md`, `migration-actions.jsonl`을 생성한다. 반환 `data`에는 최소한 `migration_run_id`, `migration_report_path`, `migration_actions_path`, `candidates`, `blockers`, `partial_apply: false`가 있어야 하고, `available_actions`는 비어 있어야 한다.
- migration blocker가 남아 있으면 `command_status: "blocked"`와 `data.block_reason: "MIGRATION_BLOCKED"`를 반환

## 11. draft validate
```bash
world-tool draft validate --world ashen-continent --draft drafts/nations/nation_ashen_empire.md --json
```

validation status:
- `pass`
- `warning`
- `conflict`
- `error`

validation command는 검증을 정상 완료했다면 `data.validation_status`가 `conflict`나 `error`여도 exit code 0을 반환할 수 있다. CLI/config/path 오류와 구분한다.

## 12. draft diff
```bash
world-tool draft diff --world ashen-continent --draft drafts/nations/nation_ashen_empire.md --json
```

동작:
- accept 시 content에 반영될 변경사항 계산
- `runs/<run-id>/diff.patch` artifact 생성
- target path 충돌 검사
- diff binding 값을 반환

diff result 필수 payload:

```json
{
  "diff_run_id": "20260530-010",
  "draft_path": "drafts/nations/nation_ashen_empire.md",
  "draft_hash": "sha256:...",
  "target_path": "content/nations/nation_ashen_empire.md",
  "target_base_hash": "sha256:...",
  "patch_hash": "sha256:..."
}
```

## 13. approval attest
`approval attest`는 OpenCrabs trusted wrapper가 accept 직전에 호출하는 helper command다. 모델이나 prompt에서 받은 actor 문자열을 승인 provenance로 쓰지 않도록, 인증된 OpenCrabs session/channel metadata와 사용자가 확인한 diff binding을 하나의 staged attestation artifact로 묶는다.

```bash
world-tool approval attest \
  --world ashen-continent \
  --diff-run-id 20260530-010 \
  --draft-hash sha256:... \
  --target-base-hash sha256:... \
  --patch-hash sha256:... \
  --approver-id alice \
  --approval-channel OpenCrabs-chat \
  --auth-context-file /var/lib/opencrabs/auth-contexts/20260530-010.json \
  --auth-context-hash sha256:... \
  --authenticated-actor openid:codex-oauth:user-123 \
  --reason-hash sha256:... \
  --json
```

동작:
- OpenCrabs trusted wrapper/session metadata에서 authenticated actor와 channel provenance를 확인한다.
- `runs/inbox/<run-id>-approval-attestation.json`을 생성하고 `approval_attestation_file`, `approval_attestation_hash`를 반환한다.
- attestation payload에는 `authenticated_actor`, `approver_id`, `approval_channel`, `diff_run_id`, `draft_hash`, `target_base_hash`, `patch_hash`, `reason_hash`, `session_id`, `created_at`을 포함한다.
- trusted auth/session metadata가 없으면 `command_status: "failed"`와 `error.code: "AUTH_CONTEXT_MISSING"`을 반환한다.

정책:
- `--authenticated-actor`는 wrapper가 채운 값이어야 하며, prompt text, model output, staged input, Docker mount path에서 파생하면 안 된다. `world-tool`은 이 flag가 wrapper/request metadata의 authenticated actor와 일치하는지 확인하고, 해당 metadata가 없으면 flag 값과 무관하게 `AUTH_CONTEXT_MISSING`으로 실패한다.
- `--auth-context-file`은 OpenCrabs trusted wrapper가 생성한 auth context 파일을 가리켜야 하며, selected world root 내부나 runtime-owned run directory에 있으면 안 된다. `--auth-context-hash`는 파일의 sha256 해시와 일치해야 하고, 파일의 `expires_at`이 만료되었으면 실패한다.
- auth context 파일에는 `session_id`, `authenticated_actor`, `approval_channel`, `issued_at`, `expires_at`이 들어 있어야 하며, `world-tool`은 파일 내용과 `--approval-channel`, `--authenticated-actor`가 일치하는지 검증한다. `--approver-id`는 non-empty audit field로 검증하고 attestation payload에 기록한다.
- test auth context는 wrapper가 explicit test fixture/mode를 선언한 경우에만 허용한다. fixture-backed context를 기본 운영 경로처럼 취급하면 안 된다.
- `approval attest`가 만든 attestation은 `draft accept`가 소비하기 전까지 `runs/inbox/`에만 머문다. `run get/list`는 이 inbox artifact를 노출하지 않는다.
- local CLI 테스트에서는 wrapper가 제공하는 동일한 auth context mock을 명시적으로 주입해야 하며, 기본 interactive shell 문자열을 신뢰하지 않는다.

## 14. draft accept
```bash
world-tool draft accept \
  --world ashen-continent \
  --draft drafts/nations/nation_ashen_empire.md \
  --diff-run-id 20260530-010 \
  --draft-hash sha256:... \
  --target-base-hash sha256:... \
  --patch-hash sha256:... \
  --approver-id alice \
  --approval-channel OpenCrabs-chat \
  --approval-attestation-file runs/inbox/20260530-011-approval-attestation.json \
  --approval-attestation-hash sha256:... \
  --authenticated-actor openid:codex-oauth:user-123 \
  --reason-file runs/inbox/20260530-011-reason.txt \
  --reason-hash sha256:... \
  --json

world-tool draft accept \
  --world ashen-continent \
  --draft drafts/nations/nation_ashen_empire.md \
  --diff-run-id 20260530-010 \
  --draft-hash sha256:... \
  --target-base-hash sha256:... \
  --patch-hash sha256:... \
  --force \
  --approver-id alice \
  --approval-channel OpenCrabs-chat \
  --approval-attestation-file runs/inbox/20260530-011-approval-attestation.json \
  --approval-attestation-hash sha256:... \
  --authenticated-actor openid:codex-oauth:user-123 \
  --reason-file runs/inbox/20260530-011-reason.txt \
  --reason-hash sha256:... \
  --json
```

동작:
- diff binding 검증
- world root lock 획득
- validation 재실행
- draft hash와 target base hash 재확인
- conflict/error 시 기본 중단
- content atomic write
- accepted draft archive 이동
- transaction result 기록

Accept 정책:
- `draft accept`와 `draft accept --force`는 `--reason-file`, `--reason-hash`, `--approver-id`, `--approval-channel`, `--approval-attestation-file`, `--approval-attestation-hash`, `--authenticated-actor`를 모두 요구한다. `reason-file`만으로는 충분하지 않다.
- `--reason-file`과 `--reason-hash`는 짝을 이뤄야 하며, `world-tool`은 acceptance metadata를 기록하기 전에 재해시해서 검증한다.
- `--authenticated-actor`는 단독으로는 신뢰되지 않는다. CLI는 `--approval-attestation-file`과 `--approval-attestation-hash`를 함께 넘겨야 하며, 이 attestation은 trusted wrapper/session metadata가 생성한 파일이어야 한다. attestation은 `runs/inbox/` 같은 trusted staging area에 있어야 하고, `world-tool`은 파일 해시를 다시 계산한 뒤 attestation 내부의 `authenticated_actor`, `approver_id`, `approval_channel`, `diff_run_id`, `draft_hash`, `target_base_hash`, `patch_hash`, `reason_hash`가 command flags와 일치하는지 검증한 다음에만 `approval.authenticated_actor`를 기록한다.
- `pass`는 explicit approval provenance가 있으면 accept 가능하다.
- `warning`은 기본 accept 가능하지만, warning과 approval provenance를 runs log에 남긴다.
- `conflict`와 `error`는 기본 accept에서 `command_status: "blocked"`로 반환한다.
- `--force`는 semantic/timeline/relationship conflict 후보만 우회할 수 있고 trusted approval attestation과 reason이 필수다. 모든 참조 대상이 이미 canon content일 때만 허용한다.
- `--force`는 active-draft-only 조건, missing related/relationship targets, structural error, id conflict, path violation, target path conflict, inactive draft, storylet canon 승격, diff binding mismatch는 우회할 수 없다.
- `change_type: deprecate`가 accept되면 target content를 `content/` 아래에서 in-place로 deprecated 상태와 deprecation audit metadata로 갱신하고, canon 파일은 제거하지 않는다. 해당 deprecate draft는 accepted draft와 동일하게 archive된다.
- `type: storylet`은 MVP에서 content canon accept 대상이 아니며 `STORYLET_NOT_CANON_TARGET`으로 blocked다.
- accept 성공 시 `data.approval`은 `approver_id`, `approval_channel`, `authenticated_actor`, `approval_attestation_path`, `reason_path`를 포함해야 한다.

Diff binding 정책:
- accept는 `--diff-run-id`, `--draft-hash`, `--target-base-hash`, `--patch-hash`가 모두 없으면 `DIFF_BINDING_REQUIRED`로 blocked다.
- 현재 draft hash가 `--draft-hash`와 다르면 `DIFF_BINDING_MISMATCH`다.
- 현재 target content hash가 `--target-base-hash`와 다르면 `DIFF_BINDING_MISMATCH`다.
- diff artifact의 patch hash가 `--patch-hash`와 다르면 `DIFF_BINDING_MISMATCH`다.

Transaction/recovery 정책:
- accept run은 시작 시 `result.json`을 `pending`으로 기록한다.
- content write는 temp file 작성 후 atomic rename으로 수행한다.
- archive move도 같은 filesystem 안에서 atomic rename을 사용한다.
- content write 성공 후 archive/result 기록이 실패하면 `ok: false`, `command_status: "failed"`, `error.code: "TRANSACTION_INCOMPLETE"`를 반환하고 recovery instruction을 `runs/<run-id>/recovery.json`에 남긴다. 이 경우 partial mutation이 이미 발생했을 수 있다.
- 실패 응답에는 `data.recovery` metadata를 포함하고 `available_actions`는 `["world_recover_run"]`를 가리켜야 한다.
- recovery는 content hash와 archive 상태를 기준으로 idempotent하게 재시도할 수 있어야 한다.
- `runs/<run-id>/recovery.json`가 unresolved인 동안에는 같은 world root를 쓰는 모든 write command가 blocked다. 여기에는 `world init` against that root, `input stage`, `approval attest`, `draft create`, `draft update`, `draft diff`, `draft accept`, `draft reject`, `content validate`가 report/artifact를 쓰는 경우가 포함된다. read-only command는 계속 허용된다.
- `run recover`는 이 차단의 예외이며, `TRANSACTION_INCOMPLETE` 또는 unresolved `recovery.json`를 복구하는 유일한 운영자용 repair command다. 이 command는 원래 write command를 다시 실행하지 않고, 기록된 transaction state만 안전하게 수습한다.

## 15. draft reject
```bash
world-tool draft reject \
  --world ashen-continent \
  --draft drafts/nations/nation_ashen_empire.md \
  --reason-file runs/inbox/20260530-012-reason.txt \
  --reason-hash sha256:... \
  --json
```

동작:
- draft를 `archive/rejected/`로 이동
- runs log에 반려 사유 기록
- `--reason-file`과 `--reason-hash`는 짝을 이뤄야 한다.
- `world-tool`은 rejection reason을 기록하기 전에 `--reason-file`과 `--reason-hash`를 재해시해서 검증한다.

## 16. run
```bash
world-tool run list --world ashen-continent --json
world-tool run get --world ashen-continent --run-id 20260530-001 --json
world-tool run get --world ashen-continent --run-id 20260530-001 --artifact manifest.json --artifact summary.json --json
world-tool run recover --world ashen-continent --run-id 20260530-001 --json
```

동작:
- 최근 실행 목록과 immutable summary만 조회
- `run get`은 기본적으로 redacted manifest와 상태 요약만 반환
- `run get`은 명시적인 safe artifact allowlist만 허용한다. `--artifact <artifact_name>`는 반복 가능하며, basename allowlist에서만 선택할 수 있다. 예: `manifest.json`, `summary.json`, `result-summary.json`, `validation.json`, redacted `recovery.json`
- sensitive artifact, raw staged input, inbox payload, unredacted result/body/reason는 `run get`로 노출하지 않는다. 이런 자료는 별도의 privileged command나 explicit privileged flag가 있어야 한다.
- `world_get_run_artifact`는 `run get`의 explicit safe artifact retrieval 전용 mapping이다. arbitrary path는 허용하지 않는다.
- `run recover`는 `TRANSACTION_INCOMPLETE` 또는 unresolved `recovery.json`를 해결하는 운영자용 command다. 대상 run의 `recovery.json`, 현재 content hash, archive state, result state를 다시 읽고, 이미 복구가 끝났다면 no-op으로 끝낸다.
- `run recover`는 멱등적이어야 하며, 반복 호출해도 duplicate write를 만들지 않는다. 복구가 필요한 경우에만 남은 단계를 수행하고 `recovery.json`를 resolved로 표시한다.
- `run recover`가 성공하면 이후 write command는 다시 허용된다.
- `run recover`의 결과 `data`는 최소한 `recovery_run_id`, `recovery_path`, `recovery_status`, `recovered_at`을 포함해야 하며, `available_actions`는 비어 있어도 된다.

## 17. OpenCrabs Dynamic Tool 매핑
`~/.opencrabs/tools.toml`에는 의미 단위 tool을 등록한다.

아래 예시는 shell executor가 template 값을 argv-safe한 인자로 전달하고 stdin/request-file payload를 지원한다고 가정한다. shell 텍스트 안에 raw string interpolation을 그대로 붙이는 방식은 허용하지 않는다. 그런 executor만 있다면 이 command 문자열을 그대로 쓰지 말고, request JSON file 하나를 받는 wrapper command를 둔다.

```toml
[[tools]]
name = "world_stage_input"
description = "Stage long user/model input into runs/inbox and return input_path and input_hash"
executor = "shell"
command = "world-tool input stage --world {{world_id}} --kind {{kind}} --stdin --json"

[[tools]]
name = "world_list"
description = "List configured worlds"
executor = "shell"
command = "world-tool world list --json"

[[tools]]
name = "world_status"
description = "Return world status and pending draft summary"
executor = "shell"
command = "world-tool world status --world {{world_id}} --json"

[[tools]]
name = "world_search_docs"
description = "Search canon and active draft documents"
executor = "shell"
command = "world-tool doc search --world {{world_id}} --scope active --query-file {{query_file}} --query-hash {{query_hash}} --json"

[[tools]]
name = "world_read_doc"
description = "Read a world document within the selected world root"
executor = "shell"
command = "world-tool doc read --world {{world_id}} --path {{path}} --json"

[[tools]]
name = "world_create_draft"
description = "Create a draft without modifying canon content, using explicit type+id"
executor = "shell"
command = "world-tool draft create --world {{world_id}} --change-type create --type {{type}} --id {{id}} --title-file {{title_file}} --title-hash {{title_hash}} --body-file {{body_file}} --body-hash {{body_hash}} --json"

[[tools]]
name = "world_create_update_draft"
description = "Create an update/retcon draft for an existing canon document"
executor = "shell"
command = "world-tool draft create --world {{world_id}} --change-type update --target-id {{target_id}} --title-file {{title_file}} --title-hash {{title_hash}} --body-file {{body_file}} --body-hash {{body_hash}} --retcon-reason-file {{retcon_reason_file}} --retcon-reason-hash {{retcon_reason_hash}} --json"

[[tools]]
name = "world_create_deprecate_draft"
description = "Create a deprecation draft for an existing canon document"
executor = "shell"
command = "world-tool draft create --world {{world_id}} --change-type deprecate --target-id {{target_id}} --title-file {{title_file}} --title-hash {{title_hash}} --body-file {{body_file}} --body-hash {{body_hash}} --retcon-reason-file {{retcon_reason_file}} --retcon-reason-hash {{retcon_reason_hash}} --json"

[[tools]]
name = "world_update_draft"
description = "Update an active draft without modifying canon content"
executor = "shell"
command = "world-tool draft update --world {{world_id}} --draft {{draft_path}} --body-file {{body_file}} --body-hash {{body_hash}} --json"

[[tools]]
name = "world_read_draft"
description = "Read an active draft"
executor = "shell"
command = "world-tool draft read --world {{world_id}} --draft {{draft_path}} --json"

[[tools]]
name = "world_validate_draft"
description = "Validate a draft against canon"
executor = "shell"
command = "world-tool draft validate --world {{world_id}} --draft {{draft_path}} --json"

[[tools]]
name = "world_diff_draft"
description = "Return the content changes that accept would apply"
executor = "shell"
command = "world-tool draft diff --world {{world_id}} --draft {{draft_path}} --json"

[[tools]]
name = "world_create_approval_attestation"
description = "Create trusted approval attestation from OpenCrabs session metadata and diff binding"
executor = "shell"
command = "world-tool approval attest --world {{world_id}} --diff-run-id {{diff_run_id}} --draft-hash {{draft_hash}} --target-base-hash {{target_base_hash}} --patch-hash {{patch_hash}} --approver-id {{approver_id}} --approval-channel {{approval_channel}} --auth-context-file {{auth_context_file}} --auth-context-hash {{auth_context_hash}} --authenticated-actor {{authenticated_actor}} --reason-hash {{reason_hash}} --json"

[[tools]]
name = "world_accept_draft"
description = "Promote a validated draft into canon after explicit user approval provenance and trusted approval attestation are recorded"
executor = "shell"
command = "world-tool draft accept --world {{world_id}} --draft {{draft_path}} --diff-run-id {{diff_run_id}} --draft-hash {{draft_hash}} --target-base-hash {{target_base_hash}} --patch-hash {{patch_hash}} --approver-id {{approver_id}} --approval-channel {{approval_channel}} --approval-attestation-file {{approval_attestation_file}} --approval-attestation-hash {{approval_attestation_hash}} --authenticated-actor {{authenticated_actor}} --reason-file {{reason_file}} --reason-hash {{reason_hash}} --json"

[[tools]]
name = "world_force_accept_draft"
description = "Force promote a draft when policy allows it and trusted approval attestation is recorded"
executor = "shell"
command = "world-tool draft accept --world {{world_id}} --draft {{draft_path}} --diff-run-id {{diff_run_id}} --draft-hash {{draft_hash}} --target-base-hash {{target_base_hash}} --patch-hash {{patch_hash}} --force --approver-id {{approver_id}} --approval-channel {{approval_channel}} --approval-attestation-file {{approval_attestation_file}} --approval-attestation-hash {{approval_attestation_hash}} --authenticated-actor {{authenticated_actor}} --reason-file {{reason_file}} --reason-hash {{reason_hash}} --json"

[[tools]]
name = "world_reject_draft"
description = "Archive a draft as rejected with a reason"
executor = "shell"
command = "world-tool draft reject --world {{world_id}} --draft {{draft_path}} --reason-file {{reason_file}} --reason-hash {{reason_hash}} --json"

[[tools]]
name = "world_recover_run"
description = "Resolve a partially committed transaction using run recovery state without replaying the original write command"
executor = "shell"
command = "world-tool run recover --world {{world_id}} --run-id {{run_id}} --json"

[[tools]]
name = "world_get_run"
description = "Read the redacted run manifest and status summary"
executor = "shell"
command = "world-tool run get --world {{world_id}} --run-id {{run_id}} --json"

[[tools]]
name = "world_get_run_artifact"
description = "Read one explicit safe run artifact by basename allowlist"
executor = "shell"
command = "world-tool run get --world {{world_id}} --run-id {{run_id}} --artifact {{artifact_name}} --json"
```

OpenCrabs는 query/title/body/reason/retcon_reason을 command template에 직접 넣지 않는다. 먼저 `world_stage_input`으로 `runs/inbox/` path를 만들고, approval attestation은 `world_create_approval_attestation`으로 attestation path/hash를 만든 뒤 후속 tool에는 path와 hash만 넘긴다. `world_search_docs`는 `--scope active`를 명시하고, 각 후속 command는 대응하는 `--*-hash`를 함께 전달해야 한다. `world_get_run_artifact`는 basename allowlist에 있는 `artifact_name`만 읽는다.

모든 template 변수는 OpenCrabs가 넣더라도 신뢰하지 않는다. `world-tool`이 `world_id`, `kind`, `type`, `id`, `scope`, `target_id`, `path`, `draft_path`, `query_file`, `query_hash`, `title_file`, `title_hash`, `body_file`, `body_hash`, `reason_file`, `reason_hash`, `retcon_reason_file`, `retcon_reason_hash`, `approver_id`, `approval_channel`, `auth_context_file`, `auth_context_hash`, `approval_attestation_file`, `approval_attestation_hash`, `authenticated_actor`, `run_id`, `diff_run_id`, `draft_hash`, `target_base_hash`, `patch_hash`, `artifact_name`를 다시 검증한다.
