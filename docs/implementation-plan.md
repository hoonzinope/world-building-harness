# implementation-plan.md

# Implementation Plan

## 1. 목적
이 문서는 문서화된 설계를 실제 구현으로 옮기기 위한 이정표다. `roadmap.md`가 큰 phase와 방향을 다룬다면, 이 문서는 구현 순서, 산출물, 완료 기준을 체크리스트로 제공한다.

기본 전제:

- OpenCrabs가 하네스/오케스트레이터다.
- 이 레포는 OpenCrabs world-building skill/tools bundle과 `world-tool` Go CLI를 제공한다.
- `content/` Markdown이 canon source of truth다.
- OpenCrabs/Codex는 판단과 생성을 담당하고, 파일 상태 변경은 `world-tool`이 강제한다.
- Codex OAuth provider가 기본 provider이며, Codex CLI provider는 fallback이다.

## 2. 구현 원칙
- 먼저 OpenCrabs 없이 동작하는 `world-tool` vertical slice를 만든다.
- 모든 파일 접근은 world root boundary 안에서만 허용한다.
- 모든 write command는 `runs/`에 audit artifact를 남긴다.
- 모든 command는 `--json` 모드에서 stdout JSON만 반환한다.
- 긴 markdown body, reason, 사용자 입력은 argv가 아니라 `--body-file`, `--reason-file`, stdin으로 받는다.
- `content/`는 `accept draft` 외의 command에서 수정하지 않는다.
- skill은 지침이고, 안전 경계는 `world-tool`에서 강제한다.

## 3. Milestone 0: Repository Scaffold
### 목표
Go CLI와 OpenCrabs bundle을 구현할 기본 디렉토리를 만든다.

### 산출물
- `go.mod`
- `cmd/world-tool/main.go`
- `internal/world`
- `internal/docs`
- `internal/drafts`
- `internal/validate`
- `internal/diff`
- `internal/audit`
- `internal/config`
- `opencrabs/skills/world-building/SKILL.md`
- `opencrabs/tools/world-tools.toml`
- `examples/worlds/ashen-continent`

### 완료 기준
- `go test ./...`가 실행된다.
- `world-tool --help` 또는 동등한 기본 command가 실행된다.
- 아직 OpenCrabs 없이도 local CLI 개발을 시작할 수 있다.

## 4. Milestone 1: World Root Boundary
### 목표
모든 후속 기능의 안전 기반이 되는 world root 접근 계층을 구현한다.

### 산출물
- `--root <path>` 기반 world root 로딩
- `harness.yaml` 기본 설정 로딩
- path normalization
- symlink escape 차단
- world root 밖 read/write 차단
- atomic write helper
- `world-tool world init`
- `world-tool world status`

### 완료 기준
- `world init`이 `content/`, `drafts/`, `runs/`, `archive/`, `graph/`, `harness.yaml`을 생성한다.
- `../`, absolute path, symlink를 통한 root 밖 접근이 차단된다.
- path violation은 JSON error와 non-zero exit code를 반환한다.

## 5. Milestone 2: Content Read Model
### 목표
OpenCrabs가 기존 canon을 안전하게 조회할 수 있게 한다.

### 산출물
- markdown 파일 목록화
- YAML frontmatter 파싱
- document id/type/title/status 추출
- `world-tool doc list`
- `world-tool doc read`
- `world-tool doc search`

### 완료 기준
- `content/`와 `drafts/` scope를 구분해서 조회할 수 있다.
- archive는 기본 조회와 active validation scope에서 제외된다.
- `doc read`는 world root 내부의 markdown만 읽는다.
- `doc search`는 최소한 title, id, tag, body text 기반 검색을 지원한다.

## 6. Milestone 3: Draft Lifecycle
### 목표
LLM 생성물을 canon에 바로 쓰지 않고 draft로 저장하는 경로를 구현한다.

### 산출물
- `world-tool draft create`
- `world-tool draft update`
- `world-tool draft read`
- `world-tool draft list`
- `world-tool draft reject`
- draft frontmatter 보정
- draft id/path 생성 규칙
- `source_run_id` 기록

### 완료 기준
- draft 생성/수정 시 `content/`는 변경되지 않는다.
- markdown body는 `--body-file` 또는 stdin으로 받을 수 있다.
- draft 생성 결과는 JSON으로 draft path, id, run id를 반환한다.
- reject는 draft를 `archive/rejected/`로 이동하고 reason을 audit에 남긴다.

## 7. Milestone 4: Validation MVP
### 목표
draft를 canon과 비교해 구조 오류와 명백한 충돌을 탐지한다.

### 산출물
- `world-tool validate draft`
- `world-tool validate content`
- validation report JSON
- validation artifact 저장
- required field rule
- id uniqueness rule
- relationship target existence rule
- timeline/event consistency rule
- target path conflict rule

### 완료 기준
- validation status는 `pass`, `warning`, `conflict`, `error` 중 하나다.
- schema 누락과 parse 실패는 `error`로 반환된다.
- 기존 canon id와 중복되는 draft는 `conflict`로 반환된다.
- 존재하지 않는 relationship target은 최소 `warning` 이상으로 반환된다.
- validation 결과는 `runs/<run-id>/validation.json`에 저장된다.

## 8. Milestone 5: Diff, Accept, Audit
### 목표
사용자 승인을 받은 draft만 canon으로 승격하고, 재현 가능한 audit trail을 남긴다.

### 산출물
- `world-tool diff draft`
- `world-tool accept draft`
- accept 직전 validation 재실행
- `--force`와 `--reason-file`
- conflict/error 기본 차단
- content atomic write
- accepted draft archive 이동
- `runs/<run-id>/diff.patch`
- `runs/<run-id>/result.json`
- `runs/<run-id>/events.jsonl`

