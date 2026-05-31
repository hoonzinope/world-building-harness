# opencrabs-integration.md

# OpenCrabs Integration

## 1. 역할 정의
OpenCrabs는 이 구조에서 하네스이자 오케스트레이터다. 사용자는 OpenCrabs와 대화하고, OpenCrabs는 Codex OAuth provider를 통해 판단/생성을 수행하며, 세계관 파일 작업은 dynamic tools가 호출하는 `world-tool`이 수행한다. Codex CLI provider는 fallback으로만 사용한다.

이 레포는 OpenCrabs를 확장하기 위한 다음 자산을 정의하는 설계 계약이다.

- `opencrabs/skills/world-building/SKILL.md`
- `opencrabs/tools/world-tools.toml`
- `world-tool` Go 바이너리
- schema와 validation 규칙

## 2. OpenCrabs가 책임지는 것
- 사용자 대화와 승인 확인
- Codex OAuth provider 기본 사용
- Codex CLI provider fallback
- world-building skill 실행
- dynamic tool 호출
- tool 결과를 바탕으로 다음 행동 판단
- Telegram/Discord/Slack 등 채널 응답

## 3. world-tool이 책임지는 것
- world root path boundary 검사
- content/drafts/archive/runs 파일 조작
- frontmatter 정규화
- validation
- diff 생성
- accept/reject 정책 강제
- JSON 결과 반환

## 4. Skill 설치
OpenCrabs user skill은 다음 위치에 둔다.

```text
~/.opencrabs/skills/world-building/SKILL.md
```

레포 내부 소스 위치:

```text
opencrabs/skills/world-building/SKILL.md
```

skill은 다음을 지시한다.

- 세계관 작업에는 `world_*` tools를 사용한다.
- `content/` 직접 수정은 하지 않는다.
- draft 생성 후 validation을 수행한다.
- accept 전에는 사용자 승인을 확인한다.
- tool output을 authoritative state로 취급한다.

## 5. Dynamic Tools 설치
OpenCrabs dynamic tools는 다음 위치에 정의한다.

```text
~/.opencrabs/tools.toml
```

레포 내부 소스 위치:

```text
opencrabs/tools/world-tools.toml
```

canonical dynamic tool:

아래 예시는 표준 command shape를 보여주는 pre-implementation contract다. 현재 OpenCrabs TOML과 shell executor는 `executor = "shell"`, argv-safe template escaping, `stdin = "{{input}}"` payload binding을 지원한다고 가정한다. 이 필드를 지원하지 않는 runtime은 표준 adapter wrapper를 사용해야 하며, wrapper는 JSON request를 받아 argv/stdin/auth-context를 안전하게 주입해야 한다. auth metadata는 wrapper/session metadata에서만 와야 하고 prompt나 staged input에서 오면 안 된다. `world_id`, `world_create_approval_attestation`과 그 attestation이 뒷받침할 정확한 downstream mutation action(`world_accept_draft` 또는 `world_force_accept_draft`), issuer/audience/scope, actor/channel/expiry, hash 검증은 trusted auth context input이 제공하고 검증해야 하며, attestation 생성만 허용하는 scope는 content mutation에 충분하지 않다. production auth context input은 wrapper-signed 또는 MACed envelope여야 하고, configured wrapper trust material으로 검증되며 expected issuer/audience/scope policy를 만족해야 한다. local fixture mode는 test-only이고 explicit opt-in이 필요하다.

