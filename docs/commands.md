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
--dry-run               content migrate에만 허용되는 plan/report 플래그; 다른 command에서 사용하면 INVALID_ARGUMENT 실패
--verbose               상세 로그 출력(stderr only)
```

`registry list`와 `world list`는 `--world`/`--root`를 받지 않는 null-root command다. 이들은 `world_id`, `registry_root`, `root`, `run_id`를 모두 null로 둔다. `registry remove`와 `registry default`는 world root를 열지 않지만 target id를 지정하기 위해 `--world <id>`가 필수다. 이들은 `world_id`에 target id를 담고 `registry_root`, `root`, `run_id`는 null로 둔다. `registry add`도 registry-only mutation이며 world root를 열지 않지만 selected world id를 `world_id`에 담고 `registry_root`, `root`, `run_id`는 null로 둔다. `world init`은 `--root`를 받으며 새 world 생성 시 `--world-id`를 함께 받는다. 그 외 command는 `--world`와 `--root` 중 정확히 하나가 필수이며, 둘 다 지정하면 실패한다.

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
- world root를 여는 command는 `world_id`, `registry_root`, `root`, `run_id`를 가진다. `world_id`는 `--world`나 `--root`+`--world-id`, 또는 `harness.yaml`에서 resolved된 값이다. `registry_root`는 registry 또는 명시적 `--root`/harness provenance로 확인된 host canonical root이고, `root`는 현재 process가 실제로 접근하는 effective root다. container/bind-mount 환경에서는 둘이 다를 수 있다. `--root` 모드에서 host canonical root provenance가 없으면 `registry_root`는 `null`로 두고 `root`만 기록한다.
- `registry list`와 `world list`는 `world_id`, `registry_root`, `root`, `run_id`를 모두 null로 둔다. `registry add`, `registry remove`, `registry default`는 world root를 열지 않지만 `world_id`에 selected/target id를 담고 `registry_root`, `root`, `run_id`는 null이다.
- `ok: false`인 경우 `error.code`와 `error.message`가 필수다.
- validation severity는 top-level에 두지 않고 `data.validation_status`에만 둔다.
- `data.block_reason`은 `command_status: "blocked"`일 때만 사용한다. `VALIDATION_BLOCKED`는 validation conflict/error를 묶는 aggregate `data.block_reason` 전용 code이고, section 3.3에 나열된 command-level blocked code는 `data.block_reason`으로 사용할 수 있다. validation/domain cause code는 `issues[].code` 또는 equivalent issue field에 넣는다. `MISSING_TARGET`는 예외적으로 `draft create --change-type update|deprecate`, `draft diff`, `draft accept`에서는 blocked `data.block_reason`으로도 쓰이고, `draft validate` completed 결과에서는 validation issue `issues[].code`로도 쓸 수 있다.

Minimum `data` shape:

| Command | 최소 `data` 필드 | Redaction 원칙 |
| --- | --- | --- |
| `world status` | `world_id`, `root`, `registry_root`, `summary` | summary는 run/draft 상세 본문을 노출하지 않는다 |
| `world list` | `worlds` | 비식별 목록만 제공한다 |
| `registry add/list/remove/default` | `world_id` 또는 `worlds`, `registry_root` when applicable | path와 registry internals는 필요한 최소만 노출한다 |
| `input stage` | `kind`, `input_path`, `input_hash` | input 본문은 반환하지 않는다 |
| `doc read/search` | `path` 또는 `results` | raw body는 explicit read에서만, search는 snippet만 허용한다 |
| `draft create/update/read/list/validate/diff/accept/reject` | `draft_path`, `draft_hash` or `drafts`, `validation_status` when applicable, `diff_*`/`approval` only where relevant | reason/body/attestation payload는 hash와 path 중심으로 redact한다 |
| `approval attest` | `approval_attestation_file`, `approval_attestation_hash`, `world_id`, `issuer`, `audience`, `scope_verification`, `downstream_action`, `reason_hash`, `authenticated_actor`, `approver_id`, `approval_channel`, `issued_at`, `expires_at`, `diff_run_id`, `draft_hash`, `target_base_hash`, `patch_hash` | auth context 원문과 session secret은 반환하지 않고, `scope_verification`에는 `world_create_approval_attestation`과 downstream action allowlist를 요약해 담는다 |
| `content validate` | `validation_status`, `blockers`, `findings` or equivalent summary | content 본문 전체는 반환하지 않는다 |
| `content migrate --dry-run` | `migration_run_id`, `migration_report_path`, `migration_actions_path`, `candidates`, `blockers`, `partial_apply` | report artifact 본문은 raw dump 대신 요약과 경로 중심으로 노출한다 |
| `run list/get/get artifact/recover` | `runs` or `manifest`/`status_summary` or single artifact fields or `recovery_*` | staged inbox payload와 unredacted sensitive artifact는 노출하지 않는다 |

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
  "issues": [
    {
      "code": "TIMELINE_CONFLICT",
      "rule": "VR-220",
      "severity": "conflict",
      "message": "timeline conflict blocks accept until the draft is updated",
      "path": "drafts/nations/nation_northern_empire.md"
    }
  ],
  "available_actions": ["world_update_draft", "world_reject_draft", "world_diff_draft", "world_validate_draft"]
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
      "code": "TIMELINE_CONFLICT",
      "rule": "VR-220",
      "severity": "conflict",
      "message": "timeline conflict blocks accept until the draft is updated",
      "path": "drafts/nations/nation_northern_empire.md"
    }
  ],
  "available_actions": ["world_update_draft", "world_reject_draft", "world_diff_draft", "world_validate_draft"]
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
  "code": "ID_CONFLICT",
  "rule": "VR-101",
  "severity": "conflict",
  "message": "id nation_northern_empire already exists in content",
  "path": "drafts/nations/nation_northern_empire.md",
  "field": "id",
  "recommendation": "use --change-type update with --target-id for retcon/update workflow"
}
```

