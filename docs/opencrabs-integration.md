# opencrabs-integration.md

# OpenCrabs Integration

## 1. 역할 정의
OpenCrabs는 이 구조에서 하네스이자 오케스트레이터다. 사용자는 OpenCrabs와 대화하고, OpenCrabs는 Codex OAuth provider를 통해 판단/생성을 수행하며, 세계관 파일 작업은 dynamic tools가 호출하는 `world-tool`이 수행한다. Codex CLI provider는 fallback으로만 사용한다.

이 레포는 OpenCrabs를 확장하기 위한 다음 자산을 제공한다.

- `opencrabs/skills/world-building/SKILL.md`
- `opencrabs/tools/world-tools.toml`
- `world-tool` Go 바이너리
- schema와 validation 규칙

## 2. OpenCrabs가 책임지는 것
- 사용자 대화와 승인 확인
- Codex OAuth provider 기본 사용
- Codex CLI provider fallback
- world-building skill 실행
- dynamic tool 호출
- tool 결과를 바탕으로 다음 행동 판단
- Telegram/Discord/Slack 등 채널 응답

## 3. world-tool이 책임지는 것
- world root path boundary 검사
- content/drafts/archive/runs 파일 조작
- frontmatter 정규화
- validation
- diff 생성
- accept/reject 정책 강제
- JSON 결과 반환

## 4. Skill 설치
OpenCrabs user skill은 다음 위치에 둔다.

```text
~/.opencrabs/skills/world-building/SKILL.md
```

레포 내부 소스 위치:

```text
opencrabs/skills/world-building/SKILL.md
```

skill은 다음을 지시한다.

- 세계관 작업에는 `world_*` tools를 사용한다.
- `content/` 직접 수정은 하지 않는다.
- draft 생성 후 validation을 수행한다.
- accept 전에는 사용자 승인을 확인한다.
- tool output을 authoritative state로 취급한다.

## 5. Dynamic Tools 설치
OpenCrabs dynamic tools는 다음 위치에 정의한다.

```text
~/.opencrabs/tools.toml
```

레포 내부 소스 위치:

```text
opencrabs/tools/world-tools.toml
```

대표 tool:

```toml
[[tools]]
name = "world_status"
description = "Return world status and pending draft summary"
executor = "shell"
command = "world-tool world status --world {{world_id}} --json"

[[tools]]
name = "world_create_draft"
description = "Create a draft without modifying canon content"
executor = "shell"
command = "world-tool draft create --world {{world_id}} --type {{type}} --title {{title}} --body-file {{body_file}} --json"

[[tools]]
name = "world_accept_draft"
description = "Promote a validated draft into canon after explicit user approval"
executor = "shell"
command = "world-tool accept draft --world {{world_id}} --draft {{draft_path}} --reason-file {{reason_file}} --json"
```

## 6. World Registry
OpenCrabs나 `world-tool`은 world id를 world root로 해석해야 한다.

예시:

```yaml
worlds:
  ashen-continent:
    title: 잿빛 대륙
    root: /host/worlds/ashen-continent
  glass-sea:
    title: 유리해
    root: /host/worlds/glass-sea
```

registry는 OpenCrabs 설정, 별도 world registry 파일, 또는 `world-tool` config로 관리할 수 있다. canon source of truth는 registry가 아니라 각 world root의 `content/`다.

## 7. 대화 플로우
```text
사용자: /world-building 북부 제국 설정 만들어줘
OpenCrabs: skill rules 적용
OpenCrabs/Codex: 관련 canon 조회를 위해 world_search_docs 호출
OpenCrabs/Codex: draft markdown 생성
OpenCrabs: world_create_draft 호출
OpenCrabs: world_validate_draft 호출
OpenCrabs: draft path, validation status, 다음 행동 제안
사용자: 승인해
OpenCrabs: world_diff_draft 호출 후 확인
OpenCrabs: world_accept_draft 호출
world-tool: validation 재실행 후 content 승격
```

## 8. Docker 운영
권장 운영 방식은 OpenCrabs와 `world-tool`을 같은 이미지에 넣고, OpenCrabs credential/config volume과 world root volume을 분리해 마운트하는 것이다. Codex OAuth provider를 사용하므로 컨테이너 안에 Codex CLI를 설치하거나 `~/.codex`를 마운트하는 것은 기본 요구사항이 아니다.

```bash
docker run --rm \
  -v opencrabs-config:/home/opencrabs/.opencrabs \
  -v /host/worlds/ashen-continent:/workspace/world \
  opencrabs-world:latest
```

강한 격리가 필요하면 OpenCrabs 자체는 host에서 실행하되, dynamic tool command가 per-world container에서 `world-tool`을 실행하도록 구성할 수 있다.

```bash
docker run --rm \
  --user 1000:1000 \
  --network none \
  --read-only \
  --tmpfs /tmp \
  -v /host/worlds/ashen-continent:/workspace/world \
  world-tool:latest \
  world-tool validate draft --root /workspace/world --draft drafts/nations/northern-empire.md --json
```

개발 단계에서는 host에 설치된 `world-tool`을 직접 호출해도 된다.

Codex CLI provider fallback을 사용할 때만 컨테이너에 `codex` CLI와 별도 Codex auth volume이 필요하다. 이 경우에도 Codex auth volume은 `/workspace/world` 안에 두지 않는다.

## 9. 실패 처리
### malformed JSON
OpenCrabs는 raw stdout을 사용자에게 그대로 보여주지 않고, tool 실패와 stderr 요약을 제공한다.

### validation conflict
OpenCrabs는 accept를 강행하지 않고 conflict 이유와 수정안을 사용자에게 보여준다.

### path violation
`world-tool`은 world root 밖 경로 접근을 error로 반환한다.

## 10. 설계 원칙
- OpenCrabs가 하네스다.
- 이 레포는 OpenCrabs skill/tools bundle이다.
- `world-tool`은 Go 단일 바이너리다.
- OpenCrabs는 판단을 하고, `world-tool`은 파일 상태 변경을 강제한다.
- Codex SDK는 직접 내장하지 않는다.
