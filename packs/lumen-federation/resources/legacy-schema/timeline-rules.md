---
title: Timeline Rules
type: schema
slug: timeline-rules
language: ko
updated_at: 2026-05-21
---

# Timeline Rules

이 문서는 세계관 연표와 사건 문서의 시간 표기 규칙을 정의한다.
목표는 연도 충돌, 순서 역전, 서술 혼선을 최소화하는 것이다.
README에서는 개념적으로 `content/`라고 부르지만, 현재 저장소의 Quartz 공개 입력은 `contents/`다.

## 적용 범위

- `content/timeline/` / `contents/timeline/`
- `content/events/` / `contents/events/`
- 시간 정보가 있는 모든 엔티티 문서

## 기본 원칙

1. 기준 연도 체계를 먼저 정하고, 문서 전체에서 일관되게 사용한다.
2. 시간 정보는 사람이 읽을 수 있어야 하며, 동시에 정렬 가능해야 한다.
3. 불명확한 날짜는 추정값과 확정값을 구분한다.
4. 사건은 발생 시점과 서술 시점을 분리할 수 있다.

## 권장 시간 형식

- 연도: `YYYY`
- 날짜: `YYYY-MM-DD`
- 범위: `YYYY..YYYY` 또는 `YYYY-MM-DD..YYYY-MM-DD`
- 상대 시간: 본문에서만 제한적으로 사용

비권장:

- `1021년쯤`
- `한참 뒤`
- `약 3년 후`

이런 표현은 본문 설명으로는 허용하되, 검증 가능한 메타데이터 필드에는 넣지 않는다.

## 필수 필드 예시

사건 문서나 연표 문서는 아래 중 일부를 가진다.

- `event_year`
- `event_date`
- `start_year`
- `end_year`
- `era`
- `sequence`

`sequence`는 같은 연도 내 순서를 표현할 때만 쓴다.

## 시간 충돌 처리

같은 문서 안에서 다음 문제가 생기면 정리 대상이다.

- `start_year`가 `end_year`보다 늦다
- 같은 사건이 서로 다른 연도로 반복된다
- 동일 인물이 동시에 둘 이상의 위치에 존재한다
- 연표와 본문 서술이 서로 다른 순서를 말한다

충돌이 나면 우선순위는 다음과 같다.

1. 명시된 날짜
2. 명시된 연도
3. 연표 문서의 정렬 순서
4. 본문 서술의 추정 표현

## 연표 작성 규칙

- 연표는 오래된 사건에서 최신 사건 순으로 정렬한다.
- 동년 사건은 `sequence`로 순서를 고정한다.
- 한 줄에 사건 하나를 두는 편이 좋다.
- 사건 설명에는 원인과 결과를 짧게 함께 적는다.

## 예시 frontmatter

```md
---
title: 제국 붕괴 연표
type: timeline
slug: imperial-collapse-timeline
era: second-age
start_year: 1010
end_year: 1042
sequence: 1
---

# 제국 붕괴 연표
```

## 검증 기준

- 날짜와 연도는 허용 형식만 사용한다.
- 시작값이 끝값보다 늦지 않는다.
- 같은 `slug`의 연표 문서는 중복 생성되지 않는다.
- 동년 사건은 순서 기준이 존재한다.
- 본문과 메타데이터의 시간 서술이 서로 모순되지 않는다.
- 상대 시간 표현은 메타데이터가 아니라 본문에 둔다.
