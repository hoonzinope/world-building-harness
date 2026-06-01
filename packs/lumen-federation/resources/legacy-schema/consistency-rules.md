---
title: Consistency Rules
type: schema
slug: consistency-rules
language: ko
updated_at: 2026-05-21
---

# Consistency Rules

이 문서는 세계관 저장소의 일관성 검토 기준을 정의한다.
여기서 일관성은 두 계층으로 나뉜다.

1. 스크립트가 바로 잡을 수 있는 형식 일관성
2. LLM이 판단해야 하는 의미 일관성

## 작업 경계

- `raw/`는 원본 보관소다.
- `raw/`는 위키화 대상이 아니다.
- `raw/`는 필요한 경우 최소 범위만 참조한다.
- README의 개념적 `content/`는 공개 및 정리 완료 문서를 뜻하며, 현재 저장소의 Quartz 공개 입력은 `contents/`다.
- `drafts/`는 중간 산출물이며, 일관성 검토 중에도 변경될 수 있다.

## 스크립트 검사 대상

스크립트는 다음을 검사한다.

- broken link
- orphan page
- duplicate slug
- invalid reference
- missing metadata
- invalid relation key
- timeline numeric conflict
- file name and slug mismatch

## LLM 검사 대상

LLM은 다음을 검사한다.

- 설정 충돌
- 세계관 논리 충돌
- 인물 관계 충돌
- 마법 체계 충돌
- 국가 역사 충돌
- 시간 서술 충돌
- 문체와 용어의 비일관성

## 일관성 우선순위

충돌이 발견되면 다음 순서로 확인한다.

1. canonical frontmatter
2. 동일 저장소 내 최신 공개 문서(`content/` 개념, 실제 `contents/`)
3. 관련 `timeline/` 문서
4. 관련 `glossary/` 문서
5. `drafts/`
6. `raw/`

`raw/`는 참고는 가능하지만 최종 기준으로 바로 삼지 않는다.

## 관계 일관성 규칙

- 동일 인물은 문서마다 다른 이름으로 불리지 않도록 한다.
- 조직명과 국가명은 구분한다.
- 위치는 상위-하위 관계를 유지한다.
- 죽음, 폐위, 해산 같은 상태 변화는 기존 문서와 동기화한다.

## 시간 일관성 규칙

- 사건의 원인과 결과 순서가 뒤집히지 않아야 한다.
- 인물의 생년과 주요 사건 연도가 논리적으로 충돌하지 않아야 한다.
- 같은 사건을 서로 다른 연표에서 다룰 경우, 공통 기준 연도를 공유해야 한다.

## 용어 일관성 규칙

- 같은 개념을 다른 한글 표기로 중복 정의하지 않는다.
- 용어의 canonical form은 `glossary/`에 둔다.
- 약칭은 본문에서 설명한 뒤 재사용한다.
- 영어 용어는 필요한 경우에만 병기한다.

## 예시 frontmatter

```md
---
title: 일관성 검토 기준
type: schema
slug: consistency-rules
tags:
  - validation
  - workflow
updated_at: 2026-05-21
---

# 일관성 검토 기준
```

## 검증 기준

- 모든 공개 문서는 최소한 `title`, `type`, `slug`를 가진다.
- 파일명과 `slug`가 다르면 설명 가능한 예외가 아닌 한 실패로 본다.
- `content/`는 개념적으로 정리 완료본을 뜻하고, 실제 저장소 입력은 `contents/`다.
- `raw/`는 직접 위키화하지 않는다.
- 시간 충돌, 관계 충돌, 용어 충돌은 별도 검토 대상으로 분리한다.
- 스크립트 검사와 LLM 검사를 같은 목록으로 섞지 않는다.
