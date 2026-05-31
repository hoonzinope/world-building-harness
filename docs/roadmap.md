# roadmap.md

# OpenCrabs World-Building Roadmap

## 1. 방향
로드맵은 별도 agent runtime을 만드는 것이 아니라, OpenCrabs에 세계관 빌딩 skill과 dynamic tools를 얹는 방식으로 진행한다. deterministic 파일 작업은 Go 단일 바이너리 `world-tool`이 담당한다.

현재 레포는 문서와 설계 중심의 MVP 준비 단계다. 실제 Go CLI, `opencrabs/skills/`, `opencrabs/tools/`, 샘플 world root는 아직 구현 대상으로 남아 있다.

구체적인 구현 이정표와 완료 기준은 `docs/implementation-plan.md`를 따른다. 이 문서는 제품 phase를 설명하고, implementation plan은 실제 작업 순서를 추적한다.

## 2. Phase 0: 결정 고정
### 목표
제품 경계와 MVP 범위를 확정한다.

### 완료 기준
- OpenCrabs가 하네스/오케스트레이터라는 결정이 문서화됨
- 이 레포는 OpenCrabs skill/tools bundle이라는 결정이 문서화됨
- `world-tool`은 Go CLI라는 결정이 문서화됨
- content Markdown이 canon source of truth라는 결정이 문서화됨
- draft와 content 경계가 문서화됨

## 3. Phase 1: world-tool Core MVP
### 목표
OpenCrabs 없이도 로컬 CLI로 파일 관리와 validation을 수행할 수 있게 한다.

### 기능
- `world-tool world init`
- `world-tool world status`
- `world-tool registry add/list/remove/default`
- `world-tool input stage`
- `world-tool doc list --scope active`
- `world-tool doc read/search --scope active`
- `world-tool draft create/update/read/list`
- `world-tool draft validate`
- `world-tool content validate`
- `world-tool draft diff`
- `world-tool approval attest`
- `world-tool draft accept`
- `world-tool draft reject`
- `world-tool run list`
- `world-tool run get`
- `world-tool run get --artifact <basename>`
- `world-tool run recover`

### 구현 주의
- path boundary와 symlink resolution을 가장 먼저 구현한다.
- Markdown parser와 YAML frontmatter parser는 round-trip 안정성을 기준으로 선택한다.
- 긴 draft body, 검색 query, title, reason, retcon_reason은 command-line argument가 아니라 stdin 또는 world root 내부 `runs/inbox/` staging file로 받는다.
- `world_stage_input`과 `world_create_approval_attestation`이 돌려준 file path와 hash만 후속 tool에 넘기고, `authenticated_actor`는 OpenCrabs 인증 세션에서만 채운다. `auth_context_hash`는 integrity binding이며 production trusted auth context input은 먼저 configured wrapper-owned auth-context boundary의 normalized regular file 또는 trusted request-file/FD mechanism을 만족해야 하고, traversal/symlink/selected world root/runtime-owned run dir/staged inbox는 hash/parse/signature/MAC/trust-material 검증 전에 거부되며, 그 뒤에 signature/MAC 또는 configured wrapper trust-material 검증과 expected issuer/audience/scope policy를 만족해야 한다.
- 모든 command는 `commands.md`의 JSON envelope와 exit code 정책을 일관되게 지킨다.
- write command는 world root lock을 사용한다.
- diff와 accept는 hash binding으로 묶는다.

### 완료 기준
- draft markdown 생성 가능
- content는 accept 전까지 변경되지 않음
- validate 결과가 runs에 저장됨
- accept 시 content로 승격됨
- accepted draft가 archive/accepted/로 이동함
- unresolved recovery가 있으면 write command가 차단되고 `world-tool run recover`만 write 예외로 동작함
- `world-tool run get`은 redacted run metadata를 조회함
- `world-tool run get --artifact <basename>`은 basename allowlist와 path boundary 검증을 통과한 safe artifact만 노출함

## 4. Phase 2: Schema & Validator 강화
### 목표
OpenCrabs/Codex 생성물을 안정적으로 검사할 rule 기반 validator를 만든다.

### 기능
- frontmatter 검사
- `world-tool content validate`
- id 중복 검사
- change_type create/update/deprecate 검사
- required field 검사
- timeline 검사
- relationship 검사
- orphan related id 검사
- validation report 생성
- world별 validation strictness 설정은 post-MVP hardening에서 다룬다
- frontmatter `retcon_reason`/source_run_id 기반 retcon 추적

### 완료 기준
- 구조 오류를 error로 탐지
- `change_type: create` id 중복을 conflict로 탐지
- timeline 충돌 후보를 warning/conflict로 탐지
- active draft에만 있는 relationship target은 accept에서 conflict로 탐지

