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
→ stage long query/title/body/reason/retcon_reason with world_stage_input
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
3. `world_search_docs` 또는 `world_read_doc`으로 관련 canon 로딩
4. OpenCrabs/Codex가 draft markdown 후보 생성
5. title와 body를 각각 `world_stage_input`으로 staging
6. `world_create_draft` 호출
7. `world_validate_draft` 호출
8. validation summary와 draft path 반환

### 정책
- OpenCrabs/Codex는 `content/`에 직접 쓰지 않는다.
- query/title/body/reason/retcon_reason은 command argv에 넣지 않고 `world_stage_input`으로 `runs/inbox/`에 staging한다.
- tool output의 `draft_id`, `run_id`, `validation_status`를 사용자에게 보여준다.

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
5. `world_accept_draft`에 `diff_run_id`, `draft_hash`, `target_base_hash`, `patch_hash`, `reason_file`, `approver_id`, `approval_channel`, `authenticated_actor`를 함께 넘긴다
6. tool 내부에서 diff binding 검증과 validation 재실행
7. policy/validation stop이면 blocked로 중단
8. content 생성 또는 갱신
9. accepted draft를 `archive/accepted/`로 이동
10. runs log와 result JSON 저장

### 정책
- accept는 tool이 강제하는 deterministic workflow다.
- warning은 accept를 차단하지 않지만 reason에 확인 맥락을 남긴다.
- blocked는 validation/policy stop을 의미하며, failed와 구분한다.
- conflict/error는 기본 accept에서 blocked로 반환된다.
- `force`는 reason이 없으면 blocked이며, semantic/timeline/relationship conflict 후보만 제한적으로 우회한다.
- structural error, id conflict, path/target 충돌, storylet canon 승격, diff binding mismatch는 force로도 우회할 수 없다.
- `force`는 `approver_id`, `approval_channel`, `authenticated_actor`가 함께 기록될 때만 승인 provenance가 완성된다.
- accept는 world root lock을 잡고 validation을 재실행한다.
- accept는 사용자가 확인한 diff binding과 현재 draft/content hash가 일치할 때만 진행된다.
- accept 이후 OpenCrabs는 content/index/cache를 다시 읽거나 재색인할 수 있다.

## 7. Reject Workflow
목적: 사용자가 승인하지 않은 draft를 반려한다.

### 단계
1. 반려 사유 확인
2. `world_reject_draft` 호출
3. draft를 `archive/rejected/`로 이동
4. runs log에 reason 기록

## 8. Run Log
예시:

```text
runs/
└── 20260530-001/
    ├── request.json
    ├── tool-call.json
    ├── draft.md
    ├── validation.json
    ├── validation.md
    ├── diff.patch
    ├── events.jsonl
    └── result.json
```

accept/diff run은 target content의 before/after hash를 포함한다.

`events.jsonl` 예시:

```json
{"step":"create_draft","status":"completed","actor":"opencrabs","time":"2026-05-30T10:00:00+09:00"}
{"step":"validate_draft","status":"completed","validation_status":"warning"}
{"step":"accept_draft","status":"blocked","reason":"validation conflict","approver_id":"park.hana","approval_channel":"OpenCrabs chat","authenticated_actor":"openid:codex-oauth:user-123"}
```

## 9. Skill 지침 요약
world-building skill은 다음을 강제하도록 지시한다.

- 모든 세계관 파일 작업은 `world_*` tools를 사용한다.
- `content/` 직접 수정 요청은 거절하고 draft workflow로 전환한다.
- validation 없이 accept하지 않는다.
- conflict/error 상태에서는 사용자에게 이유를 설명하고 수정안을 제안한다.
- tool output을 source of truth로 삼아 다음 행동을 결정한다.