### 완료 기준
- accept는 validation을 다시 실행한다.
- conflict/error가 있으면 기본 accept가 실패한다.
- force accept는 reason 없이는 실패한다.
- accept 성공 시 content 문서가 생성 또는 갱신된다.
- accept 성공 시 draft 원본은 `archive/accepted/`로 이동한다.
- 모든 변경은 runs artifact로 추적 가능하다.

## 9. Milestone 6: OpenCrabs Skill
### 목표
OpenCrabs 대화에서 세계관 작업 규칙을 안정적으로 주입한다.

### 산출물
- `opencrabs/skills/world-building/SKILL.md`
- draft-first workflow 지침
- content 직접 수정 금지 지침
- validation 후 사용자 승인 지침
- warning/conflict/error 응답 지침
- tool output을 authoritative state로 취급하는 지침

### 완료 기준
- skill만 읽어도 OpenCrabs가 어떤 tool을 어떤 순서로 호출해야 하는지 알 수 있다.
- 사용자가 content 직접 수정을 요청해도 draft workflow로 유도한다.
- 사용자가 warning 무시 또는 accept 강행을 요청할 때 reason/audit/force 정책을 설명한다.

## 10. Milestone 7: OpenCrabs Dynamic Tools
### 목표
OpenCrabs가 `world-tool`을 범용 shell이 아니라 의미 단위 tool로 호출하게 한다.

### 산출물
- `opencrabs/tools/world-tools.toml`
- `world_status`
- `world_search_docs`
- `world_read_doc`
- `world_create_draft`
- `world_update_draft`
- `world_validate_draft`
- `world_diff_draft`
- `world_accept_draft`
- `world_reject_draft`
- `world_get_run`

### 완료 기준
- 각 tool은 stdout JSON만 반환한다.
- 긴 body/reason은 temp file 또는 stdin 방식으로 전달된다.
- 범용 `world_exec_shell` 같은 tool은 제공하지 않는다.
- malformed JSON, timeout, non-zero exit에 대한 실패 메시지 정책이 정리되어 있다.

## 11. Milestone 8: Container Runtime
### 목표
OpenCrabs, `world-tool`, skill/tools bundle을 컨테이너에서 운영할 수 있게 한다.

### 산출물
- Dockerfile
- docker-compose 예시
- OpenCrabs config/auth volume
- world root volume
- per-world container 예시
- Codex OAuth provider 기본 설정 안내
- Codex CLI provider fallback 설정 안내

### 완료 기준
- OpenCrabs와 `world-tool`이 같은 컨테이너에서 실행될 수 있다.
- OpenCrabs credential/config volume은 world root volume과 분리된다.
- world root는 특정 폴더만 volume mount된다.
- Codex OAuth provider 사용 시 컨테이너에 Codex CLI나 `~/.codex` 마운트가 필요하지 않다.
- Codex CLI provider fallback을 사용할 때만 별도 auth volume을 사용한다.

## 12. Milestone 9: Sample World E2E
### 목표
샘플 world root로 전체 흐름을 검증한다.

### 산출물
- `examples/worlds/ashen-continent/content/...`
- character/place/event 최소 3개 canon 문서
- pass draft fixture
- conflict draft fixture
- reject fixture
- manual OpenCrabs test script 또는 checklist

### 완료 기준
- `world init -> draft create -> validate -> diff -> accept`가 샘플 world에서 통과한다.
- conflict draft는 기본 accept에서 차단된다.
- rejected draft는 `archive/rejected/`로 이동한다.
- accepted draft는 `content/`에 반영되고 `archive/accepted/`로 이동한다.
- `runs/` artifact만 보고 어떤 변경이 있었는지 추적할 수 있다.

## 13. Milestone 10: Hardening
### 목표
MVP 이후 장기 운영에 필요한 안정성 기능을 추가한다.

### 후보 작업
- validation rule strictness를 world별로 설정
- graph rebuild/check
- archive pruning/compression/export
- retcon/versioning report
- schema migration
- OpenCrabs tool calling retry/timeout 정책
- semantic search integration
- storylet/exporter

### 완료 기준
- 장편 세계관에서 archive와 runs가 늘어나도 운영 정책이 있다.
- graph는 content에서 재생성 가능한 인덱스로 유지된다.
- validator가 확정 판정기가 아니라 conflict 후보 탐지기라는 경계가 유지된다.

## 14. 권장 구현 순서
최소 vertical slice는 다음 순서로 구현한다.

1. Milestone 0: Repository Scaffold
2. Milestone 1: World Root Boundary
3. Milestone 2: Content Read Model
4. Milestone 3: Draft Lifecycle
5. Milestone 4: Validation MVP
6. Milestone 5: Diff, Accept, Audit
7. Milestone 6: OpenCrabs Skill
8. Milestone 7: OpenCrabs Dynamic Tools
9. Milestone 8: Container Runtime
10. Milestone 9: Sample World E2E

Milestone 10은 MVP가 end-to-end로 동작한 뒤 진행한다.

## 15. MVP Done Definition
MVP는 다음을 만족할 때 완료로 본다.

- OpenCrabs 없이 `world-tool`만으로 draft를 만들고 accept할 수 있다.
- `content/`는 accept 전까지 변경되지 않는다.
- validation 결과와 accept 결과가 JSON으로 반환된다.
- conflict/error draft는 기본 accept에서 차단된다.
- 모든 write operation은 `runs/`와 `archive/`에 추적 가능한 기록을 남긴다.
- OpenCrabs skill과 dynamic tools가 같은 workflow를 호출할 수 있다.
- 샘플 world에서 end-to-end 시나리오가 재현된다.
