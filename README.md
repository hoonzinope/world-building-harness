# world-building-harness

OpenCrabs를 세계관 빌딩 하네스이자 오케스트레이터로 사용하기 위한 문서/도구 설계 레포다.

현재 상태는 **구현 전 설계 기준**이다. 아직 Go CLI, OpenCrabs skill, dynamic tools, sample world root는 이 레포에 구현되어 있지 않다. 이 문서들은 그 구현을 위한 제품 경계와 명령 계약을 고정한다.

## 목표
- OpenCrabs 대화에서 세계관 설정을 draft로 생성한다.
- `world-tool` Go CLI가 validation, diff, accept/reject, audit log를 deterministic하게 수행한다.
- `content/` Markdown을 canon source of truth로 유지한다.
- `content/` 변경은 사용자 승인 이후 `world_accept_draft` 경로에서만 허용한다.
- 모든 tool output은 JSON 계약을 따른다.

## 문서 읽는 순서
1. [docs/prd.md](docs/prd.md): 제품 경계와 MVP 목표
2. [docs/system-design.md](docs/system-design.md): 전체 컴포넌트와 workflow
3. [docs/commands.md](docs/commands.md): `world-tool` CLI와 JSON 계약
4. [docs/schema.md](docs/schema.md): Markdown frontmatter와 relationship schema
5. [docs/validation-rules.md](docs/validation-rules.md): validator 규칙과 accept 차단 정책
6. [docs/security-boundary.md](docs/security-boundary.md): path, secret, Docker 보안 경계
7. [docs/implementation-plan.md](docs/implementation-plan.md): 구현 milestone과 완료 기준

## 구현 예정 산출물
```text
cmd/world-tool/                    Go CLI entrypoint
internal/{world,docs,drafts,...}/  deterministic implementation
opencrabs/skills/world-building/   OpenCrabs skill
opencrabs/tools/world-tools.toml   OpenCrabs dynamic tools
schema/                            machine-readable schemas
examples/worlds/                   sample world roots and fixtures
```

## MVP vertical slice
```text
world-tool world init
world-tool draft create
world-tool validate draft
world-tool diff draft
world-tool accept draft
```

## 현재 주의점
- 이 레포의 현재 파일은 문서뿐이다.
- `cmd/`, `internal/`, `opencrabs/`, `schema/`, `examples/`는 목표 구조다.
- 문서의 canonical command와 JSON 계약은 [docs/commands.md](docs/commands.md)를 우선한다.
- schema/validation 세부 계약은 [docs/schema.md](docs/schema.md)와 [docs/validation-rules.md](docs/validation-rules.md)를 우선한다.
