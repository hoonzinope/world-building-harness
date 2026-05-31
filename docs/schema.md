# schema.md

# OpenCrabs World Tools Schema

## 1. 목적
schema는 세계관 문서의 표준 frontmatter와 본문 섹션을 정의한다. LLM 생성 결과를 일정한 구조로 정리하고 validator가 검사할 기준을 제공한다.

### Machine-readable schema
구현 시 `schema/` 디렉토리에 다음 파일을 둔다.

```text
schema/
├── world-doc.schema.json
├── relationship-types.yaml
└── document-types.yaml
```

이 문서는 사람이 읽는 기준이고, validator는 위 machine-readable schema를 기준으로 동작해야 한다. 문서와 schema 파일이 충돌하면 구현 단계에서는 schema 파일을 고치고 이 문서를 같이 갱신한다.

schema version:
- Markdown frontmatter에는 `schema_version: world-doc.v1`을 기록한다.
- `world-tool draft create`는 새 draft에 `schema_version`과 `change_type`을 자동 주입한다.
- legacy/import 문서에 `schema_version`이 없으면 MVP에서는 warning으로 처리할 수 있지만, 새로 생성되는 문서에는 필수다.

`world-doc.schema.json` 최소 구조:
- `$schema`, `$id`, `type: object`
- common required: `schema_version`, `id`, `type`, `status`, `title`, `created_at`, `updated_at`
- common optional: `tags`, `related`, `relationships`, `source_run_id`, `change_type`, `target_id`, `retcon_reason`
- type-specific definitions: `character`, `nation`, `organization`, `place`, `event`, `timeline`, `magic`, `glossary`, `storylet`
- relationship item definition: `type`, `target`, optional `note`
- status/type enum

`document-types.yaml`은 type별 content directory, draft directory, id prefix, storylet 예외를 정의한다. `relationship-types.yaml`은 relationship allowlist, inverse, contradicts, source/range, symmetric 여부를 정의한다. validator는 `relationships[]`를 canonical graph로 쓰고, convenience field를 normalize할 때 이 metadata를 사용한다.

## 2. 공통 Frontmatter
모든 content와 draft 문서는 아래 공통 필드를 가진다. 단, `change_type` 계열 필드는 draft에만 의미가 있다.

```yaml
---
schema_version: world-doc.v1
id: character_aria
type: character
status: draft
title: 아리아
tags: [north, royal-family]
created_at: 2026-05-29
updated_at: 2026-05-29
related: []
relationships: []
source_run_id: 20260529-001
change_type: create
target_id: null
retcon_reason: null
---
```

### 필드 정의
- schema_version: 문서 schema 버전. MVP 기본값은 `world-doc.v1`.
- id: 전역 고유 id. 타입 접두어를 권장한다.
- type: character, nation, organization, place, event, timeline, magic, glossary, storylet 중 하나.
- status: draft, canon, deprecated, rejected 중 하나.
- title: 사람이 읽는 제목.
- tags: 검색과 분류를 위한 태그.
- created_at: 생성일.
- updated_at: 마지막 수정일.
- related: 관련 entity id 목록. 느슨한 참조와 context loading에 사용한다.
- relationships: typed relationship 목록. canonical graph representation이다.
- source_run_id: 생성 또는 마지막 major update를 만든 run id. 기존 수동 문서나 import 문서는 null일 수 있다.
- change_type: draft가 canon에 적용되는 방식. draft에서는 `create`, `update`, `deprecate` 중 하나가 필수다. content 문서는 생략하거나 null로 둔다.
- target_id: `update` 또는 `deprecate` 대상 canon id. draft의 `update`와 `deprecate`에서는 필수다. `create`에서는 null 또는 생략이다.
- retcon_reason: 기존 canon을 수정하거나 폐기하는 이유. draft의 `update`와 `deprecate`에서는 필수다. `create`에서는 null 또는 생략이다.

relationship 예시:
```yaml
relationships:
  - type: member_of
    target: org_gray_order
    note: 정식 단원
  - type: capital_of
    target: nation_ashen_empire
```