`issues[]`의 canonical issue code는 `code` 필드다. `rule`, `severity`, `message`, `path`, `field`, `recommendation`은 설명용 메타데이터다. `rule`에는 `VR-*` 같은 validation rule id를 두고, `code`에는 symbolic issue code를 둔다.

Exit code 정책:

| Exit code | 의미 |
| --- | --- |
| 0 | command가 실행됐고 stdout JSON이 authoritative result다. completed/blocked 같은 domain result는 0을 사용할 수 있다. |
| 2 | CLI argument, registry, config, path boundary 오류 |
| 3 | 파일 I/O, lock 획득 실패, atomic write 실패 |
| 4 | 내부 오류 또는 panic 복구 |

OpenCrabs는 `ok`, `command_status`, `data.validation_status`, `data.block_reason`, `issues`, `available_actions`, `error.code` 순서로 해석한다.

### 3.1 MVP code registry

| Code | 분류 | validation issue로도 사용 | 비고 |
| --- | --- | --- | --- |
| `INVALID_ARGUMENT` | failed | 아니오 | CLI argument, flag combination, unsupported mode |
| `REGISTRY_NOT_FOUND` | failed | 아니오 | registry/config lookup failure |
| `WORLD_NOT_FOUND` | failed | 아니오 | world root/registry lookup failure |
| `PATH_OUTSIDE_ROOT` | failed | 아니오 | path boundary failure |
| `PATH_NOT_MARKDOWN` | failed | 아니오 | non-markdown path rejection |
| `PATH_SCOPE_DENIED` | failed | 아니오 | scope boundary rejection |
| `INPUT_HASH_MISMATCH` | failed | 아니오 | staged input hash mismatch |
| `AUTH_CONTEXT_MISSING` | failed | 아니오 | trusted auth context missing |
| `AUTH_CONTEXT_HASH_MISMATCH` | failed | 아니오 | auth context hash mismatch |
| `AUTH_CONTEXT_EXPIRED` | failed | 아니오 | auth context expiry exceeded |
| `AUTH_CONTEXT_SCOPE_DENIED` | failed | 아니오 | auth context scope or exact downstream-action binding mismatch for world/action/issuer/audience bindings |
| `AUTH_CONTEXT_TEST_MODE_REQUIRED` | failed | 아니오 | fixture opt-in missing |
| `APPROVAL_ATTESTATION_HASH_MISMATCH` | failed | 아니오 | staged approval attestation hash mismatch at accept time |
| `APPROVAL_ATTESTATION_EXPIRED` | failed | 아니오 | accept-time staged approval attestation expiry exceeded |
| `APPROVAL_ATTESTATION_BINDING_MISMATCH` | failed | 아니오 | staged approval attestation payload mismatch against command/world/diff/reason/actor/channel/scope bindings |
| `ID_CONFLICT` | blocked | 예 | canonical id already exists; create blocked reason 또는 validation issue code로 사용 |
| `TARGET_PATH_CONFLICT` | blocked | 예 | target path would collide before mutation; accept blocked reason 또는 validation issue code로 사용 |
| `MISSING_TARGET` | blocked | 예 | missing canon target; `draft create --change-type update|deprecate`, `draft diff`, `draft accept`의 blocked reason으로도, `draft validate` completed 결과의 validation issue code로도 쓰인다 |
| `DRAFT_NOT_ACTIVE` | blocked | 예 | draft not active at accept/diff time |
| `DIFF_BINDING_REQUIRED` | blocked | 예 | accept missing diff binding inputs |
| `DIFF_BINDING_MISMATCH` | blocked | 예 | accept diff binding mismatch |
| `STORYLET_NOT_CANON_TARGET` | blocked | 예 | storylet cannot be canonical accept target |
| `TIMELINE_CONFLICT` | validation issue | 예 | validation timeline conflict; use in issues[].code, not data.block_reason |
| `VALIDATION_BLOCKED` | blocked | 아니오 | accept blocked by validation stop envelope; use underlying validation issue codes in issues[].code |
| `MIGRATION_BLOCKED` | blocked | 예 | content migration blocker |
| `FORCE_NOT_ALLOWED` | blocked | 예 | `draft accept --force`가 policy-non-overridable validation/domain stop에 부딪힐 때만 사용; `MISSING_TARGET`, `DIFF_BINDING_REQUIRED`, `DIFF_BINDING_MISMATCH`, `DRAFT_NOT_ACTIVE`, `STORYLET_NOT_CANON_TARGET`, `TARGET_PATH_CONFLICT`, `LOCK_BUSY`, `IO_ERROR`, `TRANSACTION_INCOMPLETE` 같은 기존 non-bypassable precondition/binding/infra failure는 원래 코드 유지 |
| `LOCK_BUSY` | failed | 아니오 | lock contention |
| `TRANSACTION_INCOMPLETE` | failed | 아니오 | partial mutation and recovery metadata required |
| `IO_ERROR` | failed | 아니오 | filesystem or persistence failure |
| `INTERNAL_ERROR` | failed | 아니오 | panic or unexpected internal failure |

