# security-boundary.md

# OpenCrabs World Tools Security Boundary

## 1. 목적
OpenCrabs/Codex가 세계관 파일을 다루는 구조에서는 명확한 보안 경계가 필요하다. 목표는 world root 밖 파일, secret, host system이 tool 호출로 노출되거나 변경되지 않게 하는 것이다. `TRANSACTION_INCOMPLETE`는 failed/recovery-required partial transaction이며, 이 상태가 unresolved인 동안에는 같은 world root에 대한 후속 write를 허용하지 않는다. 복구는 `world_recover_run` / `world-tool run recover`만 허용하며, 원래 write command를 직접 다시 실행해서 복구를 대신하는 경로는 허용하지 않는다.

## 2. 기본 원칙
- 세계관 파일 작업은 `world_*` dynamic tools로 수행한다.
- dynamic tools는 `world-tool` Go CLI를 호출한다.
- `world-tool`은 선택된 world root 밖 파일을 기본적으로 읽거나 쓰지 않는다. 유일한 read 예외는 `approval attest`가 `--auth-context-file`/`--auth-context-hash`로 검증하는 trusted auth context input이며, 이 파일도 쓰거나 run artifact로 복사하지 않는다. production auth context input은 wrapper-signed 또는 MACed envelope여야 하고 configured wrapper trust material으로 검증되며 expected issuer/audience/scope policy를 만족해야 한다. local fixture mode는 test-only이고 explicit opt-in이 필요하다. 이 input은 `world_id`, `allowed_actions`/scope, `issuer`, `audience`, `authenticated_actor`, `approval_channel`, and the exact `downstream_action`을 함께 묶는 wrapper provenance다. approval attestation은 `runs/inbox/*-approval-attestation.json`에 staged되는 별도 아티팩트이며, `world_create_approval_attestation`과 exact downstream action(`world_accept_draft` 또는 `world_force_accept_draft`)을 함께 bind해야 한다. auth context input의 exact downstream action이 없거나 `--downstream-action`과 mismatch면 `AUTH_CONTEXT_SCOPE_DENIED`다. attestation 생성만 허용하는 scope는 content mutation에 충분하지 않으며, staged attestation의 accept/force binding mismatch는 `APPROVAL_ATTESTATION_BINDING_MISMATCH`로 다룬다. 세부 계약은 `docs/commands.md`를 따른다.
- `content/`는 accept tool에서만 변경된다.
- draft 생성과 canon 승격은 분리한다.
- OpenCrabs/Codex 출력은 후보이며 tool validation을 통과해야 한다.
- 모든 write 작업은 runs log에 기록한다.
- unresolved recovery가 있으면 같은 world root에 대한 `world init`, `input stage`, `approval attest`, `draft create`, `draft update`, `draft validate`(validation artifact writer), `draft diff`, `draft accept`, `draft reject`, `content validate` artifact writer, `content migrate` report writer, 기타 content report writer를 포함한 모든 write command가 차단되며, `world_recover_run`만 write 예외다. read-only inspection은 허용한다.

## 3. 파일 시스템 경계
### 허용 경로
선택된 world root 내부에서만 허용한다.

- content/
- drafts/
- raw/
- graph/
- schema/
- runs/
- runs/inbox/
- archive/
- harness.yaml

`runs/inbox/`는 privileged transient staging area다. 일반 browse/list/get/audit 대상이 아니며, `input stage`와 `approval attest`만 여기에 write할 수 있다. unresolved recovery가 있으면 이 staging area에 대한 write도 막힌다.

### 금지 경로
- world root 상위 디렉토리
- 다른 world root
- 사용자의 home directory
- `.ssh/`
- `.git-credentials`
- secret이 담긴 `.env`
- Docker socket
- system path

## 4. Path Traversal 방지
사용자 입력으로 들어온 path는 반드시 normalize 후 world root 내부인지 검사한다.

금지 예시:

```text
../../.ssh/id_rsa
~/private.txt
/var/run/docker.sock
```

## 5. Dynamic Tool Boundary
나쁜 tool:

```toml
[[tools]]
name = "world_exec_shell"
executor = "shell"
command = "{{command}}"
```

좋은 tool:

```toml
[[tools]]
name = "world_accept_draft"
executor = "shell"
command = "world-tool draft accept --world {{world_id}} --draft {{draft_path}} --diff-run-id {{diff_run_id}} --draft-hash {{draft_hash}} --target-base-hash {{target_base_hash}} --patch-hash {{patch_hash}} --approver-id {{approver_id}} --approval-channel {{approval_channel}} --approval-attestation-file {{approval_attestation_file}} --approval-attestation-hash {{approval_attestation_hash}} --authenticated-actor {{authenticated_actor}} --reason-file {{reason_file}} --reason-hash {{reason_hash}} --json"
```

