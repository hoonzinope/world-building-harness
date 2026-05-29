# workflow.md

# World-Building Harness Workflow

## 1. 원칙
모든 workflow는 요청을 바로 canon에 반영하지 않는다. 생성 결과는 draft에 저장되고, validate를 통과하거나 충돌 후보를 검토한 뒤 accept 단계에서만 content로 승격된다. `content/` Markdown은 canon source of truth이며 OpenCrab DB, graph, search index는 이를 보조한다.

LLM 또는 Codex SDK runner는 특정 step의 실행 엔진일 뿐 workflow 전체를 결정하지 않는다. workflow 순서, permission boundary, content write 여부는 world-harness core가 결정한다.

## 2. Orchestration 책임 분리
```text
OpenCrab
  - world registry 관리
  - 사용자/채널/승인 UX 관리
  - long-running job 상태 관리
  - world CLI를 argv subprocess로 호출

world-harness
  - workflow state machine 실행
  - context loading / validation / writer 수행
  - content write와 archive 정책 집행
  - runs log 작성

Agent Runner
  - draft 생성
  - context 요약
  - semantic validation 후보 생성
```

## 3. 공통 실행 단계
```text
receive request
→ create run id
→ load harness config
→ check permission boundary
→ load context
→ run workflow steps
→ write outputs
→ write run log
→ return summary
```

모든 workflow는 `runs/{run_id}/events.jsonl`에 step 상태를 append한다. 기존 run id로 이어 실행하는 경우에도 기존 artifact를 덮어쓰기보다 새 event와 결과 artifact를 추가한다.

## 4. Genesis Workflow
목적: 자연어 요청을 기반으로 새로운 세계관 설정 draft를 생성한다.

### 입력
- 사용자 자연어 요청
- optional target type: character, nation, event, magic, organization, place, timeline
- optional related entity ids

### 단계
1. run id 생성
2. 요청 의도 분석
3. target type 추론
4. 관련 content 문서 검색
5. 관련 graph node/edge 검색
6. 필요한 경우 사용자에게 추가 질문 생성
7. Agent Runner로 draft 생성
8. frontmatter 보정
9. drafts/에 markdown 저장
10. canon validation 실행
11. graph candidates 생성
12. runs/{run_id}/에 request, context, result, validation 저장
13. 요약 반환

### 출력
- drafts/{type}/{slug}.md
- runs/{run_id}/result.md
- runs/{run_id}/validation.md
- runs/{run_id}/graph-candidates.json

## 5. Validate Workflow
목적: draft 또는 content 후보가 기존 canon과 충돌하는지 검사한다.

### 입력
- draft path
- optional validation level: light, normal, strict

### 단계
1. 대상 draft 읽기
2. frontmatter 파싱
3. 필수 필드 검사
4. id 중복 검사
5. 관련 canon 문서 로딩
6. timeline 검사
7. entity relationship 검사
8. terminology 중복/불일치 검사
9. optional semantic validation 실행
10. validation report 생성
11. runs/{run_id}/validation.md 저장

### 출력
- validation status: pass, warning, conflict, error
- conflict candidates
- recommended fix

## 6. Accept Workflow
목적: 사람이 승인한 draft를 content로 승격한다.

### 입력
- draft path
- optional commit message

### 단계
1. draft 존재 확인
2. validate 재실행
3. conflict가 있으면 기본적으로 중단
4. content target path 계산
5. git diff 또는 파일 diff 생성
6. content 문서 생성 또는 갱신
7. content frontmatter status를 canon으로 갱신
8. accepted draft 원본을 archive/accepted/로 이동
9. graph 업데이트
10. runs log 업데이트
11. OpenCrab index/cache 재색인 이벤트 제공
12. optional git commit
13. 결과 요약 반환

### 정책
- conflict 상태에서는 `--force` 없이는 accept 불가
- `--force`를 사용하더라도 runs log에 강제 승인 사유를 남긴다
- accept 이후 draft는 archive/accepted/로 이동하며 active context, id 중복 검사, validation 대상에서 제외한다
- accept workflow는 deterministic하게 실행하며 Agent Runner가 직접 수행하지 않는다

## 7. Storylet Workflow
목적: 기존 세계관을 기반으로 짧은 사건, 갈등, 퀘스트, 장면 씨앗을 생성한다.

### 단계
1. 요청 분석
2. 관련 canon 로딩
3. 등장 entity 후보 선정
4. 갈등 구조 생성
5. canon 침범 여부 검사
6. drafts/storylets/에 저장

### 정책
storylet은 canon이 아니라 creative candidate다. content 승격은 별도의 accept가 필요하다.

## 8. Export Workflow
목적: LLM 출력 또는 raw note를 표준 Markdown 문서로 변환한다.

### 단계
1. 입력 문서 읽기
2. target schema 선택
3. frontmatter 생성
4. 섹션 정규화
5. 관련 문서 링크 후보 생성
6. markdown 저장

## 9. Rebuild Graph Workflow
목적: content 전체를 기준으로 graph store를 재생성한다.

### 단계
1. content/**/*.md 스캔
2. frontmatter id 수집
3. 문서 내 relationship 파싱
4. nodes.json 생성
5. edges.json 생성
6. orphan link report 생성

## 10. Run Log 예시
```text
runs/
└── 20260529-001/
    ├── request.json
    ├── context-manifest.json
    ├── context.md
    ├── draft.md
    ├── validation.json
    ├── validation.md
    ├── graph-candidates.json
    ├── diff.patch
    ├── events.jsonl
    └── result.json
```

`events.jsonl` 예시:
```json
{"step":"create_run","status":"completed","time":"2026-05-29T10:00:00+09:00"}
{"step":"load_context","status":"completed","time":"2026-05-29T10:00:02+09:00"}
{"step":"generate_draft","status":"completed","runner":"codex-sdk"}
{"step":"validate_draft","status":"completed","validation_status":"warning"}
```

## 11. Workflow 설정 예시
```yaml
workflows:
  genesis:
    steps:
      - create_run
      - load_context
      - generate_draft
      - write_draft
      - validate_canon
      - write_graph_candidates
      - write_run_log

  accept:
    steps:
      - validate_canon
      - create_diff
      - promote_draft
      - archive_accepted_draft
      - update_graph
      - write_run_log
```