### 필드 타입
공통 타입 규칙:
- `id`, `type`, `status`, `title`, `schema_version`은 string이다.
- `change_type`, `target_id`, `retcon_reason`은 string 또는 null이다.
- `tags`, `related`, `aliases`, `affiliation`, `participants`, `locations`는 string array다.
- `capital`, `located_in`, `headquarters`는 string 또는 null이다.
- `relationships`는 object array다.
- `created_at`, `updated_at`은 `YYYY-MM-DD` 또는 RFC3339 timestamp를 허용한다. 같은 world 안에서는 하나의 형식을 유지하는 것을 권장한다.
- 연도 필드는 integer 또는 null이다. 불확실한 연도는 본문 `Canon Notes`에 남긴다.
- 참조 필드는 entity id를 값으로 사용한다. 사람이 읽는 title을 넣지 않는다.

### Draft change contract
- draft 문서는 `change_type`을 반드시 가진다.
- `change_type`이 `update` 또는 `deprecate`이면 `target_id`와 `retcon_reason`이 모두 필수다.
- `change_type`이 `create`이면 `target_id`와 `retcon_reason`은 null이거나 생략되어야 한다.
- missing, null, 잘못된 enum 값은 validator error다.

### Relationship normalization contract
- `relationships[]`가 canonical graph다.
- `affiliation`, `capital`, `located_in`, `participants`, `locations`는 authoring convenience field다.
- validator는 convenience field를 아래와 같이 관계로 normalize한다.
- `affiliation` -> `member_of`
- `capital` -> `capital_of`
- `located_in` -> `located_in`
- `participants` -> `participates_in`
- `locations` -> `occurred_at`
- `affiliation`은 strict membership shorthand다. 더 느슨한 관계는 `relationships[]`의 `affiliated_with`를 직접 쓴다.
- convenience field와 explicit `relationships[]`가 같은 사실을 다른 방식으로 표현해야 한다. 같은 source/target/type로 normalize되지 않으면 conflict다.
- convenience field가 여러 값이면 각 값은 별도 edge로 normalize된다.

### Relationship Type Allowlist
MVP validator가 정적으로 이해하는 relationship type은 아래 allowlist다. `inverse`는 graph builder가 합성하는 normalized inverse label이고, `contradicts`는 같은 사실 공간에서 동시에 존재할 수 없는 type을 뜻한다.

| type | inverse | contradicts | domain | range | notes |
| --- | --- | --- | --- | --- | --- |
| `member_of` | `has_member` | - | character, organization | organization, nation | canonical membership edge |
| `affiliated_with` | `has_affiliation` | - | character, organization | organization, nation | loose affiliation edge |
| `located_in` | `contains` | - | place, organization | place, nation | normalized from `located_in` convenience field |
| `capital_of` | `has_capital` | - | place | nation | normalized from `capital` convenience field |
| `rules` | `ruled_by` | - | character, organization | nation, place | governance edge |
| `parent_of` | `child_of` | `child_of` on the same oriented pair | character | character | parent/child pair is inverse-normalized |
| `child_of` | `parent_of` | `parent_of` on the same oriented pair | character | character | parent/child pair is inverse-normalized |
| `sibling_of` | `sibling_of` | - | character | character | symmetric edge |
| `ally_of` | `ally_of` | `rival_of` | character, nation, organization | character, nation, organization | symmetric alliance edge |
| `rival_of` | `rival_of` | `ally_of` | character, nation, organization | character, nation, organization | symmetric hostility edge |
| `predecessor_of` | `successor_of` | `successor_of` on the same oriented pair | nation, organization, event | nation, organization, event | predecessor/successor pair is inverse-normalized |
| `successor_of` | `predecessor_of` | `predecessor_of` on the same oriented pair | nation, organization, event | nation, organization, event | predecessor/successor pair is inverse-normalized |
| `participates_in` | `has_participant` | - | character, organization, nation | event | normalized from `participants` convenience field |
| `occurred_at` | `hosts_event` | - | event | place | normalized from `locations` convenience field |
| `uses_magic` | `used_by` | - | character, organization, nation | magic | spell/source ownership edge |

