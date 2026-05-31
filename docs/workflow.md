# workflow.md

# OpenCrabs World-Building Workflow

## 1. 원칙
OpenCrabs가 하네스와 오케스트레이터 역할을 한다. 판단과 생성은 OpenCrabs의 Codex OAuth provider가 수행하고, 실제 파일 변경은 OpenCrabs dynamic tool이 호출하는 `world-tool` Go CLI가 수행한다. Codex CLI provider는 fallback이다.

이 문서는 구현 전의 목표 workflow를 설명하는 설계 기준이다.

`content/` Markdown은 canon source of truth다. 생성물은 `drafts/`에 먼저 저장되고, validation과 사용자 승인을 거친 뒤에만 `content/`로 승격된다.

## 2. 책임 분리
```text
OpenCrabs
  - 사용자 대화
  - Codex OAuth provider를 통한 판단/생성
  - skill 실행
  - dynamic tool 호출

world-building skill
  - 작업 순서와 안전 규칙 안내
  - 어떤 tool을 언제 써야 하는지 지시
  - accept 전 사용자 승인 요구

world-tool
  - 파일 읽기/쓰기
  - validation
  - diff
  - accept/reject
  - runs/audit log
```

## 3. 공통 실행 단계
```text
receive user request in OpenCrabs
→ world-building skill rules apply
→ inspect world state with world_* tools
→ stage long query/title/body with world_stage_input, and stage reason/retcon_reason when that branch needs it
→ OpenCrabs/Codex drafts candidate content
→ call world_create_draft
→ call world_validate_draft
→ return summary and available actions
```

## 4. Draft 생성 Workflow
목적: 자연어 요청을 기반으로 새로운 세계관 설정 draft를 생성한다.

### 단계
1. `world_status`로 world root 상태 확인
2. 긴 검색 query는 `world_stage_input`으로 staging
3. `world_search_docs(scope=active)` 또는 `world_read_doc`으로 관련 canon 로딩
4. OpenCrabs/Codex가 draft markdown 후보 생성
5. title와 body를 각각 `world_stage_input`으로 staging
6. create용 명시적 `--id`를 정한 뒤 `world_create_draft`를 호출해 `draft_path`를 받는다. update/deprecate는 명시적 `--target-id`를 사용한다.
7. `world_validate_draft` 호출
8. validation summary와 id/`draft_path` 반환

### 정책
- OpenCrabs/Codex는 `content/`에 직접 쓰지 않는다.
- query/title/body/reason/retcon_reason은 command argv에 넣지 않고 `world_stage_input`으로 `runs/inbox/`에 staging한다.
- tool output의 id, `draft_path`, `run_id`, `validation_status`를 사용자에게 보여준다.

## 5. Validate Workflow
목적: draft가 schema와 canon 규칙을 만족하는지 검사한다.

### 단계
1. `world_validate_draft` 호출
2. frontmatter, id, type, required fields 검사
3. timeline, relationship, terminology 검사
4. validation report 저장
5. pass/warning/conflict/error 반환

### 출력
- validation status
- issues
- recommended fixes
- available actions

## 6. Accept Workflow
목적: 사용자가 승인한 draft를 canon content로 승격한다.