`data.block_reason`는 command-level stop 이유이고 `issues[].code`는 underlying validation/domain cause code다. 둘을 섞지 않는다. `VALIDATION_BLOCKED`는 validation conflict/error를 묶는 aggregate `data.block_reason` 전용 code이고, section 3.3에 나열된 command-level blocked code만 `data.block_reason`으로 허용한다. `MISSING_TARGET`는 예외적으로 blocked reason과 validation issue code 양쪽에서 문맥에 따라 쓸 수 있다. `FORCE_NOT_ALLOWED`는 `draft accept --force`가 policy상 non-overridable인 validation/domain stop에 부딪힐 때만 `data.block_reason`으로 사용하고, `MISSING_TARGET`, `DIFF_BINDING_REQUIRED`, `DIFF_BINDING_MISMATCH`, `DRAFT_NOT_ACTIVE`, `STORYLET_NOT_CANON_TARGET`, `TARGET_PATH_CONFLICT`, `LOCK_BUSY`, `IO_ERROR`, `TRANSACTION_INCOMPLETE` 같은 기존 non-bypassable precondition/binding/infra failure는 원래 코드로 유지한다. `draft validate`는 validation 자체가 완료되면 항상 completed로 끝나며, `issues[].code`만 채울 수 있다.

### 3.2 available_actions enum and recommended mapping

허용값은 최소한 다음을 포함한다: `world_update_draft`, `world_read_draft`, `world_accept_draft`, `world_force_accept_draft`, `world_reject_draft`, `world_create_approval_attestation`, `world_recover_run`, `world_get_run`, `world_get_run_artifact`, `world_stage_input`, `world_diff_draft`, `world_validate_draft`.

| 상태/결과 | recommended `available_actions` |
| --- | --- |
| validation `pass` | `world_diff_draft`, `world_update_draft`, `world_reject_draft` |
| validation `warning` | `world_diff_draft`, `world_update_draft`, `world_reject_draft` |
| validation `conflict` | `world_update_draft`, `world_reject_draft`, `world_diff_draft`, `world_validate_draft` |
| validation `error` | `world_update_draft`, `world_reject_draft`, `world_validate_draft` |
| `MISSING_TARGET` blocked on create/update/diff/accept | `world_update_draft`, `world_reject_draft`, `world_diff_draft`, `world_validate_draft` |
| `DIFF_BINDING_REQUIRED` or `DIFF_BINDING_MISMATCH` | `world_diff_draft`, `world_validate_draft`, `world_update_draft` |
| `TRANSACTION_INCOMPLETE` or recovery-needed state | `world_recover_run`, `world_get_run_artifact` |
| safe redacted run artifact inspection | `world_get_run`, `world_get_run_artifact` |

`world_get_run_artifact`는 redacted run artifact 조회 전용이다. staged inbox input이나 approval-attestation payload inspection은 여기서 제공하지 않는다.
`world_accept_draft`는 fresh diff가 다시 생성되고, 그 diff에 정확히 바인딩된 `world_create_approval_attestation`이 downstream action까지 성공적으로 만든 뒤에만 광고할 수 있다. `validation pass`와 `warning`은 그 전 단계의 탐색용으로만 `world_diff_draft`/`world_update_draft`/`world_reject_draft`를 권장한다.
`DIFF_BINDING_REQUIRED`와 `DIFF_BINDING_MISMATCH`에서는 attestation이 아직 유효한 현재 diff에 바인딩될 수 없으므로 `world_create_approval_attestation`를 권장하지 않는다. 먼저 `world_diff_draft`로 fresh diff를 다시 만들고, 필요하면 `world_validate_draft`와 `world_update_draft`로 정리한 뒤 attestation을 생성해야 한다.
`world_force_accept_draft`는 force가 정책상 허용되고, 현재 diff/reason/actor/channel에 바인딩된 fresh, non-expired approval attestation이 있으며 attestation의 `downstream_action`이 `world_force_accept_draft`로 exact match할 때만 권장된다. `world_accept_draft`용 attestation은 force 권한을 대체하지 않는다.

### 3.3 command별 block_reason / issue code 매핑