알 수 없는 relationship type은 MVP 기본 모드에서 conflict다. graph builder가 generic edge로 흡수하는 것은 후속 기능의 명시적 opt-in 모드에서만 허용한다.

### Document Type Directory
MVP target path는 title slug가 아니라 id 기반으로 계산한다.

| type | content directory | draft directory | id prefix |
| --- | --- | --- | --- |
| `character` | `content/characters/` | `drafts/characters/` | `character_` |
| `nation` | `content/nations/` | `drafts/nations/` | `nation_` |
| `organization` | `content/organizations/` | `drafts/organizations/` | `org_` |
| `place` | `content/places/` | `drafts/places/` | `place_` |
| `event` | `content/events/` | `drafts/events/` | `event_` |
| `timeline` | `content/timeline/` | `drafts/timeline/` | `timeline_` |
| `magic` | `content/magic/` | `drafts/magic/` | `magic_` |
| `glossary` | `content/glossary/` | `drafts/glossary/` | `term_` |
| `storylet` | n/a | `drafts/storylets/` only | `storylet_` |

content target path는 `<content directory>/<id>.md`다. non-storylet draft path는 `<draft directory>/<id>.md`다. `change_type: update`와 `change_type: deprecate`는 기존 content path를 유지한다.

## 3. Character Schema
```yaml
---
schema_version: world-doc.v1
id: character_aria
type: character
status: draft
title: 아리아
tags: [north, royal-family]
created_at: 2026-05-29
updated_at: 2026-05-29
aliases: []
affiliation: []
birth_year: null
death_year: null
related: []
relationships: []
source_run_id: 20260529-001
change_type: create
target_id: null
retcon_reason: null
---
```

본문 섹션:
```markdown
# 인물명

## 한 줄 요약

## 개요

## 배경

## 성격과 욕망

## 소속과 관계

## 주요 사건

## Canon Notes

## Related
```

## 4. Nation Schema
```yaml
---
schema_version: world-doc.v1
id: nation_ashen_empire
type: nation
status: draft
title: 잿빛 제국
tags: [empire]
created_at: 2026-05-29
updated_at: 2026-05-29
aliases: []
capital: null
founded_year: null
collapsed_year: null
related: []
relationships: []
source_run_id: 20260529-001
change_type: create
target_id: null
retcon_reason: null
---
```

본문 섹션:
```markdown
# 국가명

## 한 줄 요약

## 지리

## 역사

## 정치 구조

## 경제와 자원

## 군사

## 문화

## 주요 인물

## 주요 사건

## Canon Notes

## Related
```

## 5. Organization Schema
```yaml
---
schema_version: world-doc.v1
id: org_gray_order
type: organization
status: draft
title: 잿빛 기사단
tags: []
created_at: 2026-05-29
updated_at: 2026-05-29
aliases: []
founded_year: null
dissolved_year: null
headquarters: null
related: []
relationships: []
source_run_id: 20260529-001
change_type: create
target_id: null
retcon_reason: null
---
```

본문 섹션:
```markdown
# 조직명

## 한 줄 요약

## 목적

## 구조

## 활동 지역

## 주요 인물

## 갈등 관계

## Canon Notes

## Related
```

## 6. Place Schema
```yaml
---
schema_version: world-doc.v1
id: place_gray_capital
type: place
status: draft
title: 잿빛 수도
tags: []
created_at: 2026-05-29
updated_at: 2026-05-29
aliases: []
located_in: null
related: []
relationships: []
source_run_id: 20260529-001
change_type: create
target_id: null
retcon_reason: null
---
```

본문 섹션:
```markdown
# 장소명

## 한 줄 요약

## 위치

## 지리적 특징

## 역사

## 거주자와 세력

## 주요 사건

## Canon Notes

## Related
```

