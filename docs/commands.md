# commands.md

# world-tool Commands

## 1. 개요
`world-tool`은 OpenCrabs dynamic tools가 호출하는 Go 단일 바이너리다. OpenCrabs가 판단과 대화를 담당하고, `world-tool`은 파일/설정 변경을 안전하게 수행한다.

모든 명령은 `--json` 출력을 지원해야 한다. OpenCrabs dynamic tool은 stdout JSON만 해석한다.

## 2. 기본 구조
```bash
world-tool [global flags] <resource> <action> [args] [command flags]
```

공통 플래그:
```bash
--world <id>            OpenCrabs/world registry의 world id
--root <path>           직접 world root를 지정할 때 사용
--json                  machine-readable JSON output
--run-id <id>           기존 run에 후속 event/artifact 추가
--dry-run               파일 변경 없이 diff와 계획만 출력
--verbose               상세 로그 출력
```

`world list`를 제외하면 `--world`와 `--root` 중 정확히 하나가 필수다. 둘 다 지정하면 실패한다.

운영에서는 `--world`를 기본으로 사용한다. `world-tool`은 registry에서 world id를 root로 해석한다. 개발/테스트에서는 `--root`를 사용할 수 있다.

`--root`는 symlink를 해석한 absolute path로 normalize한 뒤 `harness.yaml`이 있는 world root로 취급한다. 모든 문서 path 인자는 world root 기준 상대 경로만 허용한다.

Canonical command name:
- `world-tool reject draft`가 표준이다.
- `world-tool draft reject`는 문서와 dynamic tool에서 사용하지 않는다.

## 3. JSON 출력 계약
모든 command는 `--json` 모드에서 stdout에 JSON 하나만 출력한다. 로그, progress, debug text는 stdout에 쓰지 않는다. 필요하면 stderr에만 쓴다.

공통 envelope:

```json
{
  "schema_version": "world-tool.v1",
  "ok": true,
  "status": "created",
  "command": "draft.create",
  "world_id": "ashen-continent",
  "root": "/workspace/world",
  "run_id": "20260530-001",
  "data": {},
  "issues": [],
  "available_actions": []
}
```

실패 envelope:

```json
{
  "schema_version": "world-tool.v1",
  "ok": false,
  "status": "error",
  "command": "doc.read",
  "world_id": "ashen-continent",
  "run_id": "20260530-002",
  "error": {
    "code": "PATH_OUTSIDE_ROOT",
    "message": "path is outside selected world root",
    "details": {
      "path": "../../.ssh/id_rsa"
    }
  },
  "issues": [],
  "available_actions": []
}
```

`issues[]` 항목 형식:

```json
{
  "rule": "VR-101",
  "severity": "conflict",
  "message": "id nation_northern_empire already exists in content",
  "path": "drafts/nations/northern-empire.md",
  "field": "id",
  "recommendation": "use a new id or update the existing canon document through retcon workflow"
}
```

Exit code 정책:

| Exit code | 의미 |
| --- | --- |
| 0 | command가 완료되었고 stdout JSON이 authoritative result다. validation `warning/conflict/error`와 accept `blocked`도 expected domain result면 0을 사용할 수 있다. |
| 2 | CLI argument, registry, config, path boundary 오류 |
| 3 | 파일 I/O, lock 획득 실패, atomic write 실패 |
| 4 | 내부 오류 또는 panic 복구 |

OpenCrabs는 stdout JSON의 `ok`, `status`, `issues`, `available_actions`를 우선 해석한다. non-zero exit이면 tool 실행 실패로 표시하되, stdout에 JSON이 있으면 그 내용을 사용자 설명에 사용한다.

대표 error code:
- `INVALID_ARGUMENT`
- `REGISTRY_NOT_FOUND`
- `WORLD_NOT_FOUND`
- `PATH_OUTSIDE_ROOT`
- `PATH_NOT_MARKDOWN`
- `DRAFT_NOT_ACTIVE`
- `TARGET_PATH_CONFLICT`
- `STORYLET_NOT_CANON_TARGET`
- `VALIDATION_BLOCKED`
- `LOCK_BUSY`
- `IO_ERROR`
- `INTERNAL_ERROR`

## 4. world
```bash
world-tool world list --json
world-tool world status --world ashen-continent --json
world-tool world init --root ./worlds/ashen-continent --json
```

동작:
- world registry 조회
- world root 기본 구조 생성
- content/drafts/runs/archive 상태 요약

## 5. doc
```bash
world-tool doc list --world ashen-continent --scope content --json
world-tool doc read --world ashen-continent --path content/nations/north.md --json
world-tool doc search --world ashen-continent --query "북부 왕국" --json
world-tool doc search --world ashen-continent --query-file runs/inbox/query.txt --json
```

동작:
- content/drafts 문서 목록
- path boundary 검사 후 문서 읽기
- title, tag, id, full-text 기반 검색

`--scope` 허용값:
- `content`
- `drafts`
- `active`

`active`는 `content/`와 active `drafts/`를 포함하고 `archive/`를 제외한다.

