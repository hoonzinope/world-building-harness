# implementation-plan.md

# Implementation Plan

## 1. 목적
이 문서는 문서화된 설계를 실제 구현으로 옮기기 위한 이정표다. `roadmap.md`가 큰 phase와 방향을 다룬다면, 이 문서는 구현 순서, 산출물, 완료 기준을 체크리스트로 제공한다.

기본 전제:

- OpenCrabs가 하네스/오케스트레이터다.
- 이 레포는 OpenCrabs world-building skill/tools bundle과 `world-tool` Go CLI를 제공한다.
- `content/` Markdown이 canon source of truth다.
- OpenCrabs/Codex는 판단과 생성을 담당하고, 파일 상태 변경은 `world-tool`이 강제한다.
- Codex OAuth provider가 기본 provider이며, Codex CLI provider는 fallback이다.

## 2. 구현 원칙
- 먼저 OpenCrabs 없이 동작하는 `world-tool` vertical slice를 만든다.
- 모든 파일 접근은 world root boundary 안에서만 허용한다.
- 모든 write command는 `runs/`에 audit artifact를 남긴다.
- 모든 command는 `--json` 모드에서 [commands.md](commands.md)의 JSON envelope만 stdout에 반환한다.
- 긴 markdown body, 검색 query, title, reason, 사용자 입력은 argv가 아니라 stdin 또는 world root 내부 `runs/inbox/` staging file로 받는다.
- `content/`는 `draft accept` 외의 command에서 수정하지 않는다.
- skill은 지침이고, 안전 경계는 `world-tool`에서 강제한다.
- write command는 world root lock을 사용하고, accept는 lock 안에서 validation을 재실행한다.
- diff와 accept는 `diff_run_id`, `draft_hash`, `target_base_hash`, `patch_hash`로 묶는다.
- unresolved recovery가 있으면 같은 world root의 `world init`, `input stage`, `approval attest`, `draft create`, `draft update`, `draft validate`(validation artifact writer), `draft diff`, `draft accept`, `draft reject`, `content validate` artifact writer, `content migrate` report writer, 기타 content report writer는 차단되고 `world_recover_run`만 write 예외다. read-only inspection은 허용한다.

## 3. Milestone 0: Repository Scaffold
### 목표
Go CLI와 OpenCrabs bundle을 구현할 기본 디렉토리를 만든다.

### 산출물
- `go.mod`
- `cmd/world-tool/main.go`
- `internal/world`
- `internal/docs`
- `internal/drafts`
- `internal/validate`
- `internal/diff`
- `internal/audit`
- `internal/config`
- `opencrabs/skills/world-building/SKILL.md`
- `opencrabs/tools/world-tools.toml`
- `schema/world-doc.schema.json`
- `schema/relationship-types.yaml`
- `schema/document-types.yaml`
- `examples/worlds/ashen-continent`

### 완료 기준
- `go test ./...`가 실행된다.
- `world-tool --help` 또는 동등한 기본 command가 실행된다.
- 아직 OpenCrabs 없이도 local CLI 개발을 시작할 수 있다.

## 4. Milestone 1: World Root Boundary
### 목표
모든 후속 기능의 안전 기반이 되는 world root 접근 계층을 구현한다.

### 산출물
- `--root <path>` 기반 world root 로딩
- `harness.yaml` 기본 설정 로딩
- path normalization
- symlink escape 차단
- world root document/artifact 밖 read/write 차단. 단, registry/config 파일은 world root 밖에 둘 수 있으므로 registry path는 별도 path validation을 거치고 `registry add/list/remove/default`는 null-root registry file 설정에서도 동작해야 하며, `approval attest`의 trusted wrapper auth_context_file/auth_context_hash는 production에서는 signature/MAC 또는 configured wrapper trust-material 검증과 expected issuer/audience/scope policy를 통과한 경우에만 read-only 예외로 허용하고, `auth_context_hash`는 integrity binding만 담당한다
- `runs/inbox/` staging path 검증
- world root lock helper
- atomic write helper
- `world-tool world init`
- `world-tool world status`
- `world-tool registry add/list/remove/default`
- `world-tool input stage`

