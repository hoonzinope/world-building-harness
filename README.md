# world-harness

Markdown 세계관 팩을 관리하고, 비공개 스토리 룸을 운영하며, draft/canon 변경을 `world-tool` CLI로 검증/승인/반영하는 Go 기반 하네스다.

현재 레포는 설계 문서만 있는 상태가 아니다. 아래 기능이 구현되어 있다.

- `world-tool` Go CLI: world registry, draft, validation, diff, approval, run, story recovery, server, admin, Telegram 명령
- 공개 읽기용 pack/wiki 웹 UI
- 로그인 사용자는 진행 가능하고, 비로그인 사용자는 읽기만 가능한 story room 웹 UI
- Telegram pack 검색, idea 저장, draft 생성, 명시적 Codex 요청
- `packs/lumen-federation/` 샘플/실사용 세계관 팩

## 빠른 실행

공개 pack/wiki UI:

```bash
docker compose up -d --build world-harness
curl -fsS http://127.0.0.1:8097/health
```

접속:

```text
http://127.0.0.1:8097/world/
```

story room UI:

```bash
cp .env.example .env
# 실제 관리자 계정이나 GM provider가 필요하면 .env를 수정한다.
docker compose up -d --build world-harness-story
curl -fsS http://127.0.0.1:8098/health
```

접속:

```text
http://127.0.0.1:8098/world-story/stories
```

비로그인 사용자는 story lobby/story room을 볼 수 있지만 선택 제출, 질문, 진행자 변경, 관리자 액션은 할 수 없다.

## CLI

Docker 이미지에는 `world-tool`이 들어간다. 로컬 개발에서는 아래처럼 실행한다.

```bash
go run ./cmd/world-tool --help
```

자주 쓰는 읽기 명령:

```bash
go run ./cmd/world-tool world list --json
go run ./cmd/world-tool world status --world lumen-federation --json
go run ./cmd/world-tool doc list --world lumen-federation --json
go run ./cmd/world-tool doc search --world lumen-federation --query lucera --json
go run ./cmd/world-tool draft list --world lumen-federation --json
```

쓰기 경로는 staged input, validation, diff, approval attestation, accept/reject 흐름을 사용한다. 자세한 CLI/JSON 계약은 [docs/commands.md](docs/commands.md)를 기준으로 본다.

## Docker 서비스

| 서비스 | 역할 | 호스트 URL |
| --- | --- | --- |
| `world-harness` | 공개 pack/wiki UI | `http://127.0.0.1:8097/world/` |
| `world-harness-story` | 인증 액션이 있는 story room UI | `http://127.0.0.1:8098/world-story/` |
| `world-harness-telegram` | Telegram bot, `telegram` profile로 실행 | HTTP 포트 없음 |

운영 명령:

```bash
docker compose ps
docker compose logs -f world-harness-story
docker compose up -d --build world-harness-story
docker compose --profile telegram up -d --build world-harness-telegram
```

Telegram은 `TELEGRAM_BOT_TOKEN`이 있어야 동작한다. 안전하게 쓰려면 `TELEGRAM_ALLOWED_CHAT_ID`도 설정한다.

## 레포 구조

```text
cmd/world-tool/                  CLI entrypoint
internal/harness/                얇은 public facade: Run(args)
internal/harness/auth/           사용자, 세션, CSRF, auth store
internal/harness/cli/            world-tool 명령 dispatcher와 CLI handler
internal/harness/core/           공통 envelope, hash, id, file helper
internal/harness/server/         HTTP server, route, handler, view model
internal/harness/story/          Story store, GM job, import, recovery, export
internal/harness/telegram/       Telegram bot command와 transport
internal/harness/ui/             HTML template, CSS, story room JS asset
internal/harness/world/          Pack/world/document/validation/run domain logic
packs/lumen-federation/          활성 세계관 팩
docs/                            제품/아키텍처/workflow/command 문서
schema/                          schema reference
opencrabs/                       OpenCrabs skill/tool 연동 자산
```

