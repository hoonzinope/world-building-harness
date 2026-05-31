# prd.md

# OpenCrabs World-Building Tools PRD

## 1. 제품 정의
이 레포는 별도 world-harness 제품을 만들기 위한 레포가 아니다. OpenCrabs를 세계관 빌딩 하네스이자 오케스트레이터로 사용하기 위한 skill, dynamic tools, 그리고 deterministic Go CLI(`world-tool`)에 대한 설계 기준을 제공한다.

OpenCrabs는 Codex OAuth provider를 기본 provider로 사용해 판단과 생성을 수행한다. 이 레포의 역할은 OpenCrabs가 세계관 파일과 설정을 안전하게 관리하도록 규칙과 도구의 설계 기준을 제공하는 것이다.

## 2. 목표
- OpenCrabs 대화에서 세계관 설정을 생성, 검증, 승인, 반려할 수 있게 한다.
- Codex OAuth provider를 기본 인증/모델 경로로 사용하고, Codex CLI provider는 fallback으로 둔다.
- `content/` Markdown을 canon source of truth로 유지한다.
- 생성 결과는 먼저 `drafts/`에 저장하고, 명시적 승인 후에만 `content/`로 승격한다. `force accept`는 오퍼레이터가 명시적으로 승인한 예외 경로일 뿐, 일반 guardrail 우회가 아니다.
- 세계관 작업은 OpenCrabs dynamic tools가 호출하는 `world-tool` Go 바이너리로 수행한다.
- 긴 query/title/body/reason/retcon_reason은 `world_stage_input`으로 staging한 뒤 후속 tool에 file path와 hash를 넘긴다.
- draft create는 명시적 `--id`를 입력으로 받고, draft update/deprecate는 명시적 `--target-id`를 사용한다. 사용자는 create에서 나온 id나 update/deprecate의 target_id에서 파생된 `draft_path`를 기준으로 validate/diff/accept 흐름을 따라간다.
- 사용자가 승인한 diff와 accept 실행은 diff_run_id와 hash binding으로 묶는다.
- 승인 provenance는 `approver_id`, `approval_channel`, `authenticated_actor`, approval attestation file/hash를 포함해야 하며, `authenticated_actor`와 attestation은 OpenCrabs 인증 세션 또는 provider identity에서만 가져오고, `auth_context_file`/`auth_context_hash`는 world root 밖에서 생성된 trusted wrapper input이어야 한다.
- tool 출력은 JSON으로 안정화해 OpenCrabs/Codex가 다음 행동을 판단할 수 있게 한다.
- 여러 world root를 OpenCrabs 설정이나 registry로 관리한다.

## 3. 비목표
- 별도 world-harness 서버나 agent runtime을 만들지 않는다.
- Codex SDK를 직접 내장하지 않는다.
- OpenCrabs 자체를 fork하거나 대체하지 않는다.
- LLM에게 `content/` 직접 수정 권한을 주지 않는다.
- MVP에서 웹 대시보드, 그래프 시각화, 멀티 에이전트 자동 병렬 실행은 제외한다.

## 4. 핵심 사용자 시나리오
### Draft 생성
사용자가 OpenCrabs에서 “북부 제국 설정을 만들어줘”라고 요청하면, world-building skill이 작업 규칙을 주입한다. OpenCrabs/Codex는 관련 canon 문서를 읽고 draft markdown을 구성한 뒤 `world_create_draft` tool을 호출한다.

### Validate
OpenCrabs는 `world_validate_draft` tool을 호출해 draft가 schema와 canon 규칙을 만족하는지 검사한다. validation 결과는 구조화 JSON과 report로 남는다.

### Accept
사용자가 승인하면 OpenCrabs는 먼저 `world_diff_draft` 결과의 diff_run_id, draft_hash, target_base_hash, patch_hash를 확인하고, reason을 staging한 뒤 `world_create_approval_attestation`으로 auth context와 diff/reason binding을 묶는다. 그 다음 `world_accept_draft` tool을 호출한다. tool은 diff binding과 validation을 재검증하고, conflict나 policy stop이 없을 때만 `content/`로 승격한 뒤 accepted draft를 archive한다. operator-approved force path가 필요한 경우에도 approval attestation/provenance는 유지되어야 하며, `approval_channel`은 attest/accept/audit 전반에서 byte-identical이어야 한다.

## 5. MVP 기능
1. OpenCrabs skill: 세계관 작업 원칙과 workflow 지침 제공
2. Dynamic tools: OpenCrabs에서 호출 가능한 `world_*` tool 세트
3. Go `world-tool` CLI: 파일/설정/검증/승격 작업을 deterministic하게 수행
4. `input stage`, `draft create/validate/diff/accept/reject`, `registry add/list` 명령 계약
5. `content/`, `drafts/`, `runs/`, `archive/` 기반 world root 구조
6. JSON 출력과 audit/run log

## 6. 성공 기준
- OpenCrabs 대화에서 draft 생성부터 validation까지 수행할 수 있다.
- `content/`는 `world_accept_draft` 이전에 변경되지 않는다.
- 모든 write 작업은 `runs/`에 입력, 결과, validation, diff, hash binding을 남긴다.
- accepted draft는 `archive/accepted/`로 이동하고 active validation 대상에서 제외된다.
- validation/policy/precondition/domain stop은 blocked로 보고하고, failed와 구분한다. `TRANSACTION_INCOMPLETE`는 failed/recovery-required partial transaction으로 별도 취급한다.
- OpenCrabs skill만으로 판단 흐름을 안내하되, 실제 파일 변경은 `world-tool`이 강제한다.

## 7. 핵심 원칙
- OpenCrabs가 하네스이자 오케스트레이터다.
- 이 레포는 OpenCrabs용 세계관 skill + tools bundle이다.
- skill은 작업 규칙을 알려주고, tool은 규칙을 강제한다.
- `content/` Markdown은 canon source of truth다.
- OpenCrabs DB/index/cache는 보조 상태이며 content에서 재생성 가능해야 한다.
- LLM 출력은 후보이며 tool validation을 통과해야 한다.
- shell template은 argv-safe execution 또는 request-file/wrapper semantics를 사용해야 하며 raw interpolation을 허용하지 않는다.
- 범용 shell 실행보다 의미 단위의 `world_*` tools를 우선한다.
