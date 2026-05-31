# world-building-harness

OpenCrabs를 세계관 빌딩 하네스이자 오케스트레이터로 사용하기 위한 문서/도구 설계 레포다.

이 문서 세트는 **구현 전 설계 기준**이다. 아직 Go CLI, OpenCrabs skill, dynamic tools, sample world root는 이 레포에 구현되어 있지 않다. 이 문서들은 그 구현을 위한 제품 경계와 명령 계약을 고정한다.

## 목표
- OpenCrabs 대화에서 세계관 설정을 draft로 생성한다.
- `world-tool` Go CLI가 validation, diff, accept/reject, audit log를 deterministic하게 수행한다.
- `content/` Markdown을 canon source of truth로 유지한다.
- `content/` 변경은 원칙적으로 사용자 승인 후 `world_accept_draft` 경로에서만 허용한다. `force accept`는 오퍼레이터가 명시적으로 승인한 예외 경로일 뿐이며, 정상 validation/policy guardrail을 약화시키는 일반 우회로가 아니다.
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
world-tool registry add
world-tool world list
world-tool input stage  # title
world-tool input stage  # body
world-tool draft create
world-tool draft validate
world-tool draft diff
world-tool input stage  # reason
world-tool approval attest
world-tool draft accept
```

## 구현 후 Quickstart 목표
아래는 CLI가 구현된 뒤 통과해야 하는 첫 성공 경로다.
`approval attest` 단계는 OpenCrabs trusted wrapper/session metadata로부터 생성된 `auth_context_file`/`auth_context_hash`를 사용하며, 이 파일은 world root 밖에서 만들어지고 prompt, model output, staged files는 신뢰하지 않는다.
`draft create`는 명시적 `--id`를 입력으로 받고, `draft update/deprecate`는 명시적 `--target-id`를 사용한다. create에서 나온 id로 파생된 `draft_path`를 기준으로 validate, diff, approval attestation, accept가 이어진다.

```bash
world-tool world init --root ./examples/worlds/ashen-continent --world-id ashen-continent --json
world-tool registry add --world ashen-continent --root ./examples/worlds/ashen-continent --title "잿빛 대륙" --json
world-tool world list --json
world-tool input stage --world ashen-continent --kind title --stdin --json
world-tool input stage --world ashen-continent --kind body --stdin --json
world-tool draft create --world ashen-continent --change-type create --type nation --id nation_ashen_empire --title-file runs/inbox/<title-file> --title-hash <hash> --body-file runs/inbox/<body-file> --body-hash <hash> --json
world-tool draft validate --world ashen-continent --draft drafts/nations/<id>.md --json
world-tool draft diff --world ashen-continent --draft drafts/nations/<id>.md --json
world-tool input stage --world ashen-continent --kind reason --stdin --json
world-tool approval attest --world ashen-continent --diff-run-id <run> --draft-hash <hash> --target-base-hash <hash> --patch-hash <hash> --approver-id <user> --approval-channel OpenCrabs-chat --authenticated-actor <actor> --auth-context-file <auth-context-file> --auth-context-hash <hash> --reason-hash <hash> --json
world-tool draft accept --world ashen-continent --draft drafts/nations/<id>.md --diff-run-id <run> --draft-hash <hash> --target-base-hash <hash> --patch-hash <hash> --approver-id <user> --approval-channel OpenCrabs-chat --approval-attestation-file runs/inbox/<approval-attestation>.json --approval-attestation-hash <hash> --authenticated-actor <actor> --reason-file runs/inbox/<reason-file> --reason-hash <hash> --json
```

## 현재 주의점
- 이 레포의 현재 파일은 문서뿐이다.
- `cmd/`, `internal/`, `opencrabs/`, `schema/`, `examples/`는 목표 구조다.
- 문서의 canonical command와 JSON 계약은 [docs/commands.md](docs/commands.md)를 우선한다.
- schema/validation 세부 계약은 [docs/schema.md](docs/schema.md)와 [docs/validation-rules.md](docs/validation-rules.md)를 우선한다.
