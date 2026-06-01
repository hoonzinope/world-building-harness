---
title: Entity Schema
type: schema
slug: entity-schema
language: ko
updated_at: 2026-05-21
---

# Entity Schema

이 문서는 README의 개념적 `content/` 표기를 따르되, 현재 저장소의 Quartz 공개 입력인 `contents/`에 들어갈 개별 엔티티 문서의 최소 구조를 정의한다.
목표는 사람이 읽기 쉽고, LLM이 정리하기 쉽고, 스크립트가 검증하기 쉬운 공통 형태를 유지하는 것이다.

## 적용 범위

- `content/characters/` / `contents/characters/`
- `content/nations/` / `contents/nations/`
- `content/factions/` / `contents/factions/`
- `content/events/` / `contents/events/`
- `content/timeline/` / `contents/timeline/`
- `content/magic/` / `contents/magic/`
- `content/locations/` / `contents/locations/`
- `content/glossary/` / `contents/glossary/`

`raw/`는 정리 전 입력 원본이며, 이 스키마의 직접 대상이 아니다.
`drafts/`는 정리 중간 단계이며, `contents/`로 이동하기 전까지는 완전성을 강제하지 않는다.

## 기본 원칙

1. 한 문서는 한 엔티티만 다룬다.
2. 엔티티의 표시명은 한국어 가능하지만, 파일명과 `slug`는 반드시 영어 슬러그를 사용한다.
3. 모든 공개 문서는 YAML frontmatter를 가진다.
4. frontmatter의 키는 가능한 한 고정 집합을 유지한다.
5. 본문 첫 제목은 `title`과 같은 표시명을 반복한다.

## 필수 frontmatter

아래 키는 모든 엔티티 문서에 권장되며, 최소한 `title`, `type`, `slug`는 필수다.

- `title`: 표시명
- `type`: 엔티티 분류
- `slug`: 영어 슬러그

권장 추가 키:

- `aliases`: 검색용 별칭 배열
- `tags`: 분류 태그 배열
- `status`: `draft`, `active`, `retired`, `dead`, `unknown` 등
- `source`: 생성 출처 또는 참고 원천
- `created_at`: 최초 생성일
- `updated_at`: 마지막 갱신일

## 엔티티별 권장 필드

### character

- `name` 또는 `title`
- `nation`
- `birth_year`
- `death_year`
- `status`
- `affiliation`

### nation

- `capital`
- `government`
- `language`
- `currency`
- `status`

### faction

- `leader`
- `base`
- `goal`
- `status`

### event

- `event_date`
- `event_year`
- `participants`
- `location`
- `outcome`

### timeline

- `start_year`
- `end_year`
- `era`
- `covered_entities`

### magic

- `system_name`
- `rules`
- `limitations`
- `cost`

### location

- `region`
- `coordinates` 또는 상대 위치 설명
- `parent_location`

### glossary

- `term`
- `definition`
- `related_terms`

## relation 표현

관계는 문자열 한 줄로 흩뿌리지 말고, 가능하면 배열 또는 참조 키로 관리한다.

권장 방식:

- 단일 참조: `nation: helion-empire`
- 다중 참조: `related: [a, b, c]`
- 구조화 참조:

```yaml
relations:
  ally_of:
    - northern-alliance
  enemy_of:
    - black-court
```

## 본문 구조

권장 순서:

1. 제목
2. 요약
3. 기본 정보
4. 상세 서술
5. 관계/연표/참고

본문은 자유 형식이지만, 다음 규칙을 지킨다.

- 첫 문단은 2~4문장 요약
- 목록은 가능하면 구조화
- 장황한 원문 인용보다 요약 우선
- 서로 다른 사실은 서로 다른 문단으로 분리

## 예시 frontmatter

```md
---
title: 아르민 베일
type: character
slug: armin-vale
aliases:
  - 아르민
  - 베일 경
tags:
  - mage
  - noble
nation: helion-empire
status: alive
birth_year: 1021
created_at: 2026-05-21
updated_at: 2026-05-21
---

# 아르민 베일
```

## 검증 기준

- `title`, `type`, `slug`가 존재한다.
- `slug`는 소문자 영어, 숫자, 하이픈만 사용한다.
- 파일명은 `slug.md` 형식을 따른다.
- `type` 값이 허용 목록 안에 있다.
- 본문 첫 제목이 `title`과 일치한다.
- `aliases`는 문자열 배열이다.
- 날짜 필드는 `YYYY-MM-DD` 또는 `YYYY` 형식만 허용한다.
- 참조값은 존재하는 다른 문서의 `slug`와 맞춰야 한다.
- 한 문서가 두 개 이상의 엔티티를 주제로 삼지 않는다.
