# roadmap.md

# World-Building Harness Roadmap

## 1. 방향
로드맵은 거대한 멀티 에이전트 시스템을 바로 만드는 것이 아니라, content Markdown을 canon source of truth로 고정한 안전한 단일 workflow에서 시작해 OpenCrab 연동, Codex SDK runner, 승인 플로우, 그래프 인덱스, 멀티 에이전트로 점진 확장하는 방식으로 진행한다.

## 2. Phase 0: 문서화와 결정 고정
### 목표
제품 경계와 MVP 범위를 확정한다.

### 산출물
- prd.md
- architecture.md
- workflow.md
- commands.md
- directory-structure.md
- schema.md
- validation-rules.md
- opencrab-integration.md
- security-boundary.md
- roadmap.md

### 완료 기준
- OpenCrab은 host/backend + adapter라는 결정이 문서화됨
- world-harness가 core라는 결정이 문서화됨
- draft와 content 경계가 문서화됨
- accept 전 content 변경 금지가 문서화됨
- content Markdown이 canon source of truth라는 결정이 문서화됨
- accepted draft는 archive 후 active set에서 제외된다는 결정이 문서화됨

## 3. Phase 1: Local CLI Core MVP
### 목표
OpenCrab 없이 로컬 CLI만으로 생성, 검증, 승격 흐름을 완성한다.

### 기능
- `world init`
- `world genesis`
- `world validate`
- `world accept`
- `world status`

### 산출물
- 기본 디렉토리 생성
- harness.yaml
- draft 생성
- validation report 생성
- runs log 생성

### 완료 기준
- 자연어 요청으로 draft markdown이 생성됨
- content는 accept 전까지 변경되지 않음
- validate 결과가 runs에 저장됨
- accept 시 content로 승격됨
- accept된 draft가 archive/accepted/로 이동하고 active validation에서 제외됨

## 4. Phase 2: Schema & Validator 강화
### 목표
LLM 생성물을 안정적으로 검사할 수 있는 rule 기반 validator를 만든다.

### 기능
- frontmatter 검사
- id 중복 검사
- required field 검사
- timeline 검사
- relationship 검사
- orphan related id 검사
- validation report 생성

### 완료 기준
- 명백한 구조 오류를 error로 탐지
- id 중복을 conflict로 탐지
- timeline 충돌 후보를 warning/conflict로 탐지

## 5. Phase 3: OpenCrab Host Integration
### 목표
OpenCrab에서 같은 배포 아티팩트의 world CLI를 호출하고 여러 world root를 관리할 수 있게 한다.

### 기능
- OpenCrab world registry
- argv subprocess 기반 `world` CLI 호출
- target world root만 마운트한 per-world job container 실행
- job status tracking
- draft summary / validation status 표시
- accept/reject 명령 연결
- accept 이후 content 재색인

### 완료 기준
- OpenCrab이 stdout JSON을 파싱해 사용자에게 요약을 반환함
- OpenCrab은 content를 직접 수정하지 않음
- world-harness CLI가 모든 canon 변경을 수행함
- harness job은 선택된 world root 밖 파일을 볼 수 없음
- OpenCrab DB/index가 비어도 content Markdown에서 복구 가능함

## 6. Phase 4: Codex SDK Runner
### 목표
Codex SDK를 기본 Agent Runner backend로 붙여 ChatGPT/Codex 구독 기반 개인용 하네스를 먼저 완성한다.

### 기능
- codex-sdk-runner
- codex-cli-runner fallback
- context manifest 전달
- draft generation job
- semantic validation candidate job
- timeout/retry/status handling
- Codex thread/job id 추적
- runs log에 runner metadata 기록

### 완료 기준
- Codex SDK runner로 draft를 생성할 수 있음
- Codex CLI runner fallback으로 동일 workflow를 실행할 수 있음
- Codex SDK runner가 content를 직접 수정하지 않음
- accept는 deterministic CLI workflow로만 수행됨
- runner 출력은 validator가 재파싱하고 검증함