## 7. Event Schema
```yaml
---
schema_version: world-doc.v1
id: event_fall_of_north
type: event
status: draft
title: 북부의 몰락
tags: []
created_at: 2026-05-29
updated_at: 2026-05-29
start_year: null
end_year: null
participants: []
locations: []
related: []
relationships: []
source_run_id: 20260529-001
change_type: create
target_id: null
retcon_reason: null
---
```

본문 섹션:
```markdown
# 사건명

## 한 줄 요약

## 배경

## 전개

## 결과

## 관련 인물

## 관련 세력

## 연표 위치

## Canon Notes

## Related
```

## 8. Magic Schema
```yaml
---
schema_version: world-doc.v1
id: magic_ash_binding
type: magic
status: draft
title: 재의 결속
tags: []
created_at: 2026-05-29
updated_at: 2026-05-29
aliases: []
related: []
relationships: []
source_run_id: 20260529-001
change_type: create
target_id: null
retcon_reason: null
---
```

본문 섹션:
```markdown
# 마법 체계명

## 한 줄 요약

## 원리

## 사용 조건

## 한계와 비용

## 사회적 영향

## 금기

## 주요 사용자

## Canon Notes

## Related
```

## 9. Timeline Schema
```yaml
---
schema_version: world-doc.v1
id: timeline_main
type: timeline
status: draft
title: 주요 연표
tags: []
created_at: 2026-05-29
updated_at: 2026-05-29
related: []
relationships: []
source_run_id: 20260529-001
change_type: create
target_id: null
retcon_reason: null
---
```

본문 섹션:
```markdown
# 연표명

## 범위

## 주요 시대

## 주요 사건

## 불확실한 시점

## Canon Notes

## Related
```

## 10. Glossary Schema
```yaml
---
schema_version: world-doc.v1
id: term_ether
type: glossary
status: draft
title: 에테르
tags: []
created_at: 2026-05-29
updated_at: 2026-05-29
aliases: []
related: []
relationships: []
source_run_id: 20260529-001
change_type: create
target_id: null
retcon_reason: null
---
```

본문 섹션:
```markdown
# 용어명

## 정의

## 사용 맥락

## 관련 설정

## 혼동하기 쉬운 용어

## Canon Notes

## Related
```

## 11. Storylet Schema
storylet은 canon이 아닌 창작 후보로 취급한다.

```yaml
---
schema_version: world-doc.v1
id: storylet_trade_conflict_001
type: storylet
status: draft
title: 무역로의 균열
tags: []
created_at: 2026-05-29
updated_at: 2026-05-29
related: []
relationships: []
canon_level: candidate
source_run_id: 20260529-001
change_type: create
target_id: null
retcon_reason: null
---
```

본문 섹션:
```markdown
# Storylet 제목

## Hook

## 등장 Entity

## 갈등

## 선택지

## 결과 후보

## Canon Impact

## Related
```

## 12. ID 규칙
권장 접두어:
- character_
- nation_
- org_
- place_
- event_
- timeline_
- magic_
- term_
- storylet_

ID는 영문 소문자, 숫자, 언더스코어만 사용한다.

## 13. Canon Notes
Canon Notes는 설정의 불변 조건, 아직 모호한 부분, 향후 검증이 필요한 부분을 기록한다.

## 14. Source of Truth 정책
- status가 canon인 문서는 content/에 있어야 한다.
- status가 draft인 문서는 drafts/ 또는 archive/에 있을 수 있다.
- archive/accepted/의 draft 원본은 active validation과 context loading에서 제외한다.
- OpenCrabs DB나 graph는 content Markdown에서 재생성 가능한 보조 데이터다.

Storylet 정책:
- MVP에서 storylet active draft는 `drafts/storylets/`에서만 관리한다.
- 기본 `draft accept`는 storylet을 `content/` canon으로 승격하지 않는다.
- `type: storylet`과 `status: canon` 조합은 content validation `error`이며 accept에서는 blocked 상태다.
- accepted/rejected archive 원본은 active draft scope 밖의 보관물이다.
- storylet을 canon 사건이나 entity로 반영하려면 별도 event/character/place draft를 생성해 accept한다.