### 완료 기준
- `world init`이 `content/`, `drafts/`, `runs/`, `archive/`, `graph/`, `harness.yaml`을 생성한다.
- `../`, absolute path, symlink를 통한 root 밖 접근이 차단된다. 단, registry/config 파일은 path validation을 거친 null-root registry 설정에서 예외적으로 허용되며, `--auth-context-file`은 `approval attest`에서만 production trusted auth context input으로서 signature/MAC 또는 configured wrapper trust-material 검증과 expected issuer/audience/scope policy, 그리고 `--auth-context-hash` integrity binding을 함께 검증한 read-only 예외다. local fixture mode는 `WORLD_TOOL_TEST_AUTH_CONTEXT=1` explicit opt-in 테스트 전용 경로다.
- `--query-file`, `--title-file`, `--body-file`, `--reason-file`, `--retcon-reason-file`도 `runs/inbox/` 아래 상대 경로만 허용된다.
- path violation은 JSON error와 non-zero exit code를 반환한다.
- `input stage`와 `approval attest`만 `runs/inbox/`에 staging file을 생성할 수 있다.
- `registry add/list/remove/default`는 null-root registry file 설정과 path validation을 포함한 boundary 규칙을 통과해야 한다.

## 5. Milestone 2: Content Read Model
### 목표
OpenCrabs가 기존 canon을 안전하게 조회할 수 있게 한다.

### 산출물
- markdown 파일 목록화
- YAML frontmatter 파싱
- document id/type/title/status 추출
- `world-tool doc list`
- `world-tool doc read`
- `world-tool doc search`

### 완료 기준
- `content/`와 `drafts/` scope를 구분해서 조회할 수 있다.
- archive는 기본 조회와 active validation scope에서 제외된다.
- `doc read`는 world root 내부의 markdown만 읽는다.
- `doc read/search/list`는 `content/`와 `drafts/`만 대상으로 하며 `runs/`, `archive/`, `raw/`, `schema/`를 읽지 않는다.
- `doc search`는 최소한 title, id, tag, body text 기반 검색을 지원한다.

## 6. Milestone 3: Draft Lifecycle
### 목표
LLM 생성물을 canon에 바로 쓰지 않고 draft로 저장하는 경로를 구현한다.

### 산출물
- `world-tool draft create`
- `world-tool draft update`
- `world-tool draft read`
- `world-tool draft list`
- `world-tool draft reject`
- draft frontmatter 보정
- draft id/path 생성 규칙
- draft create는 explicit `--id` 또는 동등한 id template variable을 필수로 받아야 하고 title에서 id를 암묵적으로 파생하지 않는다.
- `change_type: create/update/deprecate`
- `target_id` 기반 retcon/update/deprecate draft
- `source_run_id` 기록

### 완료 기준
- draft 생성/수정 시 `content/`는 변경되지 않는다.
- markdown body는 `--body-file` 또는 stdin으로 받을 수 있다.
- draft 생성 결과는 JSON으로 draft path, id, run id를 반환한다.
- reject는 draft를 `archive/rejected/`로 이동하고 reason을 audit에 남긴다.

## 7. Milestone 4: Validation MVP
### 목표
draft를 canon과 비교해 구조 오류와 명백한 충돌을 탐지한다.

### 산출물
- `world-tool draft validate`
- `world-tool content validate`
- validation report JSON
- validation artifact 저장
- required field rule
- invalid change_type rule
- draft path/type rule
- change_type/target_id/retcon_reason rule
- id uniqueness rule
- create with target_id/retcon_reason rule
- missing target_id rule
- missing update/deprecate target rule
- target id mismatch rule
- relationship target existence rule
- relationship normalization, dedupe, and consistency rule
- inverse/symmetric contradiction rule
- timeline/event consistency rule
- target path conflict rule
- target content base hash 계산

