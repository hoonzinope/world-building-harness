# system-design.md

# OpenCrabs World-Building System Design

## 1. 한 줄 정의
OpenCrabs가 하네스와 오케스트레이터 역할을 하고, 이 레포는 OpenCrabs가 세계관을 안전하게 관리하도록 `world-building` skill, dynamic tools, Go `world-tool` CLI를 제공한다.

이 문서는 구현 전의 목표 동작을 설명하는 설계 계약이다.

```text
판단과 대화: OpenCrabs + Codex OAuth provider
작업 규칙: world-building skill
상태 변경: world-tool Go CLI
원천 데이터: world root의 content/ Markdown
```

## 2. 전체 컴포넌트
```mermaid
flowchart TD
    U["User"] --> OC["OpenCrabs"]
    OC --> CP["Codex OAuth provider"]
    OC --> SK["world-building skill"]
    SK --> OC
    OC --> DT["Dynamic tools (world_*)"]
    DT --> WT["world-tool Go CLI"]
    WT --> WR["World root"]
    WR --> C["content/ canon"]
    WR --> D["drafts/ candidates"]
    WR --> R["runs/ audit log"]
    WR --> A["archive/ accepted/rejected"]
    WR --> G["graph/ index"]
```

## 3. 실행 책임
### OpenCrabs
- 사용자와 대화한다.
- Codex OAuth provider로 요청을 해석하고 draft 내용을 생성한다.
- `world-building` skill의 규칙을 따른다.
- `world_*` dynamic tools를 호출한다.
- tool output을 보고 다음 행동을 사용자에게 제안한다.

### world-building Skill
- `content/` 직접 수정 금지 같은 작업 원칙을 주입한다.
- draft → validate → diff → user approval → accept 순서를 안내한다.
- conflict/error 상태에서 accept하지 않도록 지시한다.
- 파일 작업은 반드시 `world_*` tools를 사용하게 한다.

### Dynamic Tools
- OpenCrabs가 호출할 수 있는 의미 단위 API다.
- shell executor로 `world-tool`을 호출한다.
- 범용 shell 실행을 열지 않는다.
- stdout JSON을 OpenCrabs가 해석한다.

### world-tool
- Go 단일 바이너리다.
- world root 밖 path를 차단한다.
- Markdown/frontmatter를 파싱하고 정규화한다.
- validation, diff, accept/reject, runs log, recovery handling을 수행한다.
- `content/` 변경은 accept command에서만 허용한다.

## 4. World Root
```mermaid
flowchart LR
    WR["world-root"] --> C["content/ canon source of truth"]
    WR --> D["drafts/ pending candidates"]
    WR --> R["runs/ audit artifacts"]
    WR --> A["archive/ accepted rejected deprecated"]
    WR --> S["schema/ document schemas"]
    WR --> G["graph/ rebuildable index"]
```

`content/`가 canon source of truth다. OpenCrabs DB, search index, graph는 보조 데이터이며 content에서 재생성 가능해야 한다.

`runs/inbox/`는 privileged transient staging area다. normal browse/search/list 대상이 아니며, `input stage`만 여기에 write한다.

canonical root binding:

| Concept | Meaning |
| --- | --- |
| `world_id` | logical registry key. Docker `--root` mode에서는 registry metadata나 `harness.yaml`에서 복원해야 하며 mount path 자체로 추론하지 않는다. |
| `registry_root` | registry에 저장된 canonical root. root-only execution이면 registry-backed resolution으로 결정된 canonical path다. |
| `root` | 실제 tool process가 접근하는 effective root |
| audit fields | `world_id`, `registry_root`, `root`, `run_id` |

native execution에서는 `registry_root == root`가 되어야 한다. Docker에서는 registry root와 effective root인 `root`가 달라질 수 있으므로, audit/result envelope가 둘을 구분해 기록해야 한다.