## 7. Phase 5: Approval UX
### 목표
Telegram 또는 OpenCrab UI에서 생성 결과를 검토하고 승인/반려할 수 있게 한다.

### 기능
- draft 요약 보기
- validation warning 보기
- diff 보기
- accept/reject 명령 연결
- pending approval 목록

### 완료 기준
- 사용자가 원격에서 draft를 검토하고 승인할 수 있음
- conflict 상태에서는 승인 전에 경고가 표시됨

## 8. Phase 6: Graph Store
### 목표
content 기반 graph 인덱스를 생성한다.

### 기능
- nodes.json 생성
- edges.json 생성
- graph rebuild
- graph check
- orphan report

### 완료 기준
- content 전체를 기준으로 graph를 재생성할 수 있음
- graph가 원천 진실이 아니라 재생성 가능한 인덱스로 유지됨

## 9. Phase 7: Storylet & Exporter
### 목표
세계관 설정 생성 외에 사건 후보와 raw note 정리 기능을 추가한다.

### 기능
- `world storylet`
- `world export`
- raw note → schema markdown 변환
- storylet candidate 생성

### 완료 기준
- raw 메모를 draft로 정리 가능
- 기존 canon 기반 storylet 생성 가능

## 10. Phase 8: Multi-Agent Orchestration
### 목표
oh-my-codex식 workflow layer 위에 역할별 agent를 추가한다.

### 후보 역할
- Genesis Agent
- Canon Validator Agent
- Markdown Exporter Agent
- Graph Curator Agent
- Storylet Generator Agent

### 원칙
멀티 에이전트는 core workflow가 안정화된 이후에 붙인다. 처음부터 멀티 에이전트로 설계하면 복잡도가 불필요하게 증가한다.

## 11. Phase 9: Search & Semantic Context
### 목표
관련 문서 로딩 품질을 높인다.

### 기능
- tag 기반 검색
- graph 기반 검색
- full-text search
- optional vector search
- recent runs context

### 완료 기준
- genesis 시 관련 canon을 더 정확히 불러올 수 있음
- 불필요한 전체 문서 로딩을 줄임

## 12. Phase 10: Web Dashboard
### 목표
운영과 검토를 위한 UI를 제공한다.

### 기능
- drafts 목록
- validation status 목록
- runs explorer
- graph viewer
- accept/reject UI

### 완료 기준
- CLI를 몰라도 생성 결과를 검토할 수 있음

## 13. 우선순위
최우선:
1. Local CLI MVP
2. validation report
3. runs log
4. accept workflow
5. OpenCrab same-deployment integration
6. Codex SDK runner

나중:
1. OpenAI API SDK runner
2. multi-agent
3. graph visualization
4. web dashboard
5. semantic search
6. automatic retcon assistant

## 14. 리스크
### 범위 과대화
멀티 에이전트, Telegram, 웹 대시보드, graph 시각화를 처음부터 넣으면 실패 가능성이 커진다.

### canon 오염
생성 결과가 바로 content에 들어가면 세계관 일관성이 무너진다.

### OpenCrab 과의존
OpenCrab 내부에 도메인 로직을 넣으면 CLI, Codex, Claude에서 재사용하기 어려워진다.

### source of truth 혼선
OpenCrab DB나 graph를 canon 원본처럼 다루면 content Markdown과 불일치가 생긴다. MVP에서는 content Markdown을 우선하고 나머지는 재색인 가능한 보조 데이터로 둔다.

### validator 과신
validator는 확정 판정기가 아니라 충돌 후보 탐지기다.

## 15. 첫 구현 추천
첫 커밋의 목표는 작아야 한다.

```text
world init
world genesis "테스트 국가 설정 생성"
world validate drafts/nations/test.md
world accept drafts/nations/test.md
```

이 네 명령이 안정적으로 동작하면 그때 OpenCrab과 멀티 에이전트로 확장한다.