### 완료 기준
- validation status는 `pass`, `warning`, `conflict`, `error` 중 하나다.
- 새 draft의 schema_version 누락과 parse 실패는 `error`로 반환된다.
- legacy/import content의 schema_version 누락은 migration warning으로 반환된다.
- draft의 `change_type` 누락, invalid enum, `update/deprecate`의 `target_id` 또는 `retcon_reason` 누락은 `error`로 반환된다.
- `change_type: create`에 `target_id` 또는 `retcon_reason`이 들어가면 `error`로 반환된다.
- `change_type: update` 또는 `change_type: deprecate`에서 target_id가 없거나 id가 target_id와 다르면 `error`로 반환된다.
- active draft path/type 규칙 위반은 `error`로 반환된다.
- `change_type: create`에서 기존 canon id와 중복되는 draft는 `conflict`로 반환된다.
- `draft create --change-type update|deprecate`에서 target_id가 canon content에 없으면 no-write blocked `MISSING_TARGET`로 종료한다.
- 이미 존재하는 invalid draft를 `draft validate`하면 `command_status: "completed"`, `data.validation_status: "conflict"`, `issues[].code: "MISSING_TARGET"`로 보고하고 `data.block_reason`은 쓰지 않는다.
- convenience field는 explicit relationships[]가 없어도 graph fact로 normalize된다.
- convenience field와 explicit relationships[]가 같은 fact로 normalize되면 dedupe된다.
- convenience field와 explicit relationships[]가 서로 다른 fact를 만들면 `conflict`로 반환된다.
- 잘못된 shape나 비문자열 id는 `error`로 반환된다.
- inverse/symmetric contradiction은 `conflict`로 반환된다.
- accept validation에서 active draft에만 존재하는 relationship target은 `conflict`로 반환되고, accept/diff validation에서 active draft에만 존재하는 relationship target은 `command_status: "blocked"`, `data.block_reason: "MISSING_TARGET"`, `data.validation_status: "conflict"`로 처리된다.
- 알 수 없는 relationship type과 domain/range mismatch는 `conflict`로 반환된다.
- related id와 relationship target이 active draft에만 존재하는 경우는 draft validate에서 warning, accept/diff에서는 `command_status: "blocked"`, `data.block_reason: "MISSING_TARGET"`, `data.validation_status: "conflict"`로 처리한다.
- unresolved recovery가 있으면 `draft validate`처럼 `runs/<run-id>/validation.json`을 쓰는 run-writing command는 blocked된다.
- validation 결과는 `runs/<run-id>/validation.json`에 저장된다.

## 8. Milestone 5: Diff, Accept, Audit
### 목표
사용자 승인을 받은 draft만 canon으로 승격하고, 재현 가능한 audit trail을 남긴다.

### 산출물
- `world-tool draft diff`
- `world-tool draft accept`
- `world-tool approval attest`
- `world-tool run recover`
- `world-tool run get`
- `world-tool run get --artifact <basename>`
- accept 직전 validation 재실행
- `--force`, `--reason-file`, `--reason-hash`, `--approval-attestation-file`, `--approval-attestation-hash`, `--approver-id`, `--approval-channel`, `--authenticated-actor`
- diff binding flags: `--diff-run-id`, `--draft-hash`, `--target-base-hash`, `--patch-hash`
- conflict/error 기본 차단
- content atomic write
- world root lock
- diff base hash와 accept 시점 hash 비교
- transaction/recovery artifact
- accepted draft archive 이동
- `runs/<run-id>/diff.patch`
- `runs/<run-id>/result.json`
- `runs/<run-id>/events.jsonl`
- `runs/<run-id>/recovery.json`
- `runs/<run-id>/run.json`