## 6. draft
```bash
world-tool draft create \
  --world ashen-continent \
  --type nation \
  --title "북부 제국" \
  --body-file runs/inbox/body.md \
  --json

world-tool draft create \
  --world ashen-continent \
  --type nation \
  --title-file runs/inbox/title.txt \
  --body-stdin \
  --json

world-tool draft update --world ashen-continent --draft drafts/nations/northern-empire.md --body-file runs/inbox/body.md --json
world-tool draft list --world ashen-continent --json
world-tool draft read --world ashen-continent --draft drafts/nations/northern-empire.md --json
```

동작:
- draft markdown 생성 또는 갱신
- frontmatter 보정
- source run 기록
- content는 변경하지 않음

`--body-file`, `--title-file`, `--query-file`은 world root 기준 상대 경로만 허용한다. OpenCrabs dynamic tool에서 자유 텍스트를 넘겨야 한다면 `runs/inbox/` 아래에 staging하고, executor가 stdin을 지원하면 stdin 기반 flag를 우선한다.

## 7. validate
```bash
world-tool validate draft --world ashen-continent --draft drafts/nations/northern-empire.md --json
world-tool validate content --world ashen-continent --json
```

상태:
- pass
- warning
- conflict
- error

validation command는 검증을 정상 완료했다면 status가 `conflict`나 `error`여도 exit code 0을 반환할 수 있다. 문서 파싱 실패 같은 world content 문제는 validation result이며, CLI/config/path 오류와 구분한다.

## 8. diff
```bash
world-tool diff draft --world ashen-continent --draft drafts/nations/northern-empire.md --json
```

동작:
- accept 시 content에 반영될 변경사항 계산
- `diff.patch` artifact 생성
- target path 충돌 검사

diff result는 accept 대상 content path, before/after hash, patch summary를 포함해야 한다.

## 9. accept
```bash
world-tool accept draft \
  --world ashen-continent \
  --draft drafts/nations/northern-empire.md \
  --reason "사용자 승인" \
  --json

world-tool accept draft \
  --world ashen-continent \
  --draft drafts/nations/northern-empire.md \
  --force \
  --reason "의도적인 retcon" \
  --json
```

동작:
- validation 재실행
- conflict/error 시 기본 중단
- draft를 content로 승격
- content frontmatter status를 canon으로 갱신
- accepted draft를 archive/accepted/로 이동
- runs log와 result JSON 기록

Accept 정책:
- `pass`는 사용자 승인 reason이 있으면 accept 가능하다.
- `warning`은 기본 accept 가능하지만, warning을 확인했다는 reason을 runs log에 남긴다.
- `conflict`와 `error`는 기본 accept에서 `status: "blocked"`로 반환한다.
- `--force`는 `conflict`를 우회할 수 있지만 reason이 필수다.
- `--force`도 structural `error`, path violation, target path conflict, inactive draft는 우회할 수 없다.
- `type: storylet`은 MVP에서 기본 accept 대상이 아니며 `STORYLET_NOT_CANON_TARGET`으로 차단한다.

OpenCrabs dynamic tool에서는 공백과 quoting 문제를 피하기 위해 `--reason`보다 `--reason-file` 사용을 권장한다.

Dynamic tool용 예시:

```bash
world-tool accept draft \
  --world ashen-continent \
  --draft drafts/nations/northern-empire.md \
  --reason-file runs/inbox/reason.txt \
  --json
```

`--reason-file`도 world root 기준 상대 경로만 허용한다.

동시성 정책:
- accept는 world root lock을 잡고 validation을 재실행한다.
- diff 시점의 target content hash와 accept 시점의 hash가 다르면 accept를 중단한다.
- lock을 획득하지 못하면 `LOCK_BUSY`를 반환한다.

## 10. reject
```bash
world-tool reject draft \
  --world ashen-continent \
  --draft drafts/nations/northern-empire.md \
  --reason "기존 제국 설정과 중복" \
  --json
```

동작:
- draft를 archive/rejected/로 이동
- runs log에 반려 사유 기록

## 11. run
```bash
world-tool run list --world ashen-continent --json
world-tool run get --world ashen-continent --run-id 20260530-001 --json
```

동작:
- 최근 실행 목록
- run artifact 조회
- validation/diff/result 요약

## 12. OpenCrabs Dynamic Tool 매핑
`~/.opencrabs/tools.toml`에는 의미 단위 tool을 등록한다.

아래 예시는 shell executor가 template 값을 argv-safe하게 escape한다고 가정한다. raw string interpolation만 지원한다면 이 command 문자열을 그대로 쓰지 말고, OpenCrabs가 생성한 JSON request file 하나만 wrapper에 넘기는 방식으로 바꾼다.

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

긴 markdown body, 검색 query, title, 사용자 입력 reason은 command template 인자로 직접 넣지 말고 `runs/inbox/`에 staging한 file 또는 stdin 기반 command로 넘긴다.

모든 template 변수는 OpenCrabs 쪽에서 primitive string으로 전달하되, `world-tool`이 다시 검증한다. `world_id`, `type`, `path`, `draft_path`, `query_file`, `title_file`, `body_file`, `reason_file`, `run_id`는 allowlist/relative path 검사를 통과해야 한다.