## 5. 주요 Tool 세트
| Tool | 내부 command | 역할 |
| --- | --- | --- |
| `world_list` | `world-tool world list` | registry에 등록된 world 목록 |
| `world_status` | `world-tool world status` | world 상태와 pending draft 요약 |
| `world_stage_input` | `world-tool input stage` | 긴 query/title/body/reason/retcon_reason을 runs/inbox에 staging하고 input path/hash를 반환 |
| `world_search_docs` | `world-tool doc search --scope active` | 관련 canon/draft 검색 |
| `world_read_doc` | `world-tool doc read` | 문서 읽기 |
| `world_create_draft` | `world-tool draft create` | canon 변경 없이 draft 생성 |
| `world_create_update_draft` | `world-tool draft create --change-type update` | 기존 canon 갱신/retcon draft 생성 |
| `world_create_deprecate_draft` | `world-tool draft create --change-type deprecate` | 기존 canon 폐기 draft 생성 |
| `world_update_draft` | `world-tool draft update` | draft 수정 |
| `world_read_draft` | `world-tool draft read` | active draft 읽기 |
| `world_validate_draft` | `world-tool draft validate` | schema/canon 검증 |
| `world_diff_draft` | `world-tool draft diff` | accept 예상 변경 확인 |
| `world_accept_draft` | `world-tool draft accept` | validation 후 content 승격, approval provenance required |
| `world_force_accept_draft` | `world-tool draft accept --force` | 오퍼레이터가 승인한 예외 경로, policy limits still apply |
| `world_reject_draft` | `world-tool draft reject` | draft 반려 |
| `world_get_run` | `world-tool run get` | run artifact 조회 |

## 6. Draft 생성 흐름
```mermaid
sequenceDiagram
    participant User
    participant OpenCrabs
    participant Skill
    participant Tool as Dynamic Tool
    participant WT as world-tool
    participant World as World Root

    User->>OpenCrabs: "북부 제국 설정 만들어줘"
    OpenCrabs->>Skill: world-building rules apply
    OpenCrabs->>Tool: world_status(world_id)
    Tool->>WT: world-tool world status --world ashen-continent --json
    WT->>World: read content/drafts/runs
    WT-->>Tool: status JSON
    Tool-->>OpenCrabs: status
    OpenCrabs->>Tool: world_stage_input(kind=query)
    Tool->>WT: world-tool input stage --world ashen-continent --kind query --stdin --json
    WT-->>OpenCrabs: query_file + query_hash
    OpenCrabs->>Tool: world_search_docs(scope=active, query_file, query_hash)
    Tool->>WT: world-tool doc search --world ashen-continent --scope active --query-file runs/inbox/<query-file> --query-hash sha256:... --json
    WT-->>OpenCrabs: related docs
    OpenCrabs->>OpenCrabs: Codex OAuth provider drafts markdown
    OpenCrabs->>Tool: world_stage_input(kind=title)
    Tool->>WT: world-tool input stage --world ashen-continent --kind title --stdin --json
    WT-->>OpenCrabs: title_file + title_hash
    OpenCrabs->>Tool: world_stage_input(kind=body)
    Tool->>WT: world-tool input stage --world ashen-continent --kind body --stdin --json
    WT-->>OpenCrabs: body_file + body_hash
    OpenCrabs->>Tool: world_create_draft(title_file, title_hash, body_file, body_hash)
    Tool->>WT: world-tool draft create --world ashen-continent --change-type create --type nation --title-file runs/inbox/<title-file> --title-hash sha256:... --body-file runs/inbox/<body-file> --body-hash sha256:... --json
    WT->>World: write drafts/ and runs/
    WT-->>OpenCrabs: draft_id, draft_path, run_id
    OpenCrabs->>Tool: world_validate_draft(draft_path)
    Tool->>WT: world-tool draft validate --world ashen-continent --draft drafts/nations/<draft>.md --json
    WT->>World: write validation artifacts
    WT-->>OpenCrabs: validation JSON
    OpenCrabs-->>User: draft summary + validation + next actions
```

위 시퀀스는 schematic이지만 command contract를 깨지 않도록 필수 인자 `--world`, 파일 경로 입력, hash binding, approval provenance를 명시한다.