| Command / case | `command_status` | `data.block_reason` | issue code / note |
| --- | --- | --- | --- |
| `draft create` id 중복 | `blocked` | `ID_CONFLICT` | issue code는 필요 시 동일 코드 또는 equivalent |
| `draft create --change-type update|deprecate` missing canon target | `blocked` | `MISSING_TARGET` | no-write; accept/diff와 동일하게 blocked reason으로 사용 |
| `draft validate` missing target | `completed` | 없음 | `issues[].code: MISSING_TARGET` |
| `draft diff` target 계열 누락 | `blocked` | `MISSING_TARGET` | related/relationship/active-draft-only 포함; blocked reason으로 사용 |
| `draft accept` target 계열 누락 | `blocked` | `MISSING_TARGET` | related/relationship/active-draft-only 포함; blocked reason으로 사용 |
| `draft accept` validation internal conflict/error | `blocked` | `VALIDATION_BLOCKED` | issue code detail는 `TIMELINE_CONFLICT` 같은 underlying validation code |
| `draft accept` diff binding mismatch | `blocked` | `DIFF_BINDING_MISMATCH` | issue code detail에 mismatch 원인 기록 |
| `draft accept` missing diff binding | `blocked` | `DIFF_BINDING_REQUIRED` | missing `--diff-run-id`/`--draft-hash`/`--target-base-hash`/`--patch-hash`; no mutation |
| `draft accept` approval attestation hash mismatch | `failed` | 없음 | accept-time provenance/artifact failure; `error.code: APPROVAL_ATTESTATION_HASH_MISMATCH`; staged attestation file hash does not match supplied hash |
| `draft accept` approval attestation expiry exceeded | `failed` | 없음 | accept-time provenance/artifact failure; `error.code: APPROVAL_ATTESTATION_EXPIRED`; staged attestation expired before content mutation |
| `draft accept` approval attestation payload mismatch | `failed` | 없음 | accept-time provenance/artifact failure; `error.code: APPROVAL_ATTESTATION_BINDING_MISMATCH`; attestation payload, command/world/diff/reason/actor/channel/scope bindings must all match |
| `draft accept` inactive draft | `blocked` | `DRAFT_NOT_ACTIVE` | issue code detail |
| `draft accept` storylet canon | `blocked` | `STORYLET_NOT_CANON_TARGET` | issue code detail |
| `draft accept` target path conflict before mutation | `blocked` | `TARGET_PATH_CONFLICT` | issue code detail은 `TARGET_PATH_CONFLICT` 또는 equivalent validation issue code |
| `draft accept --force` policy-non-overridable validation/domain stop | `blocked` | `FORCE_NOT_ALLOWED` | force로도 우회할 수 없는 validation/domain stop에만 사용하고, non-bypassable precondition/binding/infra failure는 기존 코드 유지 |

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
| `run get` | default redacted manifest only; explicit single `--artifact <basename>` allowlist only; multiple artifacts require repeated `run get` calls; sensitive artifacts require privileged access or a separate command; `runs/inbox/**` excluded |
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

`registry list`는 `world_id`, `registry_root`, `root`, `run_id`를 null로 두고, `registry add`는 selected world id를 `world_id`에 담은 채 `registry_root`, `root`, `run_id`를 null로 둔다. `registry remove`와 `registry default`는 target id를 `world_id`에 담은 채 `registry_root`, `root`, `run_id`를 null로 둔다.
`registry add`는 null-root envelope이며 `--world`, `--root`, `--title`을 모두 필수로 받는다. `registry remove`와 `registry default`는 target id를 지정하는 `--world <id>`를 필수로 받는다. `registry list`와 `world list`는 `--world`/`--root`를 받지 않는다.

`--registry` 해석 우선순위:
1. command flag `--registry <path>`
2. 환경변수 `WORLD_TOOL_REGISTRY`
3. `~/.opencrabs/worlds.yaml`
4. `~/.config/world-tool/worlds.yaml`

밖으로 나가는 registry/config 접근 가드레일:
- world root 밖의 registry/config read/write는 위 4개 경로만 허용한다. 그 외의 home/config 탐색은 허용하지 않는다.
- resolved registry path는 사용 전에 normalize하고 symlink를 검사해야 한다.
- registry/config 해석은 일반적인 home/config discovery로 확장하면 안 된다.
- 선택된 world root 밖의 content/staging/run path는 어떤 경우에도 허용하지 않는다. absolute path, symlink 경유, parent traversal 모두 거부한다.

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

`world list`는 registry-only/null-root command라서 `world_id`, `registry_root`, `root`, `run_id`를 null로 둔다.
`world list`는 `--world`/`--root`를 받지 않는다.

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
- `input.stage`의 JSON 출력은 항상 `input_path`와 `input_hash`를 유지한다. OpenCrabs dynamic tools는 이 값을 `{{kind}}_file`과 `{{kind}}_hash`로 재매핑해서 후속 tool call에 넘긴다. 예를 들어 `kind: query`는 `query_file`/`query_hash`, `kind: title`은 `title_file`/`title_hash`가 된다.

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
- `update`와 `deprecate`는 `--target-id`가 explicit draft/content id다. 이 change type에서는 `--id`를 받지 않으며, `--id`를 넘기면 `INVALID_ARGUMENT`로 실패한다. `draft create --change-type update|deprecate --target-id ...`에서 target id가 canon content에 없으면 draft를 쓰지 않고 `command_status: "blocked"`, `data.block_reason: "MISSING_TARGET"`, `data.validation_status: "conflict"`를 반환한다.
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
- `content migrate`는 post-MVP hardening command contract다. partial apply나 `--apply`는 허용하지 않으며, post-MVP에서도 반드시 `--dry-run`으로만 호출한다.
- `content migrate --dry-run`만 `--dry-run`을 허용한다. `draft diff`는 이미 read/report command라 `--dry-run`을 받지 않으며, `content migrate`를 `--dry-run` 없이 호출하면 `INVALID_ARGUMENT`로 실패한다.
- `content migrate --dry-run`은 content를 변경하지 않지만 `runs/<run-id>/migration.json`, `migration.md`, `migration-actions.jsonl` 같은 run/report artifacts는 생성할 수 있다. 반환 `data`에는 최소한 `migration_run_id`, `migration_report_path`, `migration_actions_path`, `candidates`, `blockers`, `partial_apply: false`가 있어야 하고, `available_actions`는 비어 있어야 한다.
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

validation command는 검증을 정상 완료했다면 `data.validation_status`가 `conflict`나 `error`여도 exit code 0을 반환할 수 있다. CLI/config/path 오류와 구분한다. 이미 존재하는 invalid draft를 `draft validate`로 검사해도 command 자체는 검증 결과를 반환하므로 `command_status: "completed"`, `data.validation_status: "conflict"`, `issues[].code: "MISSING_TARGET"` 또는 equivalent issue code로 보고한다. 이 경우 `data.block_reason`은 사용하지 않는다.