tool은 의미 단위 작업이어야 하며 shell 권한을 넓게 열지 않는다.
긴 markdown body, 검색 query, title, reason, retcon_reason, note는 command line argument가 아니라 stdin 또는 world root 내부 `runs/inbox/` staging file로 전달한다.

긴 입력 staging은 `world_stage_input`이 호출하는 `world-tool input stage`가 담당한다. approval attestation staging은 별도 `approval attest` 경로다. OpenCrabs/Codex가 임의 path에 직접 파일을 만들었다고 가정하지 않는다.

OpenCrabs shell executor는 template 값을 argv-safe하게 escape해야 한다. 그 보장이 없으면 dynamic tool command에 사용자 입력 변수를 직접 넣지 않고, request JSON file 또는 stdin payload를 받는 wrapper command를 사용한다. raw shell interpolation은 허용하지 않는다.

`runs/inbox/` 정책:
- world root 내부에 있어야 한다.
- active command가 읽은 뒤 해당 run artifact로 복사하거나 삭제한다.
- symlink는 허용하지 않는다.
- 파일 크기 상한을 둔다. MVP 기본값은 문서당 1 MiB를 권장한다.

Command별 path allowlist:
- `doc list/read/search --scope active`: `content/**/*.md`, `drafts/**/*.md`
- `doc search --query-file`: `runs/inbox/**` read/consume only for staged query payloads
- `draft create/update`: generated draft paths under `drafts/**/*.md`, staged title/body/retcon_reason inputs under `runs/inbox/**` read/consume only
- `draft read/validate/diff/accept/reject`: draft paths under `drafts/**/*.md`, staged reason/approval attestation inputs under `runs/inbox/**` read/consume only where the command defines them
- `content validate`: `content/**/*.md`
- `input stage`: `runs/inbox/**` write only
- `approval attest`: `runs/inbox/**` write only for approval attestation artifacts; `--auth-context-file`/`--auth-context-hash` may read a trusted auth context input path outside the world root in read-only mode after signature/MAC/trust-material verification plus expected issuer/audience/scope checks. 승인 attestation은 `runs/inbox/*-approval-attestation.json`으로 staged되며 `world_create_approval_attestation`과 exact downstream action(`world_accept_draft` 또는 `world_force_accept_draft`)을 함께 bind해야 한다. attestation 생성만 허용하는 scope는 content mutation에 충분하지 않으며, 상세 필드 계약은 `docs/commands.md`를 따른다.
- `run get`: 기본 redacted manifest/status summary만 반환하고, explicit safe artifact allowlist 또는 dedicated safe-artifact command가 있을 때만 추가 artifact를 읽는다. `runs/inbox/**`는 제외한다.
- `run list`: immutable run index/summary files only, `runs/inbox/**` 제외
- `run recover`: unresolved `recovery.json`을 정리하는 repair path
- `archive/`, `raw/`, `schema/`는 MVP dynamic tool의 doc 조회 대상이 아니다.

## 6. Network Boundary
`world-tool` MVP는 임의 네트워크 요청을 수행하지 않는다.

OpenCrabs provider가 Codex OAuth/OpenAI/Claude/Gemini 등으로 네트워크를 사용하는 것은 OpenCrabs 설정의 책임이다. world 파일 작업 tool은 외부 URL fetch, repo clone, arbitrary curl을 수행하지 않는다.

## 7. Secret Handling
API key, OAuth token, bot token은 다음 위치에 저장하지 않는다.

- runs/
- drafts/
- content/
- validation report
- graph/
- world root 내부

OpenCrabs credential은 OpenCrabs의 credential store나 별도 secret mount로 관리한다. 기본 provider는 Codex OAuth이며, `world-tool`은 provider API key를 필요로 하지 않는다.

Credential/config volume과 world root volume은 분리한다.

좋음:
```text
opencrabs-config -> /home/opencrabs/.opencrabs
/host/worlds/ashen-continent -> /workspace/world
```

나쁨:
```text
/host/worlds/ashen-continent -> /home/opencrabs
/host/worlds/ashen-continent -> /home/opencrabs/.opencrabs
```

Codex CLI provider fallback을 사용할 때만 별도 Codex auth volume이 필요하다. 이 경우에도 Codex auth는 world root 내부에 두지 않는다.

## 8. LLM Output Boundary
OpenCrabs/Codex는 다음을 직접 수행하지 않는다.

- content 직접 수정
- accept 우회
- validation 생략
- graph 확정 업데이트
- 다른 world root 접근
- OpenCrabs 보안 설정 변경

LLM은 후보 markdown과 판단을 만들 수 있지만, 파일 상태 변경은 `world-tool`이 수행한다.

## 9. Approval Boundary
draft가 content로 승격되려면 `world_accept_draft`를 통과해야 한다.

