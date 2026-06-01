# World Lore System Architecture

## 목적

Markdown 기반 세계관 설정 저장소를 구축하고,
LLM이 읽고 정리하기 쉬운 구조로 관리한다.

시스템은 다음 목표를 가진다.

* Git 기반 버전 관리
* 모바일 웹 조회 가능
* self-hosted 운영
* LLM 기반 설정 검토 및 consistency 관리
* container 기반 서비스 분리
* markdown-native workflow 유지

---

# 전체 구조

README에서는 개념적 문서 묶음을 `content/`라고 부르고, 실제 저장소 입력은 `contents/`를 쓴다.

```text id="h43g4v"
Markdown Repository
    ↓
Quartz Build Container
    ↓
Static HTML Output
    ↓
Quartz Site Container (nginx)
    ↓
Existing nginx Reverse Proxy
    ↓
https://yourdomain.com/world/
```

---

# Repository 구조

```text id="9mjlwm"
world-lore/
├── raw/
├── drafts/
├── contents/
│   ├── characters/
│   ├── nations/
│   ├── factions/
│   ├── events/
│   ├── timeline/
│   ├── magic/
│   ├── locations/
│   └── glossary/
├── schema/
├── prompts/
└── tools/
```

---

# Directory 설명

## raw/

정리되지 않은 원본 메모 저장.

예:

* 즉흥 아이디어
* 설정 파편
* 대화 로그
* 조사 자료

외부 공개 대상 아님.

---

## drafts/

정리 중인 초안 문서 저장.

LLM이 raw 기반으로 재구성한 문서 또는
검토 중인 설정 저장.

외부 공개 대상 아님.

---

## contents/

Quartz 공개 대상 markdown 문서 저장. README의 개념적 `content/`에 해당하는 실제 Quartz 입력 경로다.

실제 world wiki 문서 위치.

---

## schema/

LLM consistency 및 문서 구조 규칙 저장.

예:

```text id="5o1y53"
entity-schema.md
timeline-rules.md
naming-rules.md
consistency-rules.md
```

---

## prompts/

LLM workflow용 prompt 저장.

예:

```text id="8sq2e7"
review-lore.md
expand-character.md
timeline-review.md
relationship-review.md
```

---

## tools/

Consistency 검사 및 보조 스크립트 저장.

예:

```text id="7jlwm0"
check-links.py
extract-entities.py
check-timeline.py
```

---

# Markdown 작성 규칙

## 파일명 규칙

모든 파일명은 영어 slug 사용.

예:

```text id="e0a5vp"
empire-chronicle.md
northern-alliance.md
armin-vale.md
```

---

## 문서 표시명

문서 제목은 한글 사용 가능.

예:

```md id="e2f1rz"
---
title: 헬리온 제국 연대기
aliases:
  - 헬리온 제국
  - 제국 연대기
type: nation
---

# 헬리온 제국 연대기
```

---

# LLM Consistency 구조

Consistency는 다음 두 계층으로 관리한다.

---

## 1. Script 기반 검사

정형 consistency 검사 담당.

검사 대상:

* broken link
* orphan page
* duplicate slug
* invalid reference
* timeline numeric conflict
* missing metadata
* invalid relation

---

## 2. LLM 기반 검사

의미 consistency 검사 담당.

검사 대상:

* 설정 충돌
* 세계관 논리 충돌
* 인물 관계 충돌
* 마법 체계 충돌
* 국가 역사 충돌
* timeline narrative conflict

---

# Quartz 운영 구조

Quartz는 static site generator 로 사용한다.

Quartz는 호스트 머신에 설치하지 않는다.

Quartz runtime 및 dependency는 모두 container 내부에서 관리한다.

---

# Host Directory

```text id="9k0l2g"
/srv/world/world-lore
/srv/world/public
/srv/world/quartz-runtime
```

---

# Quartz Runtime

`/srv/world/quartz-runtime`

포함 항목:

```text id="xq7xll"
package.json
package-lock.json
quartz.config.ts
quartz.layout.ts
Dockerfile
```

Quartz dependency 및 node_modules는 container 내부에서 관리한다.

---

# Quartz Build Container

역할:

* markdown 읽기
* quartz build 수행
* static html 생성

입력:

```text id="7y3odj"
/srv/world/world-lore/content
```

출력:

```text id="s0sz6d"
/srv/world/public
```

---

# Quartz Site Container

nginx 기반 static file serving container.

역할:

* generated static html serving

mount:

```text id="i4rn0k"
/srv/world/public
→ /usr/share/nginx/html
```

---

# Existing nginx

외부 reverse proxy 역할 수행.

route:

```nginx id="o93tdv"
location /world/ {
    proxy_pass http://quartz-site/;
}
```

---

# Git Repository

모든 lore 문서는 GitHub private repository 로 관리한다.

대상:

* raw
* drafts
* content
* schema
* prompts
* tools

---

# Build Flow

```text id="9mxj82"
git pull
    ↓
quartz build
    ↓
static html generate
    ↓
public deploy
```

---

# Mobile Access

모바일 브라우저를 통해 접근한다.

URL:

```text id="3m7ww9"
https://yourdomain.com/world/
```

---

# Entity Structure

모든 핵심 entity는 metadata 포함.

예:

```md id="vkzw5q"
---
title: 아르민 베일
type: character
nation: helion-empire
birth_year: 1021
status: alive
tags:
  - mage
  - noble
---

# 아르민 베일
```

---

# Consistency Workflow

```text id="9gmxw5"
raw 작성
    ↓
draft 작성
    ↓
LLM 구조화
    ↓
script consistency check
    ↓
LLM semantic review
    ↓
commit
    ↓
quartz build
```

---

# 목표

이 시스템의 핵심 목적은:

```text id="9vg94s"
"읽기 좋은 위키"
보다
"LLM이 관리 가능한 세계관 저장소"
```

를 구축하는 것이다.
