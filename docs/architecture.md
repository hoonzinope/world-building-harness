# architecture.md

# World-Building Harness Architecture

## 1. 개요
World-Building Harness는 Markdown 세계관 root를 대상으로 동작하는 core workflow engine과 외부 입력 채널을 연결하는 adapter/host layer로 구성된다. `content/` Markdown은 canon source of truth이며, graph나 OpenCrab DB는 재생성 가능한 인덱스 또는 운영 상태 저장소로 취급한다.

OpenCrab은 사용자 입장에서는 세계관 빌딩 하네스처럼 보이지만, 구현 관점에서는 world-harness CLI를 호출하고 승인 UX와 job 관리를 제공하는 host/backend다. OpenCrab과 world-harness CLI는 같은 이미지나 배포 아티팩트에 포함될 수 있으나, core는 OpenCrab 내부 구현에 의존하지 않는다. 운영 환경에서는 OpenCrab이 target world root만 마운트한 job container를 실행하는 방식을 권장한다.

## 2. 전체 구조
```text
User
  ↓
CLI / OpenCrab / Codex / Claude / Web UI
  ↓
Adapter / Host Layer
  ↓
World Harness Core
  ↓
Workflow Runner / State Machine
  ↓
Context Loader / Agent Runner / Validator / Writer
  ↓
Markdown World Root + Graph Store + Runs Log
```

OpenCrab과 world CLI를 같은 이미지로 배포하는 경우의 물리 구조:

```text
opencrab server
  ↓ starts job
world-harness job container
├── world CLI
├── world-harness library/core
└── /workspace/world  # only one target world root is mounted
```

## 3. 레이어 구분
### Adapter Layer
외부 입력을 world-harness 명령으로 변환한다. OpenCrab은 adapter이면서 동시에 world registry, job queue, approval UX를 제공하는 host/backend 역할을 한다.
- CLI adapter
- OpenCrab adapter
- Telegram adapter
- Codex/Claude command adapter
- future Web UI adapter

Adapter는 파일 구조, canon 규칙, validation 규칙을 직접 구현하지 않는다. OpenCrab은 여러 world root의 경로와 사용자 권한을 관리할 수 있지만, content write 가능 여부는 world-harness core가 최종 판단한다.

### Core Layer
세계관 생성, 검증, 승인 흐름을 책임지는 deterministic workflow layer다.
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
- Archive service

### Infrastructure Services
파일 시스템, LLM 호출, git, graph 저장소 같은 외부 의존성을 감싼다.
- file repository
- markdown parser
- frontmatter parser
- LLM provider
- Codex SDK runner
- git client
- graph storage

## 4. 핵심 컴포넌트
### Command Router
`world genesis`, `world validate`, `world accept`, `world status` 같은 명령을 해석해 적절한 workflow를 호출한다.

### Workflow Runner
코드 또는 설정에 정의된 step 목록을 순서대로 실행한다. 각 step은 입력과 출력을 명확히 가진다. MVP에서는 코드 기반 state machine으로 시작하고, 필요해지면 YAML workflow 정의를 추가할 수 있다.

### Context Loader
요청과 관련된 content, drafts, graph, 최근 runs를 로딩한다. 관련 문서 검색은 파일 경로, 태그, id, graph 관계를 기반으로 시작하고, 이후 semantic search를 붙일 수 있다.

### Agent Runner
OpenAI API, Codex SDK, Claude API, Gemini API, Codex CLI, Claude Code, OpenCode 같은 실행 엔진을 추상화한다. Core는 특정 LLM에 종속되지 않는다.

Agent Runner는 draft 생성, context 요약, semantic validation 같은 비결정적 step만 수행한다. content 승격, archive 이동, graph 확정 업데이트, git commit은 Agent Runner가 직접 수행하지 않는다.

### Validator
LLM 결과를 canon으로 믿지 않고 충돌 후보를 탐지한다. validator 결과는 확정 판정이 아니라 사람이 검토할 근거다.

### Writer
draft 작성, validation report 작성, graph candidate 작성, content 승격, accepted draft archive를 담당한다. content 변경은 accept workflow에서만 허용된다.

### Run Logger
각 실행마다 runs/{run_id}/ 아래에 입력, 컨텍스트, 계획, 결과, 검증 결과, diff를 저장한다.

## 5. 데이터 저장소
### Markdown World Root
사람이 읽고 수정하는 원본 저장소다. 각 world root는 독립적인 content, drafts, runs, graph, schema를 가진다.

### content/
canon source of truth다. OpenCrab DB나 graph가 손실되어도 content Markdown에서 canon을 복구할 수 있어야 한다.

### Graph Store
content 문서에서 추출한 entity와 relationship의 보조 인덱스다. canon의 원천 진실은 Markdown이며, graph는 재생성 가능해야 한다.

### Runs Log
하네스 실행 기록이다. 디버깅, 회귀 검증, 이전 판단 추적에 사용한다.

### OpenCrab Backend Store
world registry, 사용자 권한, job 상태, approval 상태, search index/cache를 저장한다. canon source of truth가 아니며, content Markdown을 기준으로 재색인할 수 있어야 한다.

## 6. 권한 경계
- core는 선택된 world root 밖 파일을 읽거나 쓰지 않는다.
- content/는 accept 단계에서만 수정된다.
- drafts/와 runs/는 생성 단계에서 수정 가능하다.
- archive/accepted/는 승인된 draft 원본 보관소이며 기본 context loading, id 중복 검사, validation 대상에서 제외된다.
- graph/는 accept 또는 graph rebuild 단계에서 갱신된다.
- OpenCrab은 직접 content를 수정하지 않고 world CLI를 argv subprocess로 호출한다.
- 강한 격리가 필요한 운영 환경에서는 CLI 실행 컨테이너에 선택된 world root 하나만 `/workspace/world`로 마운트한다.

## 7. 확장 방향
MVP 이후 multi-agent runner, semantic index, Telegram approval UI, graph visualization, automatic regression validation을 추가할 수 있다.