## 5. Phase 3: OpenCrabs Skill
### 목표
OpenCrabs에서 세계관 작업 규칙을 재사용 가능한 skill로 제공한다.

### 기능
- `opencrabs/skills/world-building/SKILL.md`
- draft-first workflow 지침
- accept 전 사용자 승인 지침
- tool 사용 우선순위 지침
- 응답 format 지침
- validation warning 무시 또는 accept 유도 요청에 대한 대응 지침

### 완료 기준
- OpenCrabs `/skills`에서 world-building skill을 실행할 수 있음
- skill이 content 직접 수정을 금지하도록 안내함
- skill이 validation/accept 순서를 안내함

## 6. Phase 4: OpenCrabs Dynamic Tools
### 목표
OpenCrabs가 `world-tool`을 의미 단위 tool로 호출하게 한다.

### 기능
- `world_list`
- `world_stage_input` (returns input_path/input_hash)
- `world_status`
- `world_search_docs` (`doc search --scope active`)
- `world_read_doc`
- `world_create_draft`
- `world_create_update_draft`
- `world_create_deprecate_draft`
- `world_update_draft`
- `world_read_draft`
- `world_validate_draft`
- `world_validate_content`
- `world_diff_draft`
- `world_create_approval_attestation`
- `world_accept_draft`
- `world_force_accept_draft`
- `world_reject_draft`
- `world_list_runs`
- `world_get_run`
- `world_get_run_artifact`
- `world_recover_run`

### 완료 기준
- OpenCrabs에서 dynamic tools가 로드됨
- 각 tool은 stdout JSON을 반환함
- 범용 shell tool 없이 세계관 workflow를 수행할 수 있음
- 긴 markdown body를 file/stdin 기반으로 전달하는 tool이 동작함
- trusted wrapper/adapter가 authenticated session metadata를 받아 `auth_context_file`을 생성하고 hash를 계산해 `auth_context_hash`와 함께 tool 변수로 채움. production trusted auth context input은 configured wrapper-owned auth-context boundary의 normalized regular file 또는 trusted request-file/FD mechanism에서만 받아야 하고, traversal/symlink/selected world root/runtime-owned run dir/staged inbox는 hash/parse/signature/MAC/trust-material 검증 전에 거부해야 하며, `auth_context_hash`는 integrity binding만 담당함
- `auth_context_file`의 trusted contents는 expiry, scope, session metadata, approval channel, authenticated actor, downstream_action provenance 등이다. wrapper는 세션 종료 또는 attestation 사용 후 이를 정리/무효화한다. local fixture mode는 `WORLD_TOOL_TEST_AUTH_CONTEXT=1` explicit opt-in 테스트 전용 경로임
- `approver_id`는 CLI/template input으로 별도 전달되는 non-authoritative audit/display label로 attestation과 audit에 기록됨.
- approval attestation은 wrapper가 주입한 auth-context provenance를 `auth_context_file/auth_context_hash`, expected issuer/audience/scope policy, actor/channel 기준으로 검증하고, raw actor 문자열만으로 승인 provenance를 만들지 않음
- `approval attest`는 invalid `auth_context_file` location과 path escape를 거부하며, configured wrapper-owned auth-context boundary 또는 trusted request-file/FD를 먼저 만족한 production trusted auth context input만 read-only 예외로 허용하고, boundary-before-hash/parse/signature/MAC/trust-material ordering을 유지함
- registry/config file path validation은 null-root registry behavior에서 explicit absolute registry/config file path를 safe normalization/validation 후 허용하되, unsafe traversal, symlink escape, directory confusion, world-root document/artifact path escape는 거부함
- safe artifact retrieval은 명시적 basename allowlist와 path boundary 검증만 허용함
- recovery inspection은 `world_get_run`과 필요 시 `world_get_run_artifact`의 조회로 가능해야 하며, `world_recover_run`은 unresolved recovery 상태만 해소해야 함
- diff 확인과 accept 실행이 같은 diff_run_id/hash binding으로 묶임

## 7. Phase 5: OpenCrabs Integration UX
### 목표
OpenCrabs 대화에서 draft 생성, 검증, 승인 흐름이 자연스럽게 동작하게 한다.

### 기능
- draft 요약
- validation warning 표시
- diff 확인
- accept/reject 안내
- run id 추적

### 완료 기준
- 사용자가 OpenCrabs 대화로 draft를 만들고 검증할 수 있음
- conflict 상태에서는 승인 전 경고가 표시됨
- accept는 명시적 사용자 승인 후에만 실행됨

## 8. Phase 6: Sample World E2E
### 목표
샘플 world root로 end-to-end 흐름을 검증한다.
세부 fixture 체크리스트는 `docs/implementation-plan.md`의 Milestone 8을 canonical 기준으로 삼는다.