```toml
[[tools]]
name = "world_stage_input"
description = "Stage long user/model input into runs/inbox and return input_path and input_hash"
executor = "shell"
command = "world-tool input stage --world {{world_id}} --kind {{kind}} --stdin --json"
stdin = "{{input}}"

[[tools]]
name = "world_list"
description = "List canonical worlds"
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
description = "Create a draft without modifying canon content"
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
description = "Create approval attestation from trusted auth context input, diff binding, and exact downstream action binding"
executor = "shell"
command = "world-tool approval attest --world {{world_id}} --downstream-action {{downstream_action}} --diff-run-id {{diff_run_id}} --draft-hash {{draft_hash}} --target-base-hash {{target_base_hash}} --patch-hash {{patch_hash}} --approver-id {{approver_id}} --approval-channel {{approval_channel}} --authenticated-actor {{authenticated_actor}} --auth-context-file {{auth_context_file}} --auth-context-hash {{auth_context_hash}} --reason-hash {{reason_hash}} --json"

[[tools]]
name = "world_accept_draft"
description = "Promote a validated draft into canon after explicit user approval and trusted approval attestation"
executor = "shell"
command = "world-tool draft accept --world {{world_id}} --draft {{draft_path}} --diff-run-id {{diff_run_id}} --draft-hash {{draft_hash}} --target-base-hash {{target_base_hash}} --patch-hash {{patch_hash}} --approver-id {{approver_id}} --approval-channel {{approval_channel}} --approval-attestation-file {{approval_attestation_file}} --approval-attestation-hash {{approval_attestation_hash}} --authenticated-actor {{authenticated_actor}} --reason-file {{reason_file}} --reason-hash {{reason_hash}} --json"

[[tools]]
name = "world_force_accept_draft"
description = "Force promote a draft only on an operator-approved path with trusted approval attestation"
executor = "shell"
command = "world-tool draft accept --world {{world_id}} --draft {{draft_path}} --diff-run-id {{diff_run_id}} --draft-hash {{draft_hash}} --target-base-hash {{target_base_hash}} --patch-hash {{patch_hash}} --force --approver-id {{approver_id}} --approval-channel {{approval_channel}} --approval-attestation-file {{approval_attestation_file}} --approval-attestation-hash {{approval_attestation_hash}} --authenticated-actor {{authenticated_actor}} --reason-file {{reason_file}} --reason-hash {{reason_hash}} --json"

[[tools]]
name = "world_reject_draft"
description = "Archive a draft as rejected with a reason"
executor = "shell"
command = "world-tool draft reject --world {{world_id}} --draft {{draft_path}} --reason-file {{reason_file}} --reason-hash {{reason_hash}} --json"

[[tools]]
name = "world_recover_run"
description = "Resolve a partially committed transaction using run recovery state"
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

`world_get_run`은 redacted manifest/status summary만 반환한다. 안전한 artifact가 필요할 때만 `world_get_run_artifact`를 호출하고, allowlist에 있는 basename만 읽는다. `runs/inbox/**`는 노출하지 않는다.

`world_create_approval_attestation`이 staged approval attestation에 기록한 `downstream_action`은 `world_accept_draft` 또는 `world_force_accept_draft`와 exact match여야 한다.

긴 markdown body, 검색 query, title, reason, retcon_reason은 command template에 직접 넣지 않는다. 먼저 `world_stage_input`으로 world root 내부 `runs/inbox/`에 staging한다. `world_stage_input`은 `input_path`와 `input_hash`만 반환하고, OpenCrabs가 kind별로 이를 `query_file/query_hash`, `title_file/title_hash`, `body_file/body_hash`, `reason_file/reason_hash`, `retcon_reason_file/retcon_reason_hash`로 다시 매핑한다. 승인 provenance는 `world_create_approval_attestation`으로 별도 approval attestation을 staging하고, trusted auth context input은 world root 밖에서 생성한 `auth_context_file`/`auth_context_hash`로 전달한다. 이 input은 `world_id`, `world_create_approval_attestation`과 exact downstream mutation action(`world_accept_draft` 또는 `world_force_accept_draft`), issuer/audience, actor/channel/expiry, hash 검증을 함께 제공해야 한다. attestation 생성 scope만 있고 downstream mutation scope가 없으면 `AUTH_CONTEXT_SCOPE_DENIED`가 되어야 하며, 후속 tool에는 이러한 file/hash binding과 staged approval attestation binding만 넘긴다.

template 변수는 OpenCrabs가 넣더라도 신뢰하지 않는다. `world-tool`은 `world_id`, `kind`, `type`, `id`, `scope`, `target_id`, `path`, `draft_path`, `query_file`, `query_hash`, `title_file`, `title_hash`, `body_file`, `body_hash`, `reason_file`, `reason_hash`, `retcon_reason_file`, `retcon_reason_hash`, `approver_id`, `approval_channel`, `approval_attestation_file`, `approval_attestation_hash`, `authenticated_actor`, `auth_context_file`, `auth_context_hash`, `downstream_action`, `run_id`, `diff_run_id`, `draft_hash`, `target_base_hash`, `patch_hash`, `artifact_name`를 다시 검증한다.

## 6. World Registry
world id는 registry 또는 harness provenance를 통해 root로 resolve한다. id 문자열 자체를 path로 취급하지 않는다.

예시:

```yaml
worlds:
  ashen-continent:
    title: 잿빛 대륙
    root: /host/worlds/ashen-continent
  glass-sea:
    title: 유리해
    root: /host/worlds/glass-sea
```

registry는 OpenCrabs 설정, 별도 world registry 파일, 또는 `world-tool` config로 관리할 수 있다. canon source of truth는 registry가 아니라 각 world root의 `content/`다.

권장 registry 위치:

```text
~/.opencrabs/worlds.yaml
```

대체 위치:

```text
~/.config/world-tool/worlds.yaml
```

해석 우선순위:
1. command flag `--registry <path>`가 있으면 해당 파일
2. 환경변수 `WORLD_TOOL_REGISTRY`
3. `~/.opencrabs/worlds.yaml`
4. `~/.config/world-tool/worlds.yaml`

registry path 자체는 world root 안에 둘 필요가 없다. 단, registry가 가리키는 root는 symlink 해석 후 absolute path로 고정하고, 이후 모든 파일 접근은 그 root 내부로 제한한다.

`registry add`는 `--world`, `--root`, `--title`을 모두 요구하는 registry-only mutation이다. 여기서 `null-root`는 world root를 열지 않는다는 뜻이며, `registry_root`, `root`, `run_id`가 null이라는 의미다. `registry add`의 `world_id`는 selected/target id를 담는다. `registry remove`와 `registry default`도 `--world`가 필수인 null-root command이며, 이들 역시 `registry_root`, `root`, `run_id`는 null이고 `world_id`에는 target id를 담는다. `registry list`와 `world list`는 `--world`/`--root`를 금지하는 조회 명령이고, 이 둘만 `world_id`, `registry_root`, `root`, `run_id`가 모두 null이다. `world list`는 canonical world 목록 조회이고, `registry list`는 registry 파일 자체를 점검하는 운영/관리 alias다.

canonical root binding:

| Concept | Meaning |
| --- | --- |
| `world_id` | registry가 식별하는 logical world key. `registry add/remove/default`에서는 selected/target id를 담고, `registry list`와 `world list`에서는 null이다. Docker `--root` mode에서는 mount path로 추측하지 않고 `--world-id` 또는 `harness.yaml` provenance가 있을 때만 resolve한다. |
| `registry_root` | registry 또는 provenance가 제공하는 host canonical world root. provenance가 없으면 null이다. |
| `root` | 실제 tool process가 접근하는 effective root |
| audit fields | command별로 `world_id`, `registry_root`, `root`, `run_id`의 null 여부를 구분해 기록한다. |

native execution에서는 `registry_root == root`여야 한다. Docker에서는 registry_root와 effective root인 `root`가 달라질 수 있으므로, registry resolution과 audit logging이 둘을 구분해 기록해야 한다.

Docker `--root` mode에서 registry/provenance가 host canonical root를 제공하면 `registry_root`에는 host canonical root를, `root`에는 container 안의 effective root를 기록한다. `--root`만 있고 host canonical root provenance가 없으면 host path를 추측하지 말고 `registry_root: null`, `root: <effective root>`로 기록하거나, root-only mode를 provenance 부족으로 제한해야 한다. 이때 `world_id`도 `--world-id` 또는 `harness.yaml` provenance 없이는 mount path에서 추측하지 않는다.

world id 규칙:
- 영문 소문자, 숫자, 하이픈만 허용한다.
- path separator, whitespace, shell metacharacter는 금지한다.
- registry 안에서 중복 id는 config error다.

등록/선택 흐름:
1. `world-tool world init --root /host/worlds/ashen-continent --world-id ashen-continent --json`
2. `world-tool registry add --world ashen-continent --root /host/worlds/ashen-continent --title "잿빛 대륙" --json`
3. 필요하면 `world-tool registry list --json`로 registry 파일 자체를 점검한다. canonical 목록 조회는 `world-tool world list --json`이다.
4. 필요하면 `world-tool registry default --world ashen-continent --json`

Docker에서 registry가 host path를 가리키고 tool container가 `/workspace/world`로 mount하는 경우, registry root와 effective in-container root가 달라질 수 있다. 이 운영 방식은 wrapper가 in-container registry/provenance를 `/workspace/world`에 매핑하는 경우에만 허용된다. registry를 컨테이너 안에서 읽을 수 있으면 canonical command는 `world-tool <command> --world ashen-continent ...`처럼 logical id를 직접 쓰고, registry를 읽을 수 없으면 wrapper가 host canonical root provenance를 확보한 뒤 `world-tool draft read --root /workspace/world --world-id ashen-continent --draft drafts/nations/nation_northern_empire.md --json` 같은 command-shaped form으로 호출해야 한다. 이때 audit/result envelope에는 `registry_root: /host/worlds/ashen-continent`와 `root: /workspace/world`를 둘 다 남겨야 하며, provenance가 없으면 root-only 모드를 쓰지 않는다.

## 7. 대화 플로우
```text
사용자: /world-building 북부 제국 설정 만들어줘
OpenCrabs: skill rules 적용
OpenCrabs: 검색 query를 world_stage_input으로 staging
OpenCrabs/Codex: 관련 canon 조회를 위해 world_search_docs(scope=active) 호출
OpenCrabs/Codex: draft markdown 생성
OpenCrabs: title를 world_stage_input으로 staging
OpenCrabs: body를 world_stage_input으로 staging
OpenCrabs: create는 명시적 `--id`를 정한 뒤 world_create_draft 호출 -> draft_path 수신; update/deprecate는 명시적 `--target-id`를 사용한다
OpenCrabs: world_validate_draft 호출
OpenCrabs: id, draft path, validation status, 다음 행동 제안
사용자: 승인해
OpenCrabs: world_diff_draft 출력은 create에서 `target_exists=false`, `target_base_hash=null`로 나오며, OpenCrabs dynamic tool layer가 approval attest/accept CLI 호출 직전에 이 null을 CLI template 변수 `target_base_hash="none"`으로 매핑하고, update/deprecate 경로의 sha256 `target_base_hash`는 사용자 확인에 묶음
OpenCrabs: reason을 world_stage_input으로 staging
OpenCrabs: world_create_approval_attestation 호출(trusted auth context input, exact downstream_action binding, diff/reason hash binding, auth_context_file/hash)
OpenCrabs: world_accept_draft 호출(create 경로는 target_base_hash=none, update/deprecate 경로는 sha256 target_base_hash; reason_file, reason_hash, approval_attestation_file, approval_attestation_hash, approver_id, approval_channel=OpenCrabs-chat, authenticated_actor 포함)
world-tool: validation 재실행 후 content 승격
```

## 8. Docker 운영
권장 운영 방식은 OpenCrabs와 `world-tool`을 같은 이미지에 넣되, security boundary를 만족하는 hardened 실행 옵션을 기본으로 두는 것이다. OpenCrabs credential/config volume과 world root volume을 분리해 마운트하고, Codex OAuth provider 호출을 위해 제한된 egress network를 붙인다. provider egress allowlist는 OpenCrabs 설정의 책임이며 `world-tool`은 arbitrary external fetch를 하지 않는다. Codex CLI provider를 쓰지 않는다면 컨테이너 안에 Codex CLI를 설치하거나 `~/.codex`를 마운트하는 것은 기본 요구사항이 아니다.

```bash
docker run --rm \
  --user 1000:1000 \
  --network opencrabs-min-egress \
  --read-only \
  --tmpfs /tmp \
  --tmpfs /run \
  --cap-drop ALL \
  --security-opt no-new-privileges \
  -v opencrabs-config:/home/opencrabs/.opencrabs \
  -v /host/worlds/ashen-continent:/workspace/world \
  opencrabs-world:latest
```

이 권장 예시는 accepted flow에서 `runs/`, `drafts/`, `archive/`, `content/`에 write해야 하므로 world root bind mount는 writable이어야 한다. `--read-only`는 root filesystem hardening으로 유지하되, world root 자체를 read-only로 두지 않는다.

같은 이미지로 `world-tool`을 실행하는 경우에도 `/workspace/world` 마운트 경로만으로 logical `world_id`를 추측하지 않는다. wrapper가 컨테이너 안 registry/provenance를 읽을 수 있으면 `world-tool <command> --world ashen-continent ...`처럼 logical id를 직접 넘기고, 그렇지 않으면 host canonical root provenance를 먼저 확보한 뒤 `world-tool draft read --root /workspace/world --world-id ashen-continent --draft drafts/nations/nation_northern_empire.md --json` 같은 command-shaped form으로 호출하며, audit envelope에 host canonical `registry_root`와 container `root`를 모두 남긴다.

강한 격리가 필요하면 OpenCrabs 자체는 host에서 실행하되, dynamic tool command가 per-world container에서 `world-tool`을 실행하도록 구성할 수 있다.

```bash
docker run --rm \
  --user 1000:1000 \
  --network none \
  --read-only \
  --tmpfs /tmp \
  --tmpfs /run \
  --cap-drop ALL \
  --security-opt no-new-privileges \
  -v /host/worlds/ashen-continent:/workspace/world:ro \
  world-tool:latest \
  world-tool draft read --root /workspace/world --world-id ashen-continent --draft drafts/nations/nation_northern_empire.md --json
```

이 예시는 read-only inspection 전용 world root 예시다. `draft validate`는 계약상 validation artifact를 쓸 수 있으므로 `:ro` world mount에 두지 않는다.

개발 단계에서는 host에 설치된 `world-tool`을 직접 호출해도 된다.

Codex CLI provider fallback을 사용할 때만 컨테이너에 `codex` CLI와 별도 Codex auth volume이 필요하다. 이 경우에도 Codex auth volume은 `/workspace/world` 안에 두지 않는다.

## 9. 실패 처리
### malformed JSON
OpenCrabs는 raw stdout을 사용자에게 그대로 보여주지 않고, tool 실패와 stderr 요약을 제공한다.

stdout에 JSON이 있으면 `schema_version`을 확인한 뒤 `ok` → `command_status` → `data.validation_status` → `data.block_reason` → `issues` → `available_actions` → `error.code` 순서로 해석한다. stdout이 비어 있거나 JSON parse가 실패하면 dynamic tool 자체 실패로 처리한다.

`world_stage_input`은 `input_path`와 `input_hash`를 반환한다. 이후 tool 호출은 그 파일 경로와 해시를 그대로 넘기고, `world-tool`이 다시 계산한 해시와 비교해야 한다. `authenticated_actor`는 OpenCrabs의 인증된 세션 또는 provider identity에서 가져와야 하며, prompt나 staging file에서 받아서는 안 된다. `world_create_approval_attestation`은 trusted auth context input이 없으면 `AUTH_CONTEXT_MISSING`으로 실패한다. 승인 provenance는 `world_id`, `world_create_approval_attestation` scope와 exact downstream mutation scope(`world_accept_draft` 또는 `world_force_accept_draft`), issuer/audience/scope policy, `reason_file`/`reason_hash`, `approval_attestation_file`/`approval_attestation_hash`, `approver_id`, `approval_channel`, `authenticated_actor`, expiry, hash 및 trust-material 검증이 모두 맞아야 유효하다. `world_create_approval_attestation`은 `auth_context_file`/`auth_context_hash`를 world root 밖에서 생성된 입력으로 받아야 하고, create 경로의 `diff_run_id`/`draft_hash`/`target_base_hash`/`patch_hash`는 `world_diff_draft`의 출력에서 오되 `target_base_hash=null`을 approval attest/accept 직전에 CLI template 변수 `target_base_hash="none"`으로 정규화해 bind해야 하며, create 경로의 `reason_hash`는 `world_stage_input`의 출력으로만 bind해야 하고, `reason_file`은 `world_accept_draft`/`world_force_accept_draft`가 소비해야 한다. update/deprecate 경로는 sha256 `target_base_hash`를 사용한다.
OpenCrabs는 blocked 결과를 읽을 때 `ok` → `command_status` → `data.validation_status` → `data.block_reason` → `issues` → `available_actions` → `error.code` 순서로 해석한다. `data.block_reason`이 있으면 먼저 domain blocked 사유로 다루고, 그 다음 `issues`로 세부 원인을 읽고, `available_actions`로 다음 행동을 고른다. `TRANSACTION_INCOMPLETE`는 failed/recovery-required partial transaction으로 다뤄야 하며, `PATH_*`, `LOCK_BUSY`, I/O/path/lock 문제는 failed JSON error로 분리해야 한다. 승인 provenance는 `world_id`, `world_create_approval_attestation` scope와 exact downstream mutation scope(`world_accept_draft` 또는 `world_force_accept_draft`), issuer/audience/scope policy, `reason_file`/`reason_hash`, `approval_attestation_file`/`approval_attestation_hash`, `approver_id`, `approval_channel`, `authenticated_actor`, expiry, hash 및 trust-material 검증이 모두 맞아야 유효하며, attestation 내부 actor/channel은 OpenCrabs trusted auth context input과 일치해야 한다. `approval_channel` 예시는 `OpenCrabs-chat`을 사용한다.

### validation conflict
OpenCrabs는 accept를 강행하지 않고 blocked 이유와 수정안을 사용자에게 보여준다.

사용자가 강행을 요청하면 OpenCrabs는 `world_force_accept_draft`를 사용한다. 이 경로도 `approval_attestation_file`, `approval_attestation_hash`, `approver_id`, `approval_channel`, `authenticated_actor`를 요구하며, `approval_channel`은 attest/accept/audit 전반에서 byte-identical이어야 한다. tool이 `FORCE_NOT_ALLOWED` 또는 `VALIDATION_BLOCKED`를 반환하면 강행하지 않고 blocked 이유를 설명한다. attestation 생성만 허용하는 auth context input은 content mutation을 정당화하지 못하며, mutation scope가 없으면 `AUTH_CONTEXT_SCOPE_DENIED`로 실패해야 한다. staged approval attestation은 `runs/inbox/*-approval-attestation.json`에 기록한다.

### path violation
`world-tool`은 world root 밖 경로 접근을 error로 반환한다. 단, `approval attest`의 `auth_context_file`은 world root 밖 trusted auth context input을 read-only로 읽는 예외다. production에서는 wrapper-signed 또는 MACed envelope를 configured wrapper trust material으로 검증하고 expected issuer/audience/scope policy를 만족해야 하며, local fixture mode는 test-only이고 explicit opt-in이 필요하다. hash/expiry만으로는 충분하지 않다.

## 10. 설계 원칙
- OpenCrabs가 하네스다.
- 이 레포는 OpenCrabs skill/tools bundle이다.
- `world-tool`은 Go 단일 바이너리다.
- OpenCrabs는 판단을 하고, `world-tool`은 파일 상태 변경을 강제한다.
- Codex SDK는 직접 내장하지 않는다.
