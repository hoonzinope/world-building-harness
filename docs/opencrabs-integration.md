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

canonical dynamic tool:

아래 예시는 OpenCrabs shell executor가 template 값을 argv-safe하게 escape한다고 가정한다. raw string interpolation만 지원한다면 그대로 사용하지 말고, request JSON file 하나를 받는 wrapper command를 둔다.

```toml
[[tools]]
name = "world_list"
description = "List configured worlds"
executor = "shell"
command = "world-tool world list --json"

[[tools]]
name = "world_status"
description = "Return world status and pending draft summary"
executor = "shell"
command = "world-tool world status --world {{world_id}} --json"

[[tools]]
name = "world_search_docs"
description = "Search canon and active draft documents"
executor = "shell"
command = "world-tool doc search --world {{world_id}} --query-file {{query_file}} --json"

[[tools]]
name = "world_read_doc"
description = "Read a world document within the selected world root"
executor = "shell"
command = "world-tool doc read --world {{world_id}} --path {{path}} --json"

[[tools]]
name = "world_create_draft"
description = "Create a draft without modifying canon content"
executor = "shell"
command = "world-tool draft create --world {{world_id}} --type {{type}} --title-file {{title_file}} --body-file {{body_file}} --json"

[[tools]]
name = "world_update_draft"
description = "Update an active draft without modifying canon content"
executor = "shell"
command = "world-tool draft update --world {{world_id}} --draft {{draft_path}} --body-file {{body_file}} --json"

[[tools]]
name = "world_read_draft"
description = "Read an active draft"
executor = "shell"
command = "world-tool draft read --world {{world_id}} --draft {{draft_path}} --json"

[[tools]]
name = "world_validate_draft"
description = "Validate a draft against canon"
executor = "shell"
command = "world-tool validate draft --world {{world_id}} --draft {{draft_path}} --json"

[[tools]]
name = "world_diff_draft"
description = "Return the content changes that accept would apply"
executor = "shell"
command = "world-tool diff draft --world {{world_id}} --draft {{draft_path}} --json"

[[tools]]
name = "world_accept_draft"
description = "Promote a validated draft into canon after explicit user approval"
executor = "shell"
command = "world-tool accept draft --world {{world_id}} --draft {{draft_path}} --reason-file {{reason_file}} --json"

[[tools]]
name = "world_reject_draft"
description = "Archive a draft as rejected with a reason"
executor = "shell"
command = "world-tool reject draft --world {{world_id}} --draft {{draft_path}} --reason-file {{reason_file}} --json"

[[tools]]
name = "world_get_run"
description = "Read run artifacts and result summary"
executor = "shell"
command = "world-tool run get --world {{world_id}} --run-id {{run_id}} --json"
```

긴 markdown body, 검색 query, title, reason은 command template에 직접 넣지 않는다. OpenCrabs executor가 stdin을 지원하면 stdin 기반 flag를 사용하고, 그렇지 않으면 world root 내부 `runs/inbox/`에 staging한 상대 경로를 `query_file`, `title_file`, `body_file`, `reason_file`로 넘긴다.

template 변수는 OpenCrabs가 넣더라도 신뢰하지 않는다. `world-tool`은 `world_id`, `type`, `path`, `draft_path`, `query_file`, `title_file`, `body_file`, `reason_file`, `run_id`를 다시 검증한다.

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

권장 registry 위치:

```text
~/.opencrabs/worlds.yaml
```

대체 위치:

```text
~/.config/world-tool/worlds.yaml
```

해석 우선순위:
1. command flag `--registry <path>`가 있으면 해당 파일
2. 환경변수 `WORLD_TOOL_REGISTRY`
3. `~/.opencrabs/worlds.yaml`
4. `~/.config/world-tool/worlds.yaml`

registry path 자체는 world root 안에 둘 필요가 없다. 단, registry가 가리키는 root는 symlink 해석 후 absolute path로 고정하고, 이후 모든 파일 접근은 그 root 내부로 제한한다.

`--world`와 `--root`가 동시에 지정되면 실패한다. `world list`는 registry만 읽고 world root를 열지 않는다.

world id 규칙:
- 영문 소문자, 숫자, 하이픈만 허용한다.
- path separator, whitespace, shell metacharacter는 금지한다.
- registry 안에서 중복 id는 config error다.

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

stdout에 JSON이 있으면 `schema_version`, `ok`, `status`, `error.code`를 우선 사용한다. stdout이 비어 있거나 JSON parse가 실패하면 dynamic tool 자체 실패로 처리한다.

### validation conflict
OpenCrabs는 accept를 강행하지 않고 conflict 이유와 수정안을 사용자에게 보여준다.

사용자가 강행을 요청하면 OpenCrabs는 `--force`가 허용되는 conflict인지 tool 결과를 기준으로 판단한다. `--force` reason은 반드시 별도 파일 또는 stdin으로 넘기고 runs log에 남긴다.

### path violation
`world-tool`은 world root 밖 경로 접근을 error로 반환한다.

## 10. 설계 원칙
- OpenCrabs가 하네스다.
- 이 레포는 OpenCrabs skill/tools bundle이다.
- `world-tool`은 Go 단일 바이너리다.
- OpenCrabs는 판단을 하고, `world-tool`은 파일 상태 변경을 강제한다.
- Codex SDK는 직접 내장하지 않는다.