## 7. Accept 흐름
```mermaid
sequenceDiagram
    participant User
    participant OpenCrabs
    participant Tool as Dynamic Tool
    participant WT as world-tool
    participant World as World Root

    User->>OpenCrabs: "이 draft 승인해"
    OpenCrabs->>Tool: world_diff_draft(draft_path)
    Tool->>WT: world-tool draft diff --world ashen-continent --draft drafts/nations/<draft>.md --json
    WT-->>OpenCrabs: diff summary + diff_run_id + hashes
    OpenCrabs-->>User: 변경 내용 확인
    User->>OpenCrabs: "승인"
    OpenCrabs->>Tool: world_stage_input(kind=reason)
    Tool->>WT: world-tool input stage --world ashen-continent --kind reason --stdin --json
    WT-->>OpenCrabs: reason_file + reason_hash
    OpenCrabs->>Tool: world_accept_draft(draft_path, diff_run_id, hashes, reason_file, reason_hash, approver_id, approval_channel, authenticated_actor)
    Tool->>WT: world-tool draft accept --world ashen-continent --draft drafts/nations/<draft>.md --diff-run-id 20260530-010 --draft-hash sha256:... --target-base-hash sha256:... --patch-hash sha256:... --approver-id park.hana --approval-channel OpenCrabs-chat --authenticated-actor openid:codex-oauth:user-123 --reason-file runs/inbox/<reason-file> --reason-hash sha256:... --json
    WT->>World: verify diff binding and draft validate again
    alt validation pass or warning
        WT->>World: write content/
        WT->>World: move draft to archive/accepted/
        WT->>World: write runs/result.json
        WT-->>OpenCrabs: accepted JSON
        OpenCrabs-->>User: accepted + content path
    else conflict, error, or policy stop
        WT->>World: write blocked result
        WT-->>OpenCrabs: blocked JSON
        OpenCrabs-->>User: blocked reason + fixes
    end
```

## 8. 상태 전이
```mermaid
stateDiagram-v2
    [*] --> DraftCreated
    DraftCreated --> Validating
    Validating --> DraftPass: pass
    Validating --> DraftWarning: warning
    Validating --> DraftConflict: conflict/error
    DraftPass --> AwaitingApproval
    DraftWarning --> AwaitingApproval
    DraftConflict --> NeedsRevision
    NeedsRevision --> DraftCreated: update draft
    AwaitingApproval --> Accepted: user approves + accept tool passes
    AwaitingApproval --> Rejected: user rejects
    Accepted --> ArchivedAccepted
    Rejected --> ArchivedRejected
```

## 9. JSON 계약 예시
정식 JSON envelope는 [commands.md](commands.md)를 기준으로 한다. 아래 예시는 각 tool의 대표 payload다.

`world_create_draft` 결과:

```json
{
  "schema_version": "world-tool.v1",
  "ok": true,
  "command_status": "completed",
  "command": "draft.create",
  "world_id": "ashen-continent",
  "registry_root": "/host/worlds/ashen-continent",
  "root": "/workspace/world",
  "run_id": "20260530-001",
  "data": {
    "draft_id": "draft_20260530_001",
    "draft_path": "drafts/nations/nation_northern_empire.md"
  },
  "issues": [],
  "available_actions": ["world_read_draft", "world_validate_draft", "world_diff_draft", "world_reject_draft"]
}
```

`world_validate_draft` 결과:

```json
{
  "schema_version": "world-tool.v1",
  "ok": true,
  "command_status": "completed",
  "command": "draft.validate",
  "world_id": "ashen-continent",
  "registry_root": "/host/worlds/ashen-continent",
  "root": "/workspace/world",
  "run_id": "20260530-001",
  "data": {
    "draft_path": "drafts/nations/nation_northern_empire.md",
    "validation_status": "warning"
  },
  "issues": [
    {
      "rule": "VR-203",
      "severity": "warning",
      "message": "기존 북부 왕국의 멸망 시점과 현재 통치 표현이 충돌할 수 있음",
      "recommendation": "후계 국가인지 역사 서술인지 명확히 할 것"
    }
  ],
  "available_actions": ["world_update_draft", "world_diff_draft", "world_reject_draft"]
}
```

