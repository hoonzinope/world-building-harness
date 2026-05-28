# prd.md

# World-Building Harness PRD

## 1. 제품 정의
World-Building Harness는 Markdown 기반 세계관 저장소를 대상으로 LLM을 이용해 설정을 생성, 검증, 수정, 승인하는 workflow + orchestration layer다.

이 제품은 세계관 웹 위키 자체가 아니며, 범용 AI 에이전트 플랫폼도 아니다. 핵심 목적은 canon 저장소를 오염시키지 않으면서 세계관 설정을 안전하게 확장하는 것이다.

## 2. 목표
- 자연어 요청을 기반으로 세계관 설정 초안을 생성한다.
- 기존 canon 문서와의 충돌 후보를 검출한다.
- draft와 content의 경계를 명확히 유지한다.
- 모든 실행 과정과 결과를 runs/에 남긴다.
- OpenCrab, CLI, Codex, Claude, Gemini 등 외부 실행 환경에서 동일한 workflow를 호출할 수 있게 한다.

## 3. 비목표
- 소설 본문 자동 집필 플랫폼을 만들지 않는다.
- OpenCrab 자체를 대체하지 않는다.
- LLM에게 canon 확정 권한을 주지 않는다.
- MVP에서 멀티 에이전트 자동 병렬 실행을 구현하지 않는다.
- MVP에서 웹 대시보드와 그래프 시각화는 제외한다.

## 4. 핵심 사용자 시나리오
### Genesis
사용자가 “북부 제국 설정을 만들어줘”라고 요청하면 하네스는 관련 문서를 읽고, 필요한 경우 질문을 생성하며, 답변을 바탕으로 draft markdown을 생성한다.

### Validate
생성된 draft가 기존 국가, 인물, 사건, 연표와 충돌하는지 검사하고 충돌 후보를 validation report로 남긴다.

### Accept
사용자가 승인한 draft만 content/로 승격한다. 이때 graph와 runs log도 갱신한다.

## 5. MVP 기능
1. `world genesis <request>`: 자연어 요청을 받아 draft 생성
2. `world validate <draft-path>`: draft와 canon 충돌 검사
3. `world accept <draft-path>`: 승인된 draft를 content로 승격
4. `world status`: draft, validation, 최근 runs 상태 확인
5. OpenCrab adapter: OpenCrab에서 위 CLI 명령을 호출 가능하게 함

## 6. 성공 기준
- 사용자는 자연어 요청 하나로 draft 문서를 생성할 수 있다.
- content는 accept 명령 이전에 변경되지 않는다.
- 모든 실행은 runs/에 request, context, result, validation, diff를 남긴다.
- validator는 최소한 frontmatter 누락, 중복 id, timeline 충돌, entity 관계 충돌 후보를 탐지한다.
- OpenCrab은 world-harness 내부 로직을 직접 갖지 않고 adapter로만 동작한다.

## 7. 핵심 원칙
- OpenCrab은 core가 아니라 adapter다.
- world-harness가 workflow의 주체다.
- LLM은 실행 엔진이지 최종 결정권자가 아니다.
- content는 canon이다.
- drafts는 후보 설정이다.
- accept 전까지 canon은 바뀌지 않는다.
- graph는 canon의 보조 인덱스다.
- runs/는 oh-my-codex의 .omx 같은 영속 실행 기록이다.
