# architecture.md

# World-Building Harness Architecture

## 1. 개요
World-Building Harness는 세계관 저장소를 직접 소유하는 core workflow engine과 외부 입력 채널을 연결하는 adapter들로 구성된다. OpenCrab은 adapter 중 하나이며, core는 OpenCrab에 의존하지 않는다.

## 2. 전체 구조
```text
User
  ↓
CLI / OpenCrab / Codex / Claude / Web UI
  ↓
Adapter Layer
  ↓
World Harness Core
  ↓
Workflow Runner
  ↓
Context Loader / Agent Runner / Validator / Writer
  ↓
Markdown Repo + Graph Store + Runs Log
```

## 3. 레이어 구분
### Adapter Layer
외부 입력을 world-harness 명령으로 변환한다.
- CLI adapter
- OpenCrab adapter
- Telegram adapter
- Codex/Claude command adapter
- future Web UI adapter

Adapter는 파일 구조, canon 규칙, validation 규칙을 직접 구현하지 않는다.

### Core Layer
세계관 생성, 검증, 승인 흐름을 책임진다.
- command router
- workflow runner
- state manager
- permission guard
- run logger

### Domain Services
세계관 도메인에 특화된 작업을 수행한다.
- Genesis service
- Canon validation service
- Storylet generation service
- Markdown export service
- Graph update service

### Infrastructure Services
파일 시스템, LLM 호출, git, graph 저장소 같은 외부 의존성을 감싼다.
- file repository
- markdown parser
- frontmatter parser
- LLM provider
- git client
- graph storage

## 4. 핵심 컴포넌트
### Command Router
`world genesis`, `world validate`, `world accept`, `world status` 같은 명령을 해석해 적절한 workflow를 호출한다.

### Workflow Runner
YAML에 정의된 step 목록을 순서대로 실행한다. 각 step은 입력과 출력을 명확히 가진다.

### Context Loader
요청과 관련된 content, drafts, graph, 최근 runs를 로딩한다. 관련 문서 검색은 파일 경로, 태그, id, graph 관계를 기반으로 시작하고, 이후 semantic search를 붙일 수 있다.

### Agent Runner
OpenAI API, Claude API, Gemini API, Codex CLI, Claude Code, OpenCode 같은 실행 엔진을 추상화한다. Core는 특정 LLM에 종속되지 않는다.

### Validator
LLM 결과를 canon으로 믿지 않고 충돌 후보를 탐지한다. validator 결과는 확정 판정이 아니라 사람이 검토할 근거다.

### Writer
draft 작성, validation report 작성, graph candidate 작성, content 승격을 담당한다. content 변경은 accept workflow에서만 허용된다.

### Run Logger
각 실행마다 runs/{run_id}/ 아래에 입력, 컨텍스트, 계획, 결과, 검증 결과, diff를 저장한다.

## 5. 데이터 저장소
### Markdown Repo
사람이 읽고 수정하는 원본 저장소다.

### Graph Store
content 문서에서 추출한 entity와 relationship의 보조 인덱스다. canon의 원천 진실은 Markdown이며, graph는 재생성 가능해야 한다.

### Runs Log
하네스 실행 기록이다. 디버깅, 회귀 검증, 이전 판단 추적에 사용한다.

## 6. 권한 경계
- core는 world root 밖 파일을 읽거나 쓰지 않는다.
- content/는 accept 단계에서만 수정된다.
- drafts/와 runs/는 생성 단계에서 수정 가능하다.
- graph/는 accept 또는 graph rebuild 단계에서 갱신된다.
- OpenCrab은 직접 content를 수정하지 않는다.

## 7. 확장 방향
MVP 이후 multi-agent runner, semantic index, Telegram approval UI, graph visualization, automatic regression validation을 추가할 수 있다.
