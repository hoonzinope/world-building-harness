# schema.md

# OpenCrabs World Tools Schema

## 1. 목적
schema는 세계관 문서의 표준 frontmatter와 본문 섹션을 정의한다. LLM 생성 결과를 일정한 구조로 정리하고 validator가 검사할 기준을 제공한다.

## 2. 공통 Frontmatter
모든 content와 draft 문서는 아래 공통 필드를 가진다.

```yaml
---
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
---
```

### 필드 정의
- id: 전역 고유 id. 타입 접두어를 권장한다.
- type: character, nation, organization, place, event, timeline, magic, glossary, storylet 중 하나.
- status: draft, canon, deprecated, rejected 중 하나.
- title: 사람이 읽는 제목.
- tags: 검색과 분류를 위한 태그.
- created_at: 생성일.
- updated_at: 마지막 수정일.
- related: 관련 entity id 목록. 느슨한 참조와 context loading에 사용한다.
- relationships: typed relationship 목록. validator와 graph builder가 정밀 관계 검증에 사용한다.
- source_run_id: 생성 또는 마지막 major update를 만든 run id. 기존 수동 문서나 import 문서는 null일 수 있다.

relationship 예시:
```yaml
relationships:
  - type: member_of
    target: org_gray_order
    note: 정식 단원
  - type: capital_of
    target: nation_ashen_empire
```

## 3. Character Schema
```yaml
---
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