### 완료 기준
- accept는 validation을 다시 실행한다.
- `run recover`는 unresolved recovery를 idempotently resolve하고 `recovery.json`을 resolved로 기록한다. 이후부터는 동일 world root의 write command가 다시 허용되지만, recovery path는 원래 `draft accept`를 재생하는 것이 아니다.
- `run get`은 recovery inspection에 필요한 최소 run metadata와 artifact 참조를 조회할 수 있다.
- `run get --artifact <basename>`은 basename allowlist를 통과한 safe artifact만 제공하고, `runs/inbox/**`는 제외하며, redacted/safe artifact만 노출한다.
- `run get`은 redacted manifest/status를 반환하고, `run get --artifact`는 redacted recovery metadata 같은 allowlisted safe artifact만 제공하며 inbox/sensitive artifact는 거부하고, `run recover` 전에 unresolved recovery를 inspection할 수 있어야 한다.
- accept는 diff binding이 없거나 불일치하면 blocked다.
- conflict/error가 있으면 기본 accept가 blocked다.
- force accept는 missing target, missing related target, missing relationship target, missing update/deprecate target, active draft-only target, path/type/id/schema 불일치, structural error, id conflict, target path conflict, diff binding mismatch, storylet canon 승격, atomic write 실패, lock 실패는 우회할 수 없다. reason 누락은 `INVALID_ARGUMENT` failed로, auth context/attestation provenance 문제는 대응하는 `AUTH_CONTEXT_*` 또는 attestation/hash mismatch failed로 처리한다.
- force accept는 semantic/timeline/relationship conflict 후보 중 referenced target이 모두 canon content에 있는 경우에만 제한적으로 우회할 수 있다.
- `approval attest`는 `WORLD_TOOL_TEST_AUTH_CONTEXT=1` explicit opt-in test fixture 경로와 trusted wrapper boundary를 연결하며, production auth context input은 signature/MAC 또는 configured wrapper trust-material 검증과 expected issuer/audience/scope policy를 통과해야 하고, `auth_context_hash`는 integrity binding으로만 사용된다.
- accept 성공 시 content 문서가 생성 또는 갱신된다.
- accept 성공 시 draft 원본은 `archive/accepted/`로 이동한다.
- 모든 변경은 runs artifact로 추적 가능하다.

## 9. Milestone 6: OpenCrabs Skill
### 목표
OpenCrabs 대화에서 세계관 작업 규칙을 안정적으로 주입한다.

### 산출물
- `opencrabs/skills/world-building/SKILL.md`
- draft-first workflow 지침
- content 직접 수정 금지 지침
- validation 후 사용자 승인 지침
- warning/conflict/error 응답 지침
- tool output을 authoritative state로 취급하는 지침

### 완료 기준
- skill만 읽어도 OpenCrabs가 어떤 tool을 어떤 순서로 호출해야 하는지 알 수 있다.
- 사용자가 content 직접 수정을 요청해도 draft workflow로 유도한다.
- 사용자가 warning 무시 또는 accept 강행을 요청할 때 reason/audit/force 정책을 설명한다.

## 10. Milestone 7: OpenCrabs Dynamic Tools
### 목표
OpenCrabs가 `world-tool`을 범용 shell이 아니라 의미 단위 tool로 호출하게 한다.

### 산출물
- `opencrabs/tools/world-tools.toml`
- approval attestation trusted wrapper/adapter contract
- `auth_context_file` creation/injection, hashing, and cleanup contract; staging is reserved for `runs/inbox/*-approval-attestation.json`
- `auth_context_hash` derivation and verification contract
- auth context expiry/scope/session metadata propagation contract
- `world_stage_input`
- `world_list`
- `world_status`
- `world_search_docs`
- `world_read_doc`
- `world_create_draft`
- `world_create_update_draft`
- `world_create_deprecate_draft`
- `world_update_draft`
- `world_read_draft`
- `world_validate_draft`
- `world_diff_draft`
- `world_create_approval_attestation`
- `world_accept_draft`
- `world_force_accept_draft`
- `world_reject_draft`
- `world_recover_run`
- `world_get_run`
- `world_get_run_artifact`