새 코드는 책임을 가진 패키지에 둔다. 서버 핸들러는 요청 해석, 권한 확인, 데이터 준비, 템플릿 호출에 집중한다. 스토리 런타임은 `story`, 문서/세계관 로직은 `world`, 저수준 공통 유틸은 `core`, 화면 템플릿과 스타일은 `ui`에 둔다.

## Pack 구조

활성 pack은 `packs/<pack-id>/` 아래에 둔다.

```text
packs/lumen-federation/
├── content/      # canon Markdown 문서
├── drafts/       # pending draft 문서
├── ideas/        # Telegram/plain idea inbox
├── raw/          # 보존된 원본 자료
├── resources/    # legacy/reference resource
├── runs/         # command run artifact와 recovery file
├── archive/      # accepted/rejected/deprecated draft
└── harness.yaml
```

`content/`는 canon source of truth다. 생성물이나 변경 후보는 먼저 `drafts/`에 들어가고, validation/diff/approval을 거쳐 accept될 때 canon에 반영된다.

## Story UI

story 서비스의 런타임 데이터는 Docker volume `world_harness_story_data`에 저장되고 컨테이너 내부 `/app/data`로 마운트된다.

주요 route:

```text
/world-story/login
/world-story/stories
/world-story/stories/new
/world-story/stories/<story-id>
```

지원 기능:

- 공개 읽기용 story lobby/story room
- 로그인 사용자의 선택/custom 입력 제출
- 활성 story 질문 제출
- 진행자 claim/release
- 관리자 story lifecycle control
- Hector story import, recovery, export flow

GM provider는 `WORLD_HARNESS_GM_PROVIDER`로 설정한다. 로컬 deterministic 테스트는 `mock`, 컨테이너에서 Codex home을 마운트해 실제 생성을 돌릴 때는 `codex_cli`를 사용한다. 별도 Hermes Agent gateway/API server를 story GM으로 쓰려면 `hermes_api`를 사용하고 `WORLD_HARNESS_HERMES_API_BASE_URL`, `WORLD_HARNESS_HERMES_API_KEY`, `WORLD_HARNESS_HERMES_MODEL`을 설정한다.

Docker Desktop에서 host의 Hermes API server에 붙이는 예:

```env
WORLD_HARNESS_GM_PROVIDER=hermes_api
WORLD_HARNESS_HERMES_API_BASE_URL=http://host.docker.internal:8642/v1
WORLD_HARNESS_HERMES_API_KEY=change-me-local-dev
WORLD_HARNESS_HERMES_MODEL=hermes-agent
```

## Telegram

지원 명령:

```text
/packs
/status [pack]
/search [pack] <query>
/ideas [pack]
/draft [pack] <type> <id> | <title> | <body>
/codex [pack] <request>
```

`/`로 시작하지 않는 일반 메시지는 idea로 저장되며 Codex를 실행하지 않는다. 모델 토큰을 사용해 draft/story 작업을 요청할 때만 `/codex`를 명시적으로 쓴다.

## 개발 검증

커밋 전 기본 검증:

```bash
go test -count=1 ./...
go build ./...
git diff --check
```

frontend/story UI를 바꾼 경우 story 컨테이너도 다시 확인한다.

```bash
docker compose up -d --build world-harness-story
curl -fsS http://127.0.0.1:8098/health
```

## 참고 문서

- [AGENTS.md](AGENTS.md): 에이전트 작업 절차와 완료 기준
- [docs/prd.md](docs/prd.md): 제품 요구사항
- [docs/system-design.md](docs/system-design.md): 시스템 설계
- [docs/architecture.md](docs/architecture.md): 아키텍처 메모
- [docs/workflow.md](docs/workflow.md): draft/approval workflow
- [docs/commands.md](docs/commands.md): CLI command와 JSON contract
- [docs/schema.md](docs/schema.md): Markdown frontmatter schema
- [docs/validation-rules.md](docs/validation-rules.md): validation rule
- [docs/security-boundary.md](docs/security-boundary.md): 보안 경계
- [docs/operations.md](docs/operations.md): 운영 메모
