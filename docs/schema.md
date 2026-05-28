# schema.md

# World-Building Harness Schema

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
created_at: 2026-05-28
updated_at: 2026-05-28
related: []
source_run_id: 20260528-001
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
- related: 관련 entity id 목록.
- source_run_id: 생성 또는 마지막 major update를 만든 run id.

## 3. Character Schema
```yaml
---
id: character_aria
type: character
status: draft
title: 아리아
aliases: []
affiliation: []
birth_year: null
death_year: null
related: []
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
aliases: []
capital: null
founded_year: null
collapsed_year: null
related: []
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
start_year: null
end_year: null
participants: []
locations: []
related: []
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

## 9. Storylet Schema
storylet은 canon이 아닌 창작 후보로 취급한다.

```yaml
---
id: storylet_trade_conflict_001
type: storylet
status: draft
title: 무역로의 균열
related: []
canon_level: candidate
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

## 10. ID 규칙
권장 접두어:
- character_
- nation_
- org_
- place_
- event_
- magic_
- term_
- storylet_

ID는 영문 소문자, 숫자, 언더스코어만 사용한다.

## 11. Canon Notes
Canon Notes는 설정의 불변 조건, 아직 모호한 부분, 향후 검증이 필요한 부분을 기록한다.
