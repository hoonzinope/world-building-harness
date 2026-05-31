# architecture.md

# OpenCrabs World-Building Architecture

## 1. 개요
최종 구조는 OpenCrabs를 세계관 빌딩 하네스로 사용하고, 이 레포가 OpenCrabs에 설치할 skill, dynamic tools, Go 기반 `world-tool` CLI를 제공하는 방식이다. 현재 레포는 구현 전 문서 기준 상태다.

OpenCrabs가 Codex OAuth provider를 통해 판단과 생성을 수행하고, `world-tool`이 파일 시스템 변경과 validation을 deterministic하게 처리한다. Codex CLI provider는 OAuth provider를 사용할 수 없는 환경의 fallback이다.

구체적인 컴포넌트 다이어그램과 sequence는 [system-design.md](system-design.md)를 기준으로 한다.

## 2. 전체 구조
```text
User
  ↓
OpenCrabs TUI / Telegram / Discord / Slack
  ↓
OpenCrabs Codex OAuth provider
  ↓
world-building Skill
  ↓
OpenCrabs Dynamic Tools
  ↓
world-tool Go CLI
  ↓
World Root: content / drafts / runs / archive / graph
```

## 3. 레이어 구분
### OpenCrabs Layer
대화, provider 선택, tool 호출, 채널 응답, 승인 UX를 담당한다.

- Codex OAuth provider 기본 사용
- Codex CLI provider fallback
- `/skills`로 world-building skill 실행
- `~/.opencrabs/tools.toml` dynamic tools 로딩
- 사용자 승인과 대화 흐름 관리

### Skill Layer
`opencrabs/skills/world-building/SKILL.md`가 OpenCrabs/Codex에게 세계관 작업 규칙을 제공한다.

- `content/` 직접 수정 금지
- draft 우선 생성
- validation 후 accept
- conflict 시 사용자 확인
- tool 출력 기반으로 다음 행동 결정

### Tool Layer
`opencrabs/tools/world-tools.toml`이 OpenCrabs dynamic tools를 정의한다. 각 tool은 shell executor로 `world-tool`을 호출한다.

dynamic tool은 `opencrab_exec_shell` 같은 범용 명령이 아니라 의미 단위 작업이어야 한다.

- `world_list`
- `world_status`
- `world_stage_input`
- `world_search_docs`
- `world_read_doc`
- `world_create_draft`
- `world_create_update_draft`
- `world_create_deprecate_draft`
- `world_update_draft`
- `world_read_draft`
- `world_validate_draft`
- `world_diff_draft`
- `world_accept_draft`
- `world_force_accept_draft`
- `world_reject_draft`
- `world_get_run`

### world-tool Layer
Go 단일 바이너리다. OpenCrabs와 독립적으로 실행 가능해야 하며, 모든 출력은 `--json`을 지원한다.

- world registry 해석
- path boundary 검사
- markdown/frontmatter 파싱
- draft/content/archive 파일 조작
- validation
- diff 생성
- runs/audit log 작성

## 4. 추천 레포 구조
아래는 목표 구조다. 현재 레포는 문서 중심 상태이며 구현 산출물은 roadmap/implementation-plan의 순서로 추가한다.

```text
world-harness/
├── cmd/
│   └── world-tool/
│       └── main.go
├── internal/
│   ├── world/
│   ├── docs/
│   ├── drafts/
│   ├── validate/
│   ├── diff/
│   ├── audit/
│   └── config/
├── opencrabs/
│   ├── skills/
│   │   └── world-building/
│   │       └── SKILL.md
│   └── tools/
│       └── world-tools.toml
├── schema/
│   ├── world-doc.schema.json
│   ├── relationship-types.yaml
│   └── document-types.yaml
├── examples/
│   └── worlds/
└── docs/
```

## 5. 데이터 저장소
### World Root
각 세계관은 독립적인 root를 가진다.

```text
world-root/
├── content/
├── drafts/
├── runs/
├── archive/
├── graph/
├── schema/
└── harness.yaml
```

### content/
canon source of truth다. OpenCrabs DB나 index가 손실되어도 content Markdown에서 canon을 복구할 수 있어야 한다.

### drafts/
pending 후보 설정이다. OpenCrabs/Codex가 생성한 설정은 먼저 drafts에 저장된다.

### runs/
모든 write workflow의 입력, 결과, validation, diff, actor, timestamp를 기록한다.

### archive/
accepted/rejected/deprecated draft를 보관한다. archive는 active validation과 id 중복 검사 기본 대상에서 제외한다.

## 6. 권한 경계
- OpenCrabs는 world 작업에 `world_*` tools를 사용한다.
- `world-tool`은 선택된 world root 밖을 읽거나 쓰지 않는다.
- `content/`는 `world_accept_draft`에서만 변경된다.
- `draft accept`는 diff binding과 validation을 재실행한다.
- `force accept`는 reason과 approval provenance가 필수이며, semantic/timeline/relationship conflict 후보에만 제한적으로 허용하고 runs log에 남긴다.
- Docker 사용 시 job container에는 선택된 world root 하나만 마운트한다.
- OpenCrabs credential/config volume과 world root volume은 분리한다.