`draft validate`는 missing target을 validation issue로만 보고한다. blocked 결과가 아니라 completed 결과에서 `issues[].code`로 반환해야 하며, missing target을 이유로 `data.block_reason`을 채우지 않는다.

## 12. draft diff
```bash
world-tool draft diff --world ashen-continent --draft drafts/nations/nation_ashen_empire.md --json
```

동작:
- accept 시 content에 반영될 변경사항 계산
- `runs/<run-id>/diff.patch` artifact 생성
- target path 충돌 검사
- diff binding 값을 반환

diff result 필수 payload (update/deprecate 기준):

```json
{
  "diff_run_id": "20260530-010",
  "draft_path": "drafts/nations/nation_ashen_empire.md",
  "draft_hash": "sha256:...",
  "target_exists": true,
  "target_path": "content/nations/nation_ashen_empire.md",
  "target_base_hash": "sha256:...",
  "patch_hash": "sha256:..."
}
```

- `create` diff에서는 `target_exists: false`, `target_base_hash: null`을 반환한다.
- `update`와 `deprecate` diff에서는 `target_exists: true`, `target_base_hash`가 sha256이어야 한다.
- `update`와 `deprecate` diff는 target canon이 없으면 `MISSING_TARGET`로 blocked다. related target, relationship target, active-draft-only related/relationship target도 canon target이 아니면 `MISSING_TARGET`로 blocked다.
- `draft diff`는 이미 dry-run성 read/report command라 `--dry-run`을 받지 않는다.

## 13. approval attest
`approval attest`는 OpenCrabs trusted wrapper가 accept 직전에 호출하는 helper command다. 모델이나 prompt에서 받은 actor 문자열을 승인 provenance로 쓰지 않도록, 인증된 OpenCrabs session/channel metadata와 사용자가 확인한 diff binding, 그리고 invoked downstream action을 하나의 staged attestation artifact로 묶는다. create diff/accept 경로에서는 dynamic tool layer가 diff JSON의 `target_base_hash: null`을 CLI template 변수 `target_base_hash="none"`으로 매핑해 `--target-base-hash none`을 사용하고, attestation payload와 JSON 결과에서는 `target_base_hash`를 null로 정규화한다. production trusted auth context input은 `world_create_approval_attestation`의 root-of-trust이며, `allowed_actions`/scope, issuer, audience, actor, channel, issued_at, expires_at와 함께 exact `downstream_action`(`world_accept_draft` 또는 `world_force_accept_draft`)을 포함하거나 cryptographically bind해야 한다. `--downstream-action`은 이 signed/MACed/trust-material-verified input과 staged attestation payload 둘 다와 일치해야 하며, auth context input의 exact downstream action이 없거나 `--downstream-action`과 mismatch면 `AUTH_CONTEXT_SCOPE_DENIED`로 실패해야 한다. `downstream_action`은 실제로 호출할 downstream command와 정확히 일치해야 하며, allowlist가 둘 다를 포함하더라도 attestation의 `downstream_action`이 `world_accept_draft` 또는 `world_force_accept_draft` 중 현재 호출과 정확히 일치하지 않으면 안 된다.

```bash
world-tool approval attest \
  --world ashen-continent \
  --diff-run-id 20260530-010 \
  --draft-hash sha256:... \
  --target-base-hash sha256:... \
  --patch-hash sha256:... \
  --approver-id alice \
  --approval-channel OpenCrabs-chat \
  --downstream-action world_accept_draft \
  --auth-context-file /var/lib/opencrabs/auth-contexts/20260530-010.json \
  --auth-context-hash sha256:... \
  --authenticated-actor openid:codex-oauth:user-123 \
  --reason-hash sha256:... \
  --json
```

동작:
- OpenCrabs trusted wrapper/session metadata에서 authenticated actor와 channel provenance를 확인한다.
- `runs/inbox/<run-id>-approval-attestation.json`을 생성하고 `approval_attestation_file`, `approval_attestation_hash`를 반환한다.
- attestation payload에는 `world_id`, `authenticated_actor`, `approver_id`, `approval_channel`, `issuer`, `audience`, `scope_verification`, `downstream_action`, `diff_run_id`, `draft_hash`, `target_base_hash`, `patch_hash`, `reason_hash`, `issued_at`, `expires_at`, `session_id`, `created_at`을 포함한다.
- create diff/accept은 `--target-base-hash none`을 사용하고 attestation payload와 JSON 결과에서는 `target_base_hash: null`로 기록한다. update/deprecate는 sha256 해시를 사용한다.
- trusted auth/session metadata가 없으면 `command_status: "failed"`와 `error.code: "AUTH_CONTEXT_MISSING"`을 반환한다. hash mismatch는 `AUTH_CONTEXT_HASH_MISMATCH`, expiry 초과는 `AUTH_CONTEXT_EXPIRED`, world/action/issuer/audience scope mismatch는 `AUTH_CONTEXT_SCOPE_DENIED`, fixture opt-in 누락은 `AUTH_CONTEXT_TEST_MODE_REQUIRED`로 실패한다.