기본 차단 조건:
- validation error
- validation conflict
- `change_type: create` id 중복
- `change_type: update/deprecate` target_id 누락 또는 target_id 불일치
- content target path 충돌
- required field 누락
- draft가 active drafts/ 밖에 있음

force accept는 가능하지만 reason만으로는 부족하다. `reason_file`/`reason_hash`, `approval_attestation_file`/`approval_attestation_hash`, `approver_id`, `approval_channel`, `authenticated_actor`를 함께 기록해야 하며, 이는 runs log와 accept result에 남아야 한다. `approval_channel` 예시는 `OpenCrabs-chat`을 사용하고, attestation, accept, audit 전반에서 byte-identical 값이어야 한다. `approval_channel`과 `authenticated_actor`는 `world_create_approval_attestation`이 확인한 trusted auth context input과 그로부터 staged된 approval attestation으로부터 와야 하며, production input은 wrapper-signed 또는 MACed envelope와 configured trust material, expected issuer/audience/scope policy를 만족해야 한다. 그 input scope는 `world_create_approval_attestation`과 exact downstream action(`world_accept_draft` 또는 `world_force_accept_draft`)을 함께 authorize해야 하고, exact downstream-action binding이 없거나 mismatch면 `AUTH_CONTEXT_SCOPE_DENIED`로 실패해야 한다. attestation 생성만 허용하는 scope는 content mutation에 충분하지 않으며, staged attestation의 accept/force binding mismatch는 `APPROVAL_ATTESTATION_BINDING_MISMATCH`다. 모델 출력이나 staging file에서 오면 안 된다.

force accept 제한:
- semantic/timeline/relationship conflict 후보만 우회 대상으로 삼는다.
- missing target, missing related target, missing relationship target, missing update/deprecate target, active-draft-only target은 `MISSING_TARGET`로 처리되며 force accept로 우회할 수 없다.
- path/type/id/schema 불일치, structural error, id conflict, target path conflict, diff binding mismatch, storylet canon 승격, atomic write 실패, lock 실패는 force accept로 우회할 수 없다.

warning은 accept를 차단하지 않지만, accept reason에 warning을 확인했다는 맥락을 남긴다.

## 10. Locking Boundary
write command는 world root 단위 lock을 사용한다.

대상:
- draft create/update/reject
- diff artifact write
- accept
- run artifact write

정책:
- lock 파일은 world root 내부 `runs/.lock` 또는 동등한 위치에 둔다.
- lock 획득 실패는 `LOCK_BUSY` JSON error로 반환한다.
- accept는 lock을 잡은 뒤 validation을 재실행한다.
- accept는 diff_run_id, draft_hash, target_base_hash, patch_hash binding을 검증한다.
- create 경로는 target absence check와 CLI `--target-base-hash none`을 사용하고, update/deprecate 경로는 sha256 `target_base_hash`를 사용한다. `diff_run_id`, `draft_hash`, `target_base_hash`, `patch_hash`는 `world_diff_draft`의 출력에서 가져와야 하며, create 경로에서는 JSON의 `target_base_hash: null`을 CLI template 변수 `target_base_hash="none"`으로 매핑해야 한다. staged file hash는 `world_stage_input`의 출력에서 가져오고, staged approval attestation은 exact downstream action binding과 함께 이 값들을 묶어야 한다. accept 시점에 world-tool이 다시 계산한 값과 일치해야 한다.
- diff 시점의 draft/content/patch hash와 accept 시점의 값이 다르면 accept를 중단한다.
- accept는 `reason_file`/`reason_hash`, `approval_attestation_file`/`approval_attestation_hash`, `approver_id`, `approval_channel`, `authenticated_actor`를 audit field로 기록한다. free-form reason이나 actor 문자열만으로는 승인 provenance가 충분하지 않다. staged approval attestation의 embedded downstream_action은 accept/force command와 exact match여야 한다. canonical 세부 계약은 `docs/commands.md`를 따른다.

## 11. Docker Boundary
권장 컨테이너 실행 원칙:
- OpenCrabs credential/config volume과 world root volume을 분리한다.
- per-world tool container에는 선택된 world root 하나만 마운트한다.
- 여러 world root를 한 컨테이너에 동시에 마운트하지 않는다.
- docker.sock을 마운트하지 않는다.
- read-only root filesystem을 고려한다.
- 네트워크 권한은 최소화한다.
- 컨테이너 유저는 root가 아닌 전용 유저를 사용한다.

예시:

```bash
docker run --rm \
  --user 1000:1000 \
  --network none \
  --cap-drop ALL \
  --security-opt no-new-privileges \
  --read-only \
  --tmpfs /tmp \
  -v /host/worlds/ashen-continent:/workspace/world:ro \
  world-tool:latest \
  world-tool draft read --root /workspace/world --world-id ashen-continent --draft drafts/nations/nation_northern_empire.md --json
```