### 완료 기준
- 각 tool은 stdout JSON만 반환한다.
- 긴 query/title/body/reason/retcon_reason은 `runs/inbox/` staging file 또는 stdin 방식으로 전달된다.
- `world_stage_input`이 staging file을 만들고 후속 tool은 path/hash만 받는다.
- trusted wrapper/adapter가 authenticated session metadata를 받아 `auth_context_file`을 생성하고, 해당 파일의 hash를 계산해 `auth_context_hash`와 함께 tool 변수로 채운다. production auth context input은 signature/MAC 또는 configured wrapper trust-material 검증과 expected issuer/audience/scope policy를 통과해야 하고, `auth_context_hash`는 integrity binding이다.
- `auth_context_file`에는 expiry, scope, approver/session identifiers, approval channel, authenticated actor를 포함한 provenance metadata가 들어가며, wrapper는 세션 종료 또는 attestation 사용 후 이를 정리/무효화한다. local fixture mode는 `WORLD_TOOL_TEST_AUTH_CONTEXT=1` explicit opt-in 테스트 전용 경로다.
- `world_create_approval_attestation`은 wrapper가 주입한 `auth_context_file/auth_context_hash`, expected issuer/audience/scope policy, actor/channel provenance를 항상 검증하고, raw actor 문자열만으로 승인 provenance를 만들지 않는다.
- approval attestation 관련 dynamic tool 변수는 trusted wrapper/adapter 경계에서만 채워지고, 사용자 입력만으로는 생성되거나 덮어쓰여서는 안 된다.
- `world_accept_draft`는 diff binding 값을 필수로 받는다.
- `world_get_run_artifact`는 safe artifact basename allowlist와 path boundary 검증을 강제하고, inbox payload나 unredacted sensitive artifact는 노출하지 않는다.
- staged-input-consuming commands는 hash mismatch를 command-level `INPUT_HASH_MISMATCH`로 반환한다.
- 범용 `world_exec_shell` 같은 tool은 제공하지 않는다.
- malformed JSON, timeout, non-zero exit에 대한 실패 메시지 정책이 정리되어 있다.

## 11. Milestone 8: Sample World E2E
### 목표
샘플 world root로 핵심 workflow를 먼저 검증한다.

### 산출물
- `examples/worlds/ashen-continent/content/...`
- character/place/event 최소 3개 canon 문서
- pass draft fixture
- invalid change_type fixture
- create with target_id/retcon_reason fixture
- missing target_id fixture
- target id mismatch fixture
- create id conflict fixture
- missing change_type fixture
- conflict draft fixture
- update/retcon draft fixture
- missing retcon_reason fixture
- missing update/deprecate target fixture
- convenience-vs-explicit relationship conflict fixture
- inverse/symmetric contradiction fixture
- related-only-active-draft target fixture
- staged input hash mismatch fixture
- target path collision fixture
- registry safe absolute path normalization fixture
- registry traversal rejection fixture
- registry symlink escape rejection fixture
- registry directory-confusion rejection fixture
- config safe absolute path normalization fixture
- config traversal rejection fixture
- config symlink escape rejection fixture
- config directory-confusion rejection fixture
- `../` traversal fixture
- absolute path fixture
- symlink escape fixture
- unsafe run artifact basename traversal fixture
- invalid `--auth-context-file` location fixture
- relationship domain/range mismatch fixture
- active draft only target at accept fixture
- recovery resolution fixture
- run get redacted manifest/status fixture
- run get safe artifact retrieval fixture
- run get inbox/sensitive artifact rejection fixture
- unresolved recovery inspection before run recover fixture
- storylet accept block fixture
- storylet content validation fixture
- missing auth context approval attestation fixture
- auth context hash mismatch approval attestation fixture
- auth context expiry approval attestation fixture
- auth context scope denial approval attestation fixture
- fixture opt-in missing approval attestation fixture
- actor/channel mismatch approval attestation fixture
- accept-time approval attestation hash/binding mismatch fixture
- active draft path/type wrong-directory fixture
- force denied fixture
- lock/base-hash mismatch fixture
- relationship allowlist fixture
- reject fixture
- stdout JSON fixture
- manual OpenCrabs test script 또는 checklist

