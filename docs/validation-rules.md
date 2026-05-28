# validation-rules.md

# World-Building Harness Validation Rules

## 1. 목적
validation은 LLM이 생성한 설정을 canon으로 믿지 않고, 기존 세계관과의 충돌 후보를 탐지하기 위한 안전장치다. validator의 결과는 최종 판정이 아니라 사람이 검토할 근거다.

## 2. Validation Status
### pass
명백한 구조 오류나 충돌 후보가 없다.

### warning
canon과 충돌한다고 확정할 수는 없지만 검토가 필요한 부분이 있다.

### conflict
기존 canon과 직접 충돌할 가능성이 높다. 기본 accept는 차단된다.

### error
파일 파싱 실패, 필수 필드 누락, schema 불일치 등으로 검증을 완료할 수 없다.

## 3. Structural Rules
### VR-001: frontmatter 존재
모든 draft와 content 문서는 YAML frontmatter를 가져야 한다.

### VR-002: 필수 필드 존재
필수 필드:
- id
- type
- status
- title
- created_at
- updated_at

### VR-003: type 허용값
허용 type:
- character
- nation
- organization
- place
- event
- timeline
- magic
- glossary
- storylet

### VR-004: status 허용값
허용 status:
- draft
- canon
- deprecated
- rejected

## 4. ID Rules
### VR-101: id 전역 중복 금지
content 전체와 draft 대상 간 id가 중복되면 conflict다.

### VR-102: id 형식
id는 영문 소문자, 숫자, 언더스코어만 허용한다.

### VR-103: type-prefix 권장
예:
- character_aria
- nation_ashen_empire
- event_fall_of_north

prefix 불일치는 warning이다.

## 5. Timeline Rules
### VR-201: 출생/사망 연도 순서
character의 birth_year는 death_year보다 늦을 수 없다.

### VR-202: 사건 시작/종료 순서
event의 start_year는 end_year보다 늦을 수 없다.

### VR-203: 국가 존속 기간과 사건 시점
멸망한 국가가 collapsed_year 이후에 현재 국가처럼 등장하면 conflict 후보로 기록한다.

### VR-204: 인물 생존 기간과 사건 참여
인물이 사망 후 사건에 참여하면 conflict 후보로 기록한다. 단, 영혼, 회귀, 불사 설정이 명시되어 있으면 warning으로 낮춘다.

## 6. Relationship Rules
### VR-301: related id 존재성
related에 적힌 id가 content 또는 draft에 없으면 warning 또는 error다.

### VR-302: affiliation 존재성
character의 affiliation이 존재하지 않는 organization/nation이면 warning이다.

### VR-303: capital 참조
nation의 capital이 place id로 존재하지 않으면 warning이다.

### VR-304: 상호 관계 불일치
A가 B의 부모라고 되어 있는데 B 문서에서 A가 형제로 되어 있으면 conflict 후보로 기록한다.

## 7. Terminology Rules
### VR-401: 동명 entity 검사
같은 title을 가진 다른 id가 존재하면 warning이다.

### VR-402: alias 충돌
동일 alias가 여러 entity에 존재하면 warning이다.

### VR-403: glossary 용어 중복
glossary term의 title 또는 alias가 중복되면 warning이다.

## 8. Canon Integrity Rules
### VR-501: content 직접 변경 금지
accept workflow 외부에서 content가 변경되면 policy violation으로 기록한다.

### VR-502: draft는 canon 참조 가능하나 canon을 덮어쓸 수 없음
draft 내용이 기존 canon의 핵심 사실을 변경하려면 explicit override note가 필요하다.

### VR-503: retcon 표시
기존 canon을 의도적으로 수정하는 경우 frontmatter 또는 Canon Notes에 retcon 사유를 남긴다.

## 9. LLM-based Validation
정적 rule로 잡기 어려운 부분은 LLM validator가 검토한다.

검토 항목:
- 세계관 톤 불일치
- 기존 설정과 미묘한 모순
- 설정 과잉
- 이미 존재하는 설정의 중복 변형
- canon을 과도하게 확정하는 문장

LLM validator 결과는 warning 또는 conflict candidate로만 취급한다.

## 10. Accept Blocking Rules
기본적으로 다음 상태는 accept를 차단한다.
- validation status = error
- validation status = conflict
- 필수 frontmatter 누락
- id 중복
- content target path 충돌

`--force` 사용 시에도 reason이 필요하며 runs log에 기록한다.

## 11. Validation Report 형식
```markdown
# Validation Report

## Status
warning

## Summary
기존 북부 왕국의 멸망 시점과 draft의 현재 시점 표현 사이에 충돌 후보가 있음.

## Issues
### VR-203 Timeline Conflict
- Existing: nation_old_north collapsed_year = 312
- Draft: 현재도 북부 왕국이 통치 중이라고 서술
- Severity: conflict
- Recommendation: 후계 국가인지, 망명 정부인지, 역사 서술인지 명확히 할 것

## Recommendation
accept 전에 draft 수정 권장.
```