### 기능
- `examples/worlds/ashen-continent`
- 최소 canon 문서 3개 이상
- draft 생성, validation conflict, update/retcon, accept/reject 기본 흐름 fixture
- path boundary 및 registry/config boundary fixture
- approval attestation 안전 fixture
- run recovery/get 라이프사이클 fixture
- storylet, force, lock/base-hash, relationship allowlist fixture
- OpenCrabs skill + tools 수동 테스트 스크립트

### 완료 기준
- init → registry add → input stage(title/body) → draft create → draft validate → draft diff → input stage(reason) → explicit approval over diff binding + staged reason file/hash checkpoint → approval attest → draft accept가 샘플 world에서 동작함
- conflict draft가 accept에서 차단됨
- diff binding mismatch가 accept에서 차단됨
- storylet draft가 content canon으로 승격되지 않음
- path boundary, registry/config boundary, approval attestation 안전 fixture가 Milestone 8 기준으로 모두 커버된다.
- recovery resolution, redacted run metadata, safe artifact retrieval, inbox/sensitive artifact rejection, unresolved recovery inspection before `run recover` 흐름이 샘플 world에서 검증된다.
- `run get`은 redacted manifest/status만 노출하고 allowlisted safe artifact만 반환하며, inbox/sensitive artifact는 거부한다.
- `run recover`는 unresolved recovery를 idempotently resolve하고 `recovery.json`을 resolved로 남긴다.
- accepted draft가 archive/accepted/로 이동함
- runs artifact가 재현 가능한 형태로 남음

## 9. Phase 7: Container Runtime
### 목표
OpenCrabs, `world-tool`, skill/tools bundle을 컨테이너에서 운영할 수 있게 한다.

### 기능
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

## 10. Phase 8: Post-MVP Hardening
### 목표
MVP 이후 안정화와 maintenance path를 추가한다. Graph Store와 Storylet & Exporter는 이 단계 아래의 post-MVP candidate work로만 다루고, 실제 구현 순서는 `docs/implementation-plan.md`를 따른다.

### 기능
- world별 validation strictness
- schema migration and migration report
- `content migrate --dry-run` report-only maintenance path
- archive pruning/compression/export
- OpenCrabs tool calling retry/timeout 정책
- migration boundary fixture

### 후보 작업
- Graph Store: nodes.json 생성, edges.json 생성, graph rebuild/check, orphan report
- Storylet & Exporter: storylet draft 생성, raw note를 schema markdown으로 변환, canon impact validation

### 완료 기준
- `content migrate`는 `--dry-run`만 허용하고 report와 artifact만 남기며 content를 직접 변경하지 않는 report-only maintenance path다. no-option과 `--apply`는 `INVALID_ARGUMENT`다.
- migration 결과가 blocked/warning/action item으로 분리됨
- post-MVP maintenance path가 MVP core와 분리됨

## 11. 우선순위
세부 체크리스트는 `docs/implementation-plan.md`의 milestone 순서를 기준으로 한다.

최우선:
1. Go `world-tool` CLI
2. validation report
3. runs log
4. accept workflow
5. OpenCrabs skill
6. OpenCrabs dynamic tools
7. sample world end-to-end test
8. container runtime

나중:
1. graph rebuild/check
2. storylet/exporter
3. semantic search
4. dashboard
5. OpenCrabs native extension 또는 deeper integration

## 12. 리스크
### skill만으로 규칙을 강제하려는 위험
Skill은 지침일 뿐이다. content 보호, validation, accept 차단은 tool에서 강제해야 한다.

### dynamic tool 권한 과대화
범용 shell tool을 열면 안전 경계가 무너진다. 의미 단위 `world_*` tools만 제공한다.

### source of truth 혼선
OpenCrabs DB나 graph를 canon 원본처럼 다루면 content Markdown과 불일치가 생긴다. MVP에서는 content Markdown을 우선한다.

### validator 과신
validator는 확정 판정기가 아니라 충돌 후보 탐지기다.

### OpenCrabs tool calling 안정성
Tool 호출 실패, malformed JSON, timeout에 따라 UX가 흔들릴 수 있다. OpenCrabs는 `ok`, `command_status`, `data.validation_status`, `data.block_reason`, `issues`, `available_actions`, `error.code` 순서로 결과를 해석하고, validation 결과와 domain blocked 결과를 구분해야 한다. OpenCrabs skill은 실패 시 재시도보다 사용자에게 상태를 설명해야 한다.

### warning 무시 유도
사용자가 “warning 무시하고 accept”를 요청할 수 있다. conflict/error는 tool에서 차단하고, force는 reason, approval attestation provenance, audit log를 필수로 한다.

### archive storage 증가
accepted/rejected archive가 계속 쌓일 수 있다. archive pruning, compression, export 정책을 나중 phase에서 추가한다.