### 완료 기준
- `world init -> registry add -> input stage(title/body) -> draft create -> draft validate -> draft diff -> input stage(reason) -> approval attest -> draft accept`가 샘플 world에서 통과한다.
- conflict draft는 기본 accept에서 차단된다.
- invalid change_type draft는 validation `error`로 보고되고 accept에서 blocked된다.
- create에 target_id 또는 retcon_reason이 들어간 draft는 validation `error`로 보고되고 accept에서 blocked된다.
- missing change_type draft는 validation `error`로 보고되고 accept에서 blocked된다.
- missing target_id draft는 validation `error`로 보고되고 accept에서 blocked된다.
- target id mismatch draft는 validation `error`로 보고되고 accept에서 blocked된다.
- create id conflict draft는 validation `conflict`로 보고되고 accept에서 blocked된다.
- missing retcon_reason draft는 validation `error`로 보고되고 accept에서 blocked된다.
- missing update/deprecate target fixture는 no draft write를 보장하고 `command_status: "blocked"`, `data.block_reason: "MISSING_TARGET"`로 종료되며 draft-bound available actions를 노출하지 않는다.
- convenience-vs-explicit relationship conflict는 conflict로 탐지된다.
- inverse/symmetric contradiction은 conflict로 탐지된다.
- diff binding mismatch는 accept에서 차단된다.
- relationship domain/range mismatch는 conflict로 탐지된다.
- related id 또는 relationship target이 active draft에만 존재하면 draft validate에서는 warning, accept/diff에서는 `command_status: "blocked"`, `data.block_reason: "MISSING_TARGET"`, `data.validation_status: "conflict"`로 처리된다.
- staged input hash mismatch는 hash binding을 소비하는 모든 command에서 command-level `INPUT_HASH_MISMATCH`로 반환된다.
- path boundary safety fixtures는 `../` traversal, absolute path, symlink escape, unsafe run artifact basename traversal, invalid `--auth-context-file` location 사례를 모두 커버한다.
- registry/config boundary fixtures는 safe absolute path normalization, traversal rejection, symlink escape rejection, directory-confusion rejection을 각각 커버한다.
- recovery resolution fixture는 `run recover`가 recorded transaction state를 idempotently resolve하고 `recovery.json`을 resolved로 남긴 뒤, 이후 write command가 다시 허용되는지 검증하며, 원래 `draft accept` 재실행이 recovery path가 아님을 확인한다.
- `run get` redacted manifest/status fixture는 run metadata의 민감 필드를 가리고 최소 상태만 노출하는지 검증한다.
- `run get --artifact recovery.json` 또는 동등한 allowlisted safe artifact fixture는 recovery artifact만 안전하게 조회되고 `runs/inbox/**`나 sensitive artifact는 거부되는지 검증한다.
- unresolved recovery inspection before `run recover` fixture는 `run get`으로 unresolved recovery 상태를 먼저 확인한 뒤 `run recover`로 해소하는 흐름을 검증한다.
- storylet draft는 content canon accept에서 차단된다.
- storylet content path/status 위반은 validation `error`로 보고되고 accept에서 blocked된다.
- approval attestation 안전 fixture는 누락된 auth context, hash mismatch, expiry, scope denial, fixture opt-in missing, actor/channel mismatch, accept-time hash/binding mismatch 사례를 모두 커버한다.
- active draft path/type wrong-directory fixture는 non-storylet draft가 `drafts/storylets/`에 있는 경우와 `storylet` draft가 `drafts/storylets/` 밖에 있는 경우를 함께 검증한다.
- 알 수 없는 relationship type은 conflict로 탐지된다.
- rejected draft는 `archive/rejected/`로 이동한다.
- accepted draft는 `content/`에 반영되고 `archive/accepted/`로 이동한다.
- `runs/` artifact만 보고 어떤 변경이 있었는지 추적할 수 있다.