정책:
- `--authenticated-actor`는 wrapper가 채운 값이어야 하며, prompt text, model output, staged input, Docker mount path에서 파생하면 안 된다. `world-tool`은 이 flag가 wrapper/request metadata의 authenticated actor와 일치하는지 확인하고, 해당 metadata가 없으면 flag 값과 무관하게 `AUTH_CONTEXT_MISSING`으로 실패한다.
- `--auth-context-file`은 OpenCrabs trusted wrapper가 생성한 production auth context envelope를 가리켜야 하며, wrapper-signed 또는 MACed envelope이거나 configured wrapper trust material로 검증 가능한 equivalent여야 한다. selected world root 내부나 runtime-owned run directory에 있으면 안 된다. `--auth-context-hash`는 파일의 sha256 해시와 일치해야 하지만, hash/expiry 일치만으로는 production authorization이 되지 않는다. `expires_at`이 만료되었으면 실패한다.
- auth context 파일에는 최소 `world_id`, `allowed_actions` 또는 equivalent scope, `issuer`, `audience`, `session_id`, `authenticated_actor`, `approval_channel`, `issued_at`, `expires_at`, `downstream_action`이 들어 있어야 하며, `world-tool`은 파일 내용과 `--world`, `--approval-channel`, `--authenticated-actor`, `--downstream-action`이 일치하는지 검증한다. `allowed_actions`는 최소한 `world_create_approval_attestation`과 downstream action(`world_accept_draft` 또는 `world_force_accept_draft`)을 포함해야 한다. `scope_verification`은 이 allowlist를 요약해 담고, auth context input의 exact downstream action이 없거나 `--downstream-action`과 mismatch면 `AUTH_CONTEXT_SCOPE_DENIED`로 실패해야 한다. staged approval attestation이 accept/force 시점의 command/world/diff/reason/actor/channel binding과 payload를 다시 검증할 때는 `APPROVAL_ATTESTATION_BINDING_MISMATCH`를 사용한다. `world_accept_draft`는 `downstream_action: world_accept_draft`가, `world_force_accept_draft`는 `downstream_action: world_force_accept_draft`가 정확히 바인딩되어야 하며 allowlist가 둘 다를 포함하더라도 이 exact match를 대체할 수 없다. `AUTH_CONTEXT_SCOPE_DENIED`는 auth context의 scope가 `world_create_approval_attestation`과 exact downstream action을 커버하지 못하거나 `--world`/`--approval-channel`/`--authenticated-actor`/`issuer`/`audience` binding이 일치하지 않을 때 사용한다. `auth_context_hash` mismatch는 `AUTH_CONTEXT_HASH_MISMATCH`, `expires_at` 초과는 `AUTH_CONTEXT_EXPIRED`를 사용한다. `--approver-id`는 non-empty audit field로 검증하고 attestation payload에 기록한다.
- test auth context는 `fixture_mode: true`를 명시한 local fixture와 `WORLD_TOOL_TEST_AUTH_CONTEXT=1` 환경변수가 둘 다 있을 때만 허용한다. 둘 중 하나라도 없으면 `AUTH_CONTEXT_TEST_MODE_REQUIRED` failed다. fixture-backed context는 test-only이며 production attestation provenance로 승격할 수 없다.
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
- `--authenticated-actor`는 단독으로는 신뢰되지 않는다. CLI는 `--approval-attestation-file`과 `--approval-attestation-hash`를 함께 넘겨야 하며, 이 attestation은 trusted wrapper/session metadata가 생성한 파일이어야 한다. attestation은 world-root-relative `runs/inbox/*-approval-attestation.json` path여야 하고 `approval attest`로 생성된 것만 허용한다. alternate staging directories, symlinks, absolute paths, 그리고 `runs/inbox/` 밖의 path는 모두 거부한다. `world-tool`은 파일 해시를 다시 계산한 뒤 attestation 내부의 `world_id`, `issuer`, `audience`, `scope_verification`, `downstream_action`, `authenticated_actor`, `approver_id`, `approval_channel`, `issued_at`, `expires_at`, `diff_run_id`, `draft_hash`, `target_base_hash`, `patch_hash`, `reason_hash`가 command/world/auth context와 command flags에 모두 일치하는지, 그리고 `downstream_action`이 실제 invoked downstream action과 정확히 일치하는지 검증한 다음에만 `approval.authenticated_actor`를 기록한다. accept 시점에는 staged attestation의 `expires_at`이 현재 시각 기준으로 아직 유효한지 먼저 검증해야 하며, 만료되었으면 `APPROVAL_ATTESTATION_EXPIRED`로 content mutation 전에 실패해야 한다. `world_accept_draft`는 `downstream_action: world_accept_draft`, `world_force_accept_draft`는 `downstream_action: world_force_accept_draft`를 요구하며, `scope_verification` allowlist에 둘 다 포함되어 있어도 exact `downstream_action`이 다르면 `APPROVAL_ATTESTATION_BINDING_MISMATCH`로 실패해야 한다. staged attestation file hash가 불일치하면 `APPROVAL_ATTESTATION_HASH_MISMATCH`, payload/command binding mismatch가 있으면 `APPROVAL_ATTESTATION_BINDING_MISMATCH`를 사용한다.
- `pass`는 explicit approval provenance가 있으면 accept 가능하다.
- `warning`은 기본 accept 가능하지만, warning과 approval provenance를 runs log에 남긴다.
- `conflict`와 `error`는 기본 accept에서 `command_status: "blocked"`로 반환한다.
- create accept는 target이 여전히 없어야 하며, accept 시점에 target이 존재하면 blocked다.
- `related target` 또는 `relationship target`이 누락되면 `MISSING_TARGET`로 blocked다. active-draft-only related/relationship target도 accept 시점에는 canon target이 없는 것으로 간주하므로 `MISSING_TARGET`를 사용한다. 즉 `MISSING_TARGET`는 related/relationship target 누락과 active-draft-only target에도 쓰인다.
- `--force`는 semantic/timeline/relationship conflict 후보만 우회할 수 있고 trusted approval attestation과 reason이 필수다. 모든 참조 대상이 이미 canon content일 때만 허용한다. reason이나 trusted attestation이 하나라도 없으면 `blocked`가 아니라 `failed`이며, 누락된 CLI 인자는 `INVALID_ARGUMENT`, auth context 문제는 `approval attest` 생성 경로에서만 대응하는 `AUTH_CONTEXT_*` 실패 코드로 처리한다. accept-time attestation hash/payload mismatch는 `APPROVAL_ATTESTATION_*` 실패 코드를 사용한다. `--approval-attestation-file`은 현재 diff/reason/actor/channel에 바인딩되고 `downstream_action: world_force_accept_draft`를 가진 fresh, non-expired attestation이어야 하며, `world_accept_draft`용 attestation은 `--force`를 허용하지 않는다.
- `--force`는 `MISSING_TARGET` 계열(blocked cases 포함 missing target, missing related target, missing relationship target, missing update/deprecate target, active-draft-only target), path/type/id/schema 불일치, structural error, id conflict, target path conflict, inactive draft, storylet canon 승격, diff binding mismatch, atomic write 실패, lock 실패는 우회할 수 없다.
- `change_type: deprecate`가 accept되면 target content를 `content/` 아래에서 in-place로 deprecated 상태와 deprecation audit metadata로 갱신하고, canon 파일은 제거하지 않는다. 해당 deprecate draft는 accepted draft와 동일하게 archive된다.
- `type: storylet`은 MVP에서 content canon accept 대상이 아니며 `STORYLET_NOT_CANON_TARGET`으로 blocked다.
- accept 성공 시 `data.approval`은 `approver_id`, `approval_channel`, `authenticated_actor`, `issuer`, `audience`, `scope_verification`, `issued_at`, `expires_at`, `attestation_validated_at`, `approval_attestation_file`, `approval_attestation_hash`, `reason_file`, `reason_hash`, `downstream_action`를 포함해야 한다. alias를 정의하지 않는 한 path suffix 이름은 쓰지 않는다.