`world_accept_draft` blocked 결과:

```json
{
  "schema_version": "world-tool.v1",
  "ok": true,
  "command_status": "blocked",
  "command": "draft.accept",
  "world_id": "ashen-continent",
  "registry_root": "/host/worlds/ashen-continent",
  "root": "/workspace/world",
  "run_id": "20260530-002",
  "data": {
    "draft_path": "drafts/nations/nation_northern_empire.md",
    "validation_status": "conflict",
    "block_reason": "VALIDATION_BLOCKED"
  },
  "issues": [
    {
      "rule": "VR-101",
      "severity": "conflict",
      "message": "id nation_northern_empire already exists in content"
    }
  ],
  "available_actions": ["world_update_draft", "world_reject_draft"]
}
```

## 10. 내부 Go 모듈
```text
cmd/world-tool
  CLI entrypoint

internal/config
  registry, harness.yaml, CLI flags

internal/world
  world root resolution and path boundary

internal/docs
  markdown/frontmatter read/write/search

internal/drafts
  draft create/update/read/list

internal/validate
  structural, id, timeline, relationship rules

internal/diff
  content target path and patch generation

internal/audit
  run id, events.jsonl, result artifacts
```

## 11. 구현 순서
1. Go module과 `world-tool --version`
2. world root resolver와 path boundary
3. `world-tool world init/status`
4. Markdown/frontmatter parser
5. `input stage`
6. `draft create/read/list/update`
7. `draft validate`
8. `draft diff`
9. `draft accept/reject`
10. `opencrabs/skills/world-building/SKILL.md`
11. `opencrabs/tools/world-tools.toml`
12. sample world root와 end-to-end smoke test

## 12. 설계 판단
- OpenCrabs가 이미 provider, skill, dynamic tools, channel UX를 제공하므로 별도 agent runtime을 만들지 않는다.
- `world-tool`은 AI SDK를 품지 않는다. AI 판단은 OpenCrabs/Codex가 한다.
- safety-critical 로직은 skill이 아니라 Go tool에서 강제한다.
- dynamic tools는 command template quoting 문제를 피하기 위해 긴 query/title/body/reason/retcon_reason을 `world_stage_input`으로 staging한 뒤 file path와 hash만 전달한다.
- 기본 provider는 Codex OAuth다. Codex CLI provider는 OAuth provider를 사용할 수 없는 환경에서만 fallback으로 둔다.
- OpenCrabs credential/config volume은 world root volume과 분리한다.
- world_id는 logical registry key, registry_root는 canonical registry path, root는 실제 실행 effective root다. Docker에서는 registry_root와 root를 분리해 audit하고, root field는 항상 실행 root 기준으로 해석한다.

## 13. MVP 준비 상태
현재 문서는 구현 전 설계 기준이다. 구현이 필요한 산출물은 다음이다.

- Go `world-tool` CLI
- `opencrabs/skills/world-building/SKILL.md`
- `opencrabs/tools/world-tools.toml`
- `schema/world-doc.schema.json`
- `schema/relationship-types.yaml`
- `examples/worlds/*` 샘플 world root
- end-to-end smoke test

가장 먼저 구현할 vertical slice:

```text
world-tool world init
→ world-tool draft create
→ world-tool draft validate
→ world-tool draft diff
→ world-tool draft accept
```

## 14. 확장 포인트
- Graph rebuild/check: content에서 nodes/edges/orphan report 재생성
- Retcon 관리: frontmatter `retcon_reason`, source_run_id, force reason을 이용해 변경 이력 추적
- Multi-world registry: OpenCrabs 설정 또는 world-tool config로 world id와 root 매핑
- Validation strictness: world별 light/normal/strict 정책
- Archive storage: accepted/rejected archive pruning, compression, export