### 단계
1. OpenCrabs가 사용자에게 명시적 승인 확인
2. `world_diff_draft`로 변경 내용 표시
3. 사용자에게 diff summary를 확인받는다
4. reason을 `world_stage_input`으로 staging
5. `world_create_approval_attestation`으로 trusted auth context input과 diff/reason hash binding을 staging한다. approval attestation은 `runs/inbox/*-approval-attestation.json`에 저장되는 별도 staged artifact이며, `world_id`, `allowed_actions`/scope, `issuer`, `audience`, `authenticated_actor`, `approval_channel`, `issued_at`, `expires_at`, `downstream_action`과 hash/expiry 검증을 함께 만족해야 한다. production auth context input은 wrapper-signed 또는 MACed envelope여야 하고 configured wrapper trust material으로 검증되며 expected issuer/audience/scope policy를 만족해야 한다. local fixture mode는 test-only이고 explicit opt-in이 필요하다. `world_create_approval_attestation`은 downstream action으로 `world_accept_draft` 또는 `world_force_accept_draft` 중 정확히 하나를 받아 staged attestation에 저장해야 하며, auth context input의 exact downstream action과 accept/force는 그 값과 exact match여야 한다. exact downstream action이 없거나 mismatch면 `AUTH_CONTEXT_SCOPE_DENIED`, staged attestation이 later accept/force binding과 mismatch면 `APPROVAL_ATTESTATION_BINDING_MISMATCH`다. 세부 계약은 `docs/commands.md`를 따른다. `auth_context_file`/`auth_context_hash`는 world root 밖에서 생성된 신뢰 가능한 입력이며, prompt/model/staged files는 신뢰하지 않는다.
6. `world_accept_draft`에 `diff_run_id`, `draft_hash`, `target_base_hash`, `patch_hash`, `reason_file`, `reason_hash`, `approval_attestation_file`, `approval_attestation_hash`, `approver_id`, `approval_channel`, `authenticated_actor`를 함께 넘긴다. `world_force_accept_draft`도 같은 staged approval attestation의 downstream_action과 exact match여야 한다.
7. tool 내부에서 diff binding, approval attestation, validation 재실행을 검증
8. validation/policy/precondition/domain blocked result이면 blocked로 중단
9. content 생성 또는 갱신
10. accepted draft를 `archive/accepted/`로 이동
11. runs log와 result JSON 저장

### 정책
- accept는 tool이 강제하는 deterministic workflow다.
- warning은 accept를 차단하지 않지만 reason에 확인 맥락을 남긴다.
- blocked는 validation/policy/precondition/domain stop을 의미하며, failed와 구분한다.
- `TRANSACTION_INCOMPLETE`는 failed/recovery-required partial transaction이며, normal blocked no-mutation 결과가 아니다.
- `PATH_*`, `LOCK_BUSY`, I/O/path/lock 오류는 failed JSON error다.
- conflict/error와 `DRAFT_NOT_ACTIVE`, `DIFF_BINDING_REQUIRED`, `MISSING_TARGET` 같은 domain blocked 결과는 기본 accept에서 blocked로 반환된다. `MISSING_TARGET`는 related/relationship target이 content에 없거나 active draft에만 존재할 때도 사용한다.
- unresolved recovery가 있으면 `world init`, `input stage`, `approval attest`, `draft create`, `draft update`, `draft validate`(`runs/<run-id>/validation.json` writer), `draft diff`, `draft accept`, `draft reject`, `content validate` artifact writer, `content migrate` report writer, 기타 content report writer를 포함한 모든 world-root/run-writing command가 차단되며, `world_recover_run`만 write 예외다. read-only inspection은 허용한다.
- `force`는 reason과 trusted approval attestation이 둘 다 있어야 하며, 하나라도 없으면 blocked가 아니라 failed다. reason 누락은 `INVALID_ARGUMENT`, auth context 문제는 approval attestation 생성 경로에서만 대응하는 `AUTH_CONTEXT_*` failed code로 처리하고, semantic/timeline/relationship conflict 후보만 제한적으로 우회한다. auth context input의 exact downstream action이 없거나 `--downstream-action`과 mismatch면 `AUTH_CONTEXT_SCOPE_DENIED`, accept-time attestation hash/payload mismatch와 downstream_action mismatch는 `APPROVAL_ATTESTATION_*` failed code로 처리한다.
- missing target, missing related target, missing relationship target, missing update/deprecate target, active-draft-only target, path/type/id/schema 불일치, structural error, id conflict, target path conflict, diff binding mismatch, storylet canon 승격, atomic write 실패, lock 실패는 force로도 우회할 수 없다.
- `force`는 `docs/commands.md`의 `data.approval` 전체 필드 집합이 함께 기록될 때만 승인 provenance가 완성되며, 핵심 바인딩 필드인 `approval_attestation_file/hash`, `reason_file/hash`, `downstream_action`과 attestation validity metadata까지 포함해야 한다. staged approval attestation의 embedded downstream_action은 `world_force_accept_draft`와 exact match여야 한다.
- `approval_channel` 예시는 `OpenCrabs-chat`을 사용하고, attestation, accept, audit 전반에서 byte-identical 값이어야 한다.
- `authenticated_actor`는 OpenCrabs 인증 세션에서만 가져오고, `approval_channel`도 같은 trusted wrapper provenance로만 가져온다. `world_stage_input`과 `world_create_approval_attestation`이 반환한 파일 경로와 hash만 후속 tool에 전달한다.
- attestation은 prompt/model output이 아니라 trusted OpenCrabs auth context input에서 만들어야 하며, 이 input은 `world_id`, `allowed_actions`/scope, `issuer`, `audience`, `issued_at`, `expires_at`, `authenticated_actor`, `approval_channel`, `downstream_action`을 함께 만족해야 한다. production input은 wrapper-signed 또는 MACed envelope여야 하고 configured wrapper trust material으로 검증되며 expected issuer/audience/scope policy를 만족해야 한다. 이 scope는 attestation minting과 exact downstream accept/force action을 함께 커버해야 하고, exact downstream action이 없거나 mismatch면 `AUTH_CONTEXT_SCOPE_DENIED`다. 세부 실패 코드는 `docs/commands.md`를 따른다. 없으면 accept 대신 `AUTH_CONTEXT_MISSING` 실패를 사용자에게 설명한다.
- accept는 world root lock을 잡고 validation을 재실행한다.
- accept는 사용자가 확인한 diff binding과 현재 draft/content hash가 일치할 때만 진행된다.
- accept 이후 OpenCrabs는 content/index/cache를 다시 읽거나 재색인할 수 있다.