Diff binding 정책:
- accept는 `--diff-run-id`, `--draft-hash`, `--target-base-hash`, `--patch-hash` 중 하나라도 없으면 `DIFF_BINDING_REQUIRED`로 blocked다.
- create accept는 `--target-base-hash none`을 사용해야 하며, JSON에서는 `target_base_hash: null`로 정규화한다.
- 현재 draft hash가 `--draft-hash`와 다르면 `DIFF_BINDING_MISMATCH`다.
- 현재 target content hash가 `--target-base-hash`와 다르면 `DIFF_BINDING_MISMATCH`다.
- diff artifact의 patch hash가 `--patch-hash`와 다르면 `DIFF_BINDING_MISMATCH`다.

Transaction/recovery 정책:
- accept run은 시작 시 `result.json`을 `pending`으로 기록한다.
- content write는 temp file 작성 후 atomic rename으로 수행한다.
- archive move도 같은 filesystem 안에서 atomic rename을 사용한다.
- content write 성공 후 archive/result 기록이 실패하면 `ok: false`, `command_status: "failed"`, `error.code: "TRANSACTION_INCOMPLETE"`를 반환하고 recovery instruction을 `runs/<run-id>/recovery.json`에 남긴다. 이 경우 partial mutation이 이미 발생했을 수 있다.
- 실패 응답에는 `data.recovery` metadata를 포함하고 `available_actions`는 `["world_recover_run", "world_get_run_artifact"]` 같은 복구/안전 조회 조합을 가리켜야 한다. write replay나 원래 write command 재실행은 허용하지 않는다.
- recovery는 content hash와 archive 상태를 기준으로 idempotent하게 재시도할 수 있어야 한다.
- `runs/<run-id>/recovery.json`가 unresolved인 동안에는 같은 world root를 쓰는 모든 write command와 write artifact/report command가 blocked다. 여기에는 같은 root의 `world init`, `input stage`, `approval attest`, `draft create`, `draft update`, `draft validate`의 validation artifact writer, `draft diff`, `draft accept`, `draft reject`, `content validate`의 artifact writer, `content migrate`의 report writer, 기타 content report writer가 포함된다. read-only command는 계속 허용된다.
- `run recover`만 이 차단의 예외이며, `TRANSACTION_INCOMPLETE` 또는 unresolved `recovery.json`를 복구하는 유일한 운영자용 repair command다. 이 command는 원래 write command를 다시 실행하지 않고, 기록된 transaction state만 안전하게 수습한다.

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
world-tool run get --world ashen-continent --run-id 20260530-001 --artifact manifest.json --json
world-tool run recover --world ashen-continent --run-id 20260530-001 --json
```

동작:
- 최근 실행 목록과 immutable summary만 조회
- `run get`은 기본적으로 redacted manifest와 상태 요약만 반환한다. 이때 `data`에는 최소 `manifest`와 `status_summary`가 들어가며 둘 다 redacted다.
- `run get`은 명시적인 safe artifact allowlist만 허용한다. `--artifact <artifact_name>`는 단일 값만 허용하며, 여러 artifact가 필요하면 `run get`을 반복 호출한다. basename allowlist에서만 선택할 수 있다. 예: `manifest.json`, `summary.json`, `result-summary.json`, `validation.json`, redacted `recovery.json`
- sensitive artifact, raw staged input, inbox payload, unredacted result/body/reason는 `run get`로 노출하지 않는다. 이런 자료는 별도의 privileged command나 explicit privileged flag가 있어야 한다.
- `world_get_run_artifact`는 `run get`의 explicit safe artifact retrieval 전용 mapping이다. arbitrary path는 허용하지 않는다. 이 command의 `data`는 단일 artifact object shape로 `run_id`, `artifact_name`, `artifact_hash`, `media_type`, `size_bytes`, `redacted`, `content`를 포함하고 `content_path`는 반환하지 않으며, 여러 artifact가 필요하면 반복 호출한다.
- `run recover`는 `TRANSACTION_INCOMPLETE` 또는 unresolved `recovery.json`를 해결하는 운영자용 command다. 대상 run의 `recovery.json`, 현재 content hash, archive state, result state를 다시 읽고, 이미 복구가 끝났다면 no-op으로 끝낸다.
- `run recover`는 멱등적이어야 하며, 반복 호출해도 duplicate write를 만들지 않는다. 복구가 필요한 경우에만 남은 단계를 수행하고 `recovery.json`를 resolved로 표시한다.
- `run recover`가 성공하면 이후 write command는 다시 허용된다.
- `run recover`의 결과 `data`는 최소한 `recovery_run_id`, `recovery_path`, `recovery_status`, `recovered_at`을 포함해야 하며, `available_actions`는 비어 있어도 된다.

## 17. OpenCrabs Dynamic Tool 매핑
`~/.opencrabs/tools.toml`에는 의미 단위 tool을 등록한다.

아래 예시는 canonical dynamic tool executor가 argv-safe shell executor이고, request payload를 `stdin = "{{input}}"`으로 명시적으로 바인딩한다고 가정한다. `stdin = "{{input}}"`은 supported field인 canonical pre-implementation contract로 취급한다. shell 텍스트 안에 raw interpolation만 붙이는 executor는 금지한다. 이런 조건을 만족시키지 못하는 executor에는 wrapper를 두어야 하며, wrapper는 fallback이 아니라 해당 조건을 만족시키기 위한 adapter다. `stdin = "{{input}}"`은 OpenCrabs request payload를 stdin으로 넘긴다는 뜻이며, `input`은 CLI argv가 아니라 request payload다.

```toml
[[tools]]
name = "world_stage_input"
description = "Stage long user/model input into runs/inbox and return input_path and input_hash"
executor = "shell"
command = "world-tool input stage --world {{world_id}} --kind {{kind}} --stdin --json"
stdin = "{{input}}"

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
description = "Create an update/retcon draft for an existing canon document using explicit target-id"
executor = "shell"
command = "world-tool draft create --world {{world_id}} --change-type update --target-id {{target_id}} --title-file {{title_file}} --title-hash {{title_hash}} --body-file {{body_file}} --body-hash {{body_hash}} --retcon-reason-file {{retcon_reason_file}} --retcon-reason-hash {{retcon_reason_hash}} --json"

