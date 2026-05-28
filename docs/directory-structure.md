# directory-structure.md

# World-Building Harness Directory Structure

## 1. 기본 구조
```text
world-lore/
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
│   └── storylets/
├── raw/
├── graph/
│   ├── nodes.json
│   ├── edges.json
│   └── orphan-report.json
├── prompts/
│   ├── genesis.md
│   ├── canon-validator.md
│   ├── storylet-generator.md
│   └── markdown-exporter.md
├── schema/
│   ├── character.md
│   ├── nation.md
│   ├── organization.md
│   ├── place.md
│   ├── event.md
│   ├── timeline.md
│   ├── magic.md
│   └── glossary.md
├── runs/
├── archive/
│   ├── accepted/
│   ├── rejected/
│   └── deprecated/
├── docs/
└── harness.yaml
```

## 2. content/
확정된 canon 문서를 저장한다. 사람이 읽는 공개 위키의 원천 데이터로 사용할 수 있다.

정책:
- content는 accept workflow에서만 수정한다.
- LLM 생성 결과가 바로 content에 들어가면 안 된다.
- 모든 content 문서는 frontmatter id를 가져야 한다.

## 3. drafts/
생성 후보 문서를 저장한다.

정책:
- genesis, storylet, export 결과는 기본적으로 drafts에 저장한다.
- draft는 canon이 아니다.
- draft는 validate와 accept를 거쳐야 content로 이동한다.

## 4. raw/
정리되지 않은 아이디어, 메모, 대화 로그, 외부 자료를 저장한다.

정책:
- raw는 canon도 draft도 아니다.
- export workflow를 통해 draft로 변환할 수 있다.

## 5. graph/
content에서 추출한 entity와 relationship의 보조 인덱스를 저장한다.

### nodes.json
인물, 국가, 조직, 장소, 사건, 개념 등의 노드를 저장한다.

### edges.json
노드 간 관계를 저장한다.

### orphan-report.json
존재하지 않는 id를 참조하는 링크, 연결되지 않은 entity 등을 기록한다.

정책:
- graph는 원천 진실이 아니다.
- content에서 재생성 가능해야 한다.

## 6. prompts/
LLM 실행에 사용하는 프롬프트 템플릿을 저장한다.

정책:
- 프롬프트는 workflow와 분리한다.
- target type별 프롬프트 확장이 가능해야 한다.

## 7. schema/
문서 타입별 표준 구조를 저장한다.

정책:
- schema는 Markdown 문서 구조와 frontmatter 필드를 정의한다.
- exporter와 validator가 schema를 참조한다.

## 8. runs/
하네스 실행 기록을 저장한다.

예시:
```text
runs/
└── 20260528-001/
    ├── request.md
    ├── context.md
    ├── plan.md
    ├── result.md
    ├── validation.md
    ├── graph-candidates.json
    ├── diff.patch
    └── metadata.json
```

정책:
- 모든 명령은 run id를 가진다.
- 재현 가능한 수준의 입력과 출력을 남긴다.
- 민감한 API key나 환경변수는 저장하지 않는다.

## 9. archive/
승인, 반려, 폐기된 draft를 보관한다.

## 10. harness.yaml
하네스 설정 파일이다.

예시:
```yaml
world_root: .
default_llm: openai
content_dir: content
draft_dir: drafts
run_dir: runs
graph_dir: graph
approval:
  require_accept: true
  allow_force_accept: true
security:
  deny_outside_root: true
  allow_network: false
```
