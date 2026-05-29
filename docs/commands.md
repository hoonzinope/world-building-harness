# commands.md

# world-tool Commands

## 1. 개요
`world-tool`은 OpenCrabs dynamic tools가 호출하는 Go 단일 바이너리다. OpenCrabs가 판단과 대화를 담당하고, `world-tool`은 파일/설정 변경을 안전하게 수행한다.

모든 명령은 `--json` 출력을 지원해야 한다. OpenCrabs dynamic tool은 stdout JSON만 해석한다.

## 2. 기본 구조
```bash
world-tool <resource> <action> [args] [flags]
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

`--world`와 `--root` 중 하나는 필수다. OpenCrabs 운영에서는 `--world`를 기본으로 사용하고, 개발/테스트에서는 `--root`를 사용할 수 있다.

## 3. world
```bash
world-tool world list --json
world-tool world status --world ashen-continent --json
world-tool world init --root ./worlds/ashen-continent --json
```

동작:
- world registry 조회
- world root 기본 구조 생성
- content/drafts/runs/archive 상태 요약

## 4. doc
```bash
world-tool doc list --world ashen-continent --scope content --json
world-tool doc read --world ashen-continent --path content/nations/north.md --json
world-tool doc search --world ashen-continent --query "북부 왕국" --json
```

동작:
- content/drafts 문서 목록
- path boundary 검사 후 문서 읽기
- title, tag, id, full-text 기반 검색

## 5. draft
```bash
world-tool draft create \
  --world ashen-continent \
  --type nation \
  --title "북부 제국" \
  --body-file ./tmp/body.md \
  --json

world-tool draft update --world ashen-continent --draft drafts/nations/northern-empire.md --body-file ./tmp/body.md --json
world-tool draft list --world ashen-continent --json
world-tool draft read --world ashen-continent --draft drafts/nations/northern-empire.md --json
```

동작:
- draft markdown 생성 또는 갱신
- frontmatter 보정
- source run 기록
- content는 변경하지 않음

## 6. validate
```bash
world-tool validate draft --world ashen-continent --draft drafts/nations/northern-empire.md --json
world-tool validate content --world ashen-continent --json
```

상태:
- pass
- warning
- conflict
- error

## 7. diff
```bash
world-tool diff draft --world ashen-continent --draft drafts/nations/northern-empire.md --json
```

동작:
- accept 시 content에 반영될 변경사항 계산
- `diff.patch` artifact 생성
- target path 충돌 검사

## 8. accept
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

OpenCrabs dynamic tool에서는 공백과 quoting 문제를 피하기 위해 `--reason`보다 `--reason-file` 사용을 권장한다.

## 9. reject
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

## 10. run
```bash
world-tool run list --world ashen-continent --json
world-tool run get --world ashen-continent --run-id 20260530-001 --json
```

동작:
- 최근 실행 목록
- run artifact 조회
- validation/diff/result 요약

## 11. OpenCrabs Dynamic Tool 매핑
`~/.opencrabs/tools.toml`에는 의미 단위 tool을 등록한다.

```toml
[[tools]]
name = "world_status"
description = "Return world status and pending draft summary"
executor = "shell"
command = "world-tool world status --world {{world_id}} --json"

[[tools]]
name = "world_read_doc"
description = "Read a world document within the selected world root"
executor = "shell"
command = "world-tool doc read --world {{world_id}} --path {{path}} --json"

[[tools]]
name = "world_validate_draft"
description = "Validate a draft against canon"
executor = "shell"
command = "world-tool validate draft --world {{world_id}} --draft {{draft_path}} --json"

[[tools]]
name = "world_accept_draft"
description = "Promote a validated draft into canon after explicit user approval"
executor = "shell"
command = "world-tool accept draft --world {{world_id}} --draft {{draft_path}} --reason-file {{reason_file}} --json"
```

긴 markdown body나 사용자 입력 reason은 command template 인자로 직접 넣지 말고 temp file 또는 stdin 기반 command로 넘긴다.