[[tools]]
name = "world_create_deprecate_draft"
description = "Create a deprecation draft for an existing canon document using explicit target-id"
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
description = "Create trusted approval attestation from OpenCrabs session metadata, diff binding, and exact downstream action"
executor = "shell"
command = "world-tool approval attest --world {{world_id}} --diff-run-id {{diff_run_id}} --draft-hash {{draft_hash}} --target-base-hash {{target_base_hash}} --patch-hash {{patch_hash}} --approver-id {{approver_id}} --approval-channel {{approval_channel}} --downstream-action {{downstream_action}} --auth-context-file {{auth_context_file}} --auth-context-hash {{auth_context_hash}} --authenticated-actor {{authenticated_actor}} --reason-hash {{reason_hash}} --json"

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

OpenCrabs는 query/title/body/reason/retcon_reason을 command template에 직접 넣지 않는다. 먼저 `world_stage_input`으로 `runs/inbox/` path를 만들고, 그 JSON 결과의 `input_path`/`input_hash`를 kind별로 `{{kind}}_file`/`{{kind}}_hash` 변수에 재매핑한 뒤 후속 tool에는 path와 hash만 넘긴다. approval attestation은 `world_create_approval_attestation`으로 attestation path/hash를 만든 뒤 후속 tool에는 path와 hash만 넘긴다. `downstream_action`은 실제 호출 경로와 정확히 일치하는 `world_accept_draft` 또는 `world_force_accept_draft` 중 하나로만 넣는다. `world_search_docs`는 `--scope active`를 명시하고, 각 후속 command는 대응하는 `--*-hash`를 함께 전달해야 한다. `world_get_run_artifact`는 basename allowlist에 있는 `artifact_name`만 읽는다.

`world_create_draft`는 `--id`를 사용하고, `world_create_update_draft`와 `world_create_deprecate_draft`는 `--target-id`를 사용한다. update/deprecate change type에는 별도의 `--id` mapping을 두지 않는다.

모든 template 변수는 OpenCrabs가 넣더라도 신뢰하지 않는다. `world-tool`이 `world_id`, `kind`, `type`, `id`, `scope`, `target_id`, `path`, `draft_path`, `query_file`, `query_hash`, `title_file`, `title_hash`, `body_file`, `body_hash`, `reason_file`, `reason_hash`, `retcon_reason_file`, `retcon_reason_hash`, `approver_id`, `approval_channel`, `downstream_action`, `auth_context_file`, `auth_context_hash`, `approval_attestation_file`, `approval_attestation_hash`, `authenticated_actor`, `run_id`, `diff_run_id`, `draft_hash`, `target_base_hash`, `patch_hash`, `artifact_name`를 다시 검증한다.