## 12. Milestone 9: Container Runtime
### 목표
OpenCrabs, `world-tool`, skill/tools bundle을 컨테이너에서 운영할 수 있게 한다.

### 산출물
- Dockerfile
- docker-compose 예시
- OpenCrabs config/auth volume
- world root volume
- per-world container 예시
- Codex OAuth provider 기본 설정 안내
- Codex CLI provider fallback 설정 안내

### 완료 기준
- OpenCrabs와 `world-tool`이 같은 컨테이너에서 실행될 수 있다.
- OpenCrabs credential/config volume은 world root volume과 분리된다.
- world root는 특정 폴더만 volume mount된다.
- Codex OAuth provider 사용 시 컨테이너에 Codex CLI나 `~/.codex` 마운트가 필요하지 않다.
- Codex CLI provider fallback을 사용할 때만 별도 auth volume을 사용한다.

## 13. Milestone 10: Post-MVP Hardening
### 목표
MVP 이후 장기 운영과 maintenance path를 추가한다.

### 후보 작업
- validation rule strictness를 world별로 설정
- graph rebuild/check
- archive pruning/compression/export
- retcon/versioning report
- schema migration and migration report
- `content migrate --dry-run` report-only maintenance workflow
- migration boundary fixture
- OpenCrabs tool calling retry/timeout 정책
- semantic search integration
- storylet/exporter

### 완료 기준
- 장편 세계관에서 archive와 runs가 늘어나도 운영 정책이 있다.
- graph는 content에서 재생성 가능한 인덱스로 유지된다.
- validator가 확정 판정기가 아니라 conflict 후보 탐지기라는 경계가 유지된다.
- `content migrate`는 `--dry-run`만 허용하고 report/artifact만 emit하며 content를 직접 변경하지 않는 report-only maintenance path다. no-option과 `--apply`는 `INVALID_ARGUMENT`다.

### Migration workflow
- `content migrate --dry-run`은 warning-only legacy/import 이슈를 actionable report로 묶고 blocked 항목과 분리하되 content를 직접 변경하지 않는다.
- migration report는 source 문서, path move 여부, field normalization 결과, before/after hash, blocker 목록을 포함한다.
- migration 완료 기준은 warning이 사라졌는지가 아니라, warning이 의사결정 가능한 action item으로 정리되고 blocked 항목이 남지 않았는지다.

## 14. 권장 구현 순서
최소 vertical slice는 다음 순서로 구현한다.

1. Milestone 0: Repository Scaffold
2. Milestone 1: World Root Boundary
3. Milestone 2: Content Read Model
4. Milestone 3: Draft Lifecycle
5. Milestone 4: Validation MVP
6. Milestone 5: Diff, Accept, Audit
7. Milestone 6: OpenCrabs Skill
8. Milestone 7: OpenCrabs Dynamic Tools
9. Milestone 8: Sample World E2E
10. Milestone 9: Container Runtime

Milestone 10은 MVP가 end-to-end로 동작한 뒤 진행한다.

## 15. MVP Done Definition
MVP는 다음을 만족할 때 완료로 본다.

- OpenCrabs 없이 `world-tool`만으로 draft를 만들고 accept할 수 있다.
- `content/`는 accept 전까지 변경되지 않는다.
- validation 결과와 accept 결과가 JSON으로 반환된다.
- JSON envelope는 `command_status`와 `data.validation_status`를 분리한다.
- conflict/error draft는 기본 accept에서 차단된다.
- diff와 accept는 binding hash로 묶인다.
- 모든 write operation은 `runs/`와 `archive/`에 추적 가능한 기록을 남긴다.
- OpenCrabs skill과 dynamic tools가 같은 workflow를 호출할 수 있다.
- 샘플 world에서 end-to-end 시나리오가 재현된다.