## 7. Reject Workflow
목적: 사용자가 승인하지 않은 draft를 반려한다.

### 단계
1. 반려 사유를 `world_stage_input`으로 staging한다.
2. `world_reject_draft`를 `reason_file`, `reason_hash`와 함께 호출한다.
3. draft를 `archive/rejected/`로 이동한다.
4. runs log에 reason을 기록한다.

### 정책
- reject reason은 staging 후 `reason_file`/`reason_hash`로 binding해야 한다.
- `world_reject_draft`는 reason이 없는 반려를 허용하지 않는다.

## 8. Run Log
예시:

```text
runs/
└── 20260530-001/
    ├── run.json
    ├── request.json
    ├── tool-call.json
    ├── draft.md
    ├── validation.json
    ├── validation.md
    ├── diff.patch
    ├── events.jsonl
    ├── result.json
    └── recovery.json (optional)
```

accept/diff run은 target content의 before/after hash를 포함한다.
`recovery.json`은 transactional/recovery-capable run이거나 recovery가 필요한 경우에만 추가로 생성된다.

`events.jsonl` 예시:

```json
{"step":"create_draft","status":"completed","actor":"opencrabs","time":"2026-05-30T10:00:00+09:00"}
{"step":"validate_draft","status":"completed","validation_status":"warning"}
{"step":"accept_draft","status":"blocked","validation_status":"conflict","block_reason":"VALIDATION_BLOCKED","issues":[{"code":"TIMELINE_CONFLICT","rule":"VR-203","severity":"conflict","message":"timeline conflict blocks accept until the draft is updated"}],"approval_attestation_file":"runs/inbox/20260530-001-approval-attestation.json","approval_attestation_hash":"sha256:3f2a0c7d4b1e8a9f6c2d5b7e1a9c4d8f0b2e6c1a5d7f8e9c0b1a2d3e4f5a6b7","reason_file":"runs/inbox/20260530-001-reason.txt","reason_hash":"sha256:8a1d2c3e4f5a6b7c9d0e1f2a3b4c5d6e7f8091a2b3c4d5e6f7a8b9c0d1e2f3a4","downstream_action":"world_accept_draft","approver_id":"park.hana","approval_channel":"OpenCrabs-chat","authenticated_actor":"openid:codex-oauth:user-123"}
```

## 9. Skill 지침 요약
world-building skill은 다음을 강제하도록 지시한다.

- 모든 세계관 파일 작업은 `world_*` tools를 사용한다.
- `content/` 직접 수정 요청은 거절하고 draft workflow로 전환한다.
- validation 없이 accept하지 않는다.
- conflict/error 상태에서는 사용자에게 이유를 설명하고 수정안을 제안한다.
- tool output을 source of truth로 삼아 다음 행동을 결정한다.