Docker root-only 실행에서는 world_id provenance를 bind mount path에서 추측하지 않는다. command site에서 `--world-id`를 명시하거나, 실행 전에 `harness.yaml` provenance를 확인할 수 있어야 하고, 둘 다 불가능하면 root-only 모드를 쓰지 않는다. `draft read`는 read-only inspection이지만 `draft validate`는 계약상 artifact를 쓸 수 있으므로 `:ro` world mount로 실행하지 않는다.

## 12. Audit Log
모든 write tool은 다음을 기록한다.

- run id
- tool name
- input summary
- modified files
- validation status
- actor
- timestamp
- force 여부와 reason
- before/after content hash
- diff_run_id, draft_hash, target_base_hash, patch_hash
- lock 획득/해제 event
- redaction 여부
- transaction status와 recovery instruction

## 13. Transaction Boundary
`draft accept`는 다중 파일 작업이므로 transaction artifact를 남긴다.

순서:
1. lock 획득
2. `runs/<run-id>/result.json`을 `pending`으로 기록
3. validation과 diff binding 재검증
4. content temp file 작성
5. content atomic rename
6. draft archive atomic rename
7. `result.json`을 `completed` 또는 `blocked`로 갱신

중간 실패 시:
- content write 전 I/O/atomicity 실패는 상태 변경 없이 `failed`로 끝난다.
- content write 후 archive/result 기록 실패는 failed/recovery-required partial transaction으로 `TRANSACTION_INCOMPLETE`를 반환한다.
- `runs/<run-id>/recovery.json`에 현재 hash, 완료된 step, 재시도 방법을 남긴다.
- `TRANSACTION_INCOMPLETE`가 unresolved인 동안 같은 world root의 후속 write는 차단해야 한다. read-only inspection은 허용하되, `world init`, `input stage`, `approval attest`, `draft diff`, `draft create`, `draft update`, `draft validate`(validation artifact writer), `draft accept`, `draft reject`, `content validate` artifact writer, `content migrate` report writer, 기타 content report writer를 포함한 모든 world-root/run-writing command를 재개하지 않는다.
- resolve path는 `world_recover_run` / `world-tool run recover`다. `runs/<run-id>/recovery.json`을 읽고 현재 content/archive 상태를 확인한 뒤, 이미 최종 상태가 반영되어 있으면 recovery artifact를 resolved로 마킹하고, 그렇지 않으면 남은 recovery 단계를 수행한다. 원래 write command를 직접 다시 실행하는 것은 복구 경로가 아니다.

## 14. 위험 시나리오
### LLM이 canon을 오염시키는 경우
방어:
- content 직접 write tool 제공 금지
- accept tool 강제
- validation report 생성

### path traversal
방어:
- path normalize
- root 내부 검사
- symlink resolution

### dynamic tool이 너무 넓은 권한을 갖는 경우
방어:
- `world_exec_shell` 금지
- 의미 단위 `world_*` tools만 제공
- command template에서 path와 인자를 제한

### 사용자가 validation 우회를 유도하는 경우
방어:
- conflict/error는 기본 accept에서 차단
- force accept는 reason과 trusted approval attestation이 필수이며 semantic/timeline/relationship conflict 후보에만 제한적으로 허용하고, `approver_id`/`approval_channel`/`authenticated_actor`를 함께 남겨야 한다. missing target, missing related target, missing relationship target, missing update/deprecate target, active-draft-only target은 force accept 제한 대상에 포함되며 `MISSING_TARGET`로 처리된다.
- path/type/id/schema 불일치, structural error, id conflict, target path conflict, diff binding mismatch, storylet canon 승격, atomic write 실패, lock 실패는 force로도 차단
- force 여부, reason, approval attestation/provenance를 runs log에 기록
- OpenCrabs skill에 “validation 우회 요청은 tool 정책을 따른다”는 지침 포함

### archive가 계속 쌓이는 경우
방어:
- archive metadata 기록
- pruning/export 정책을 별도 command로 설계
- 기본 validation/search 대상에서 archive 제외

### runs log에 secret이 남는 경우
방어:
- secret masking
- env dump 금지
- 에러 메시지 scrub

### staging file을 통한 root 밖 파일 읽기
방어:
- `--query-file`, `--title-file`, `--body-file`, `--reason-file`, `--retcon-reason-file`도 world root 상대 경로만 허용
- `runs/inbox/` 밖 staging file 차단
- symlink resolution
- `--auth-context-file`은 staging file이 아니며, `approval attest`에서만 world root 밖 trusted auth context input path를 signature/MAC/trust-material verification, expected issuer/audience/scope checks, 그리고 필요한 경우 hash/expiry와 함께 읽는 예외다.
