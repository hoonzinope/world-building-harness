# directory-structure.md

# Directory Structure

## 1. 레포 구조
이 레포는 OpenCrabs용 세계관 skill/tools bundle과 `world-tool` Go CLI를 담는다.

아래는 목표 구조다. 현재 레포는 문서 중심 상태이며 `cmd/`, `internal/`, `opencrabs/`, `schema/`, `examples/`는 구현 대상이다.

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
├── docs/
└── README.md
```

## 2. world root 구조
아래 구조는 하나의 세계관 root 예시다. OpenCrabs는 registry를 통해 여러 world root를 선택할 수 있다.

```text
world-root/
├── content/
│   ├── characters/
│   ├── nations/
│   ├── organizations/
│   ├── places/
│   ├── events/
│   ├── timeline/
│   ├── magic/
│   └── glossary/
├── drafts/
│   ├── characters/
│   ├── nations/
│   ├── organizations/
│   ├── places/
│   ├── events/
│   ├── timeline/
│   ├── magic/
│   ├── glossary/
│   └── storylets/
├── raw/
├── graph/
│   ├── nodes.json
│   ├── edges.json
│   └── orphan-report.json
├── schema/
├── runs/
│   ├── inbox/
│   └── .lock
├── archive/
│   ├── accepted/
│   ├── rejected/
│   └── deprecated/
└── harness.yaml
```

## 3. content/
확정된 canon 문서를 저장한다. `content/` Markdown은 canon source of truth다.

정책:
- content는 `world-tool draft accept`에서만 수정한다.
- LLM 생성 결과가 바로 content에 들어가면 안 된다.
- 모든 content 문서는 frontmatter id를 가져야 한다.
- content target path는 `schema.md`의 type directory와 id 기반 file name 규칙을 따른다.
- `type: storylet` 문서는 MVP에서 content 아래에 둘 수 없다.
- OpenCrabs DB, graph, search index는 content에서 재생성 가능해야 한다.

## 4. drafts/
생성 후보 문서를 저장한다.

정책:
- OpenCrabs/Codex 생성 결과는 기본적으로 drafts에 저장한다.
- draft는 canon이 아니다.
- draft는 validate와 accept를 거쳐야 content로 승격된다.
- non-storylet draft는 `schema.md`에 정의된 type별 draft directory 아래에 저장한다.
- glossary draft는 `drafts/glossary/` 아래에 저장한다.
- storylet draft는 `drafts/storylets/` 아래에만 저장한다.
- accept 이후 draft 원본은 archive/accepted/로 이동한다.

## 5. runs/
tool 실행 기록을 저장한다.

예시:

```text
runs/
├── inbox/
└── 20260530-001/
    ├── request.json
    ├── tool-call.json
    ├── draft.md
    ├── validation.json
    ├── validation.md
    ├── diff.patch
    ├── events.jsonl
    └── result.json
```

정책:
- 모든 write tool은 run id를 가진다.
- 재현 가능한 수준의 입력과 출력을 남긴다.
- secret과 환경변수는 저장하지 않는다.
- `runs/inbox/`는 dynamic tool이 긴 query/title/body/reason/retcon_reason과 approval attestation을 world root 내부에 staging하는 임시 입력 위치다.
- `runs/inbox/` 파일은 `world-tool input stage`와 `world-tool approval attest`만 생성한다.
- `run get/list`는 `runs/inbox/`를 노출하지 않으며, unresolved `recovery.json`의 repair는 `world-tool run recover`만 수행한다.
- `runs/.lock` 또는 동등한 lock은 write command 동시 실행을 막기 위해 사용한다.
- accept/diff artifact는 target content의 before/after hash를 남긴다.
- transaction이 중간 실패하면 `recovery.json`을 남긴다.

## 6. archive/
승인, 반려, 폐기된 draft를 보관한다.

정책:
- archive/accepted/는 승인 당시 draft 원본을 보존한다.
- archive/rejected/는 반려된 draft와 반려 사유를 보존한다.
- archive/deprecated/는 더 이상 쓰지 않는 draft나 이전 canon 후보를 보존한다.
- archive 아래 문서는 기본 context loading, active validation, id 중복 검사 대상에서 제외한다.

## 7. opencrabs/
OpenCrabs에 설치할 자산을 담는다.

```text
opencrabs/
├── skills/world-building/SKILL.md
└── tools/world-tools.toml
```

설치 대상:

```text
~/.opencrabs/skills/world-building/SKILL.md
~/.opencrabs/tools.toml
```

## 8. harness.yaml
world root 내부 설정 파일이다.

`--root`로만 world root를 여는 실행에서는 이 파일이 `world_id` provenance를 복원하는 기준이 된다. mount path 자체를 world_id로 쓰지 않는다.
`--root` 실행 시에는 command site에서 `--world-id`를 명시하거나, 첫 tool 호출 전에 `harness.yaml`을 읽어 provenance를 확인해야 한다.

예시:

```yaml
schema_version: world-harness.v1
world_id: ashen-continent
world_root: .
content_dir: content
draft_dir: drafts
run_dir: runs
inbox_dir: runs/inbox
graph_dir: graph
archive_dir: archive
approval:
  require_accept: true
  allow_force_accept: true
security:
  deny_outside_root: true
  allow_network: false
locking:
  enabled: true
  lock_file: runs/.lock
```
