# validation-rules.md

# OpenCrabs World Tools Validation Rules

## 1. 목적
validation은 LLM이 생성한 설정을 canon으로 믿지 않고, 기존 세계관과의 충돌 후보를 탐지하기 위한 안전장치다. validator의 결과는 최종 판정이 아니라 사람이 검토할 근거다.

검증의 기준 canon은 `content/` Markdown이다. OpenCrabs DB, graph, search index는 보조 자료로 사용할 수 있지만 source of truth로 취급하지 않는다.

## 2. Validation Status
최종 status는 발견된 issue 중 가장 높은 severity를 따른다.

```text
error > conflict > warning > pass
```

### pass
명백한 구조 오류나 충돌 후보가 없다.

### warning
canon과 충돌한다고 확정할 수는 없지만 검토가 필요한 부분이 있다.

### conflict
기존 canon과 직접 충돌할 가능성이 높다. 기본 accept는 차단된다.

### error
파일 파싱 실패, 필수 필드 누락, schema 불일치 등으로 검증을 완료할 수 없다.

validation 결과의 `error`와 `conflict`는 accept 단계에서 command-level `blocked`를 뜻한다. `draft validate` 자체는 검증 결과를 반환하는 command이며, 이 문서에서 `blocked`는 policy나 validation 때문에 accept를 진행할 수 없다는 의미로 쓴다.

## 3. Structural Rules
### VR-001: frontmatter 존재
모든 active draft와 content 문서는 YAML frontmatter를 가져야 한다.

### VR-002: 필수 필드 존재
필수 필드:
- schema_version
- id
- type
- status
- title
- created_at
- updated_at

권장 필드:
- tags
- related
- relationships
- source_run_id

기존 수동 문서나 import 문서는 source_run_id가 null일 수 있다.

legacy/import 문서에 `schema_version`이 없으면 migration 전까지 warning으로 낮출 수 있다. `world-tool draft create`가 만든 새 문서에는 `schema_version` 누락을 error로 처리한다.

draft 문서의 추가 contract:
- `change_type`은 필수이며 `create`, `update`, `deprecate` 중 하나여야 한다.
- `change_type`이 `update` 또는 `deprecate`이면 `target_id`와 `retcon_reason`이 필수다.
- `update`와 `deprecate` draft는 target canon을 덮어쓰는 계약이므로 draft id와 `target_id`가 같아야 한다.
- `change_type`이 `update` 또는 `deprecate`이고 `target_id`가 canon content에 없으면 accept validation에서 conflict이며 `command_status: "blocked"`, `data.block_reason: "MISSING_TARGET"`, `data.validation_status: "conflict"`로 반환한다.
- `change_type`이 `create`이면 `target_id`와 `retcon_reason`은 null이거나 생략되어야 한다.
- missing, 빈 문자열, 잘못된 enum 값은 error다.

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

### VR-005: 날짜 형식
created_at과 updated_at은 `YYYY-MM-DD` 또는 RFC3339 timestamp여야 한다. world 안에서는 하나의 형식을 유지하는 것을 권장하며, 혼용은 warning이다.

### VR-006: draft path/type 규칙
활성 draft의 path는 type과 일치해야 한다.

- `character`, `nation`, `organization`, `place`, `event`, `timeline`, `magic`, `glossary` draft는 [schema.md](schema.md)의 type-specific draft directory 아래에 있어야 한다.
- `storylet` draft는 `drafts/storylets/` 아래에만 있어야 한다.
- non-storylet draft가 `drafts/storylets/`에 있으면 error다.
- `storylet` draft가 `drafts/storylets/` 밖에 있으면 error다.
- active draft path와 type이 다르면 accept는 blocked다.
- archive/accepted, archive/rejected, archive/deprecated는 active draft path 규칙의 예외다.

## 4. ID Rules
### VR-101: id 전역 중복 금지
`change_type: create` draft의 id가 content 전체 또는 active draft 대상과 중복되면 conflict다.

`change_type: update` 또는 `change_type: deprecate` draft는 예외적으로 기존 content id와 같아야 한다. 이 경우 `target_id`가 필수이며, draft id와 target content id가 다르면 error다.

archive/accepted, archive/rejected, archive/deprecated 아래 문서는 기본 id 중복 검사 대상에서 제외한다. 필요하면 `--include-archive` 같은 별도 점검 모드에서만 포함한다.

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
draft validation에서는 related에 적힌 id가 content 또는 active draft에 없으면 warning이다. strict mode에서는 error로 올릴 수 있다.

accept validation에서는 related id가 content에 있어야 한다. active draft에만 있는 id를 canon 문서가 참조하려고 하면 conflict다. 단, 향후 batch accept 기능에서는 같은 batch 안에서 함께 accept되는 draft target만 예외로 둘 수 있다.
`--force`라도 missing related target, missing relationship target, missing update/deprecate target, 또는 active draft에만 존재하는 target은 bypass하지 못한다.

### VR-302: canonical relationship graph
`relationships[]`는 canonical graph다. `relationships[].type`으로 authored 할 수 있는 값은 [schema.md](schema.md)의 allowlist `type` 열뿐이다. `affiliation`, `capital`, `headquarters`, `located_in`, `participants`, `locations`는 convenience field이며, validator는 relationship metadata의 domain/range를 authority로 사용해 이를 아래 authored type으로 normalize하고 graph fact로 추가한다. range-side convenience field는 내부 inverse label을 거친 normalized fact와 비교하되, inverse label 자체는 authored relationship type이 아니다.

- `character.affiliation: org_id` -> `(character_id, member_of, org_id)`
- `nation.capital: place_id` -> `(place_id, capital_of, nation_id)`
- `organization.headquarters: place_id` -> `(place_id, headquarters_of, organization_id)`
- `place.located_in: parent_id` 또는 `organization.located_in: place_id` -> `(source_id, located_in, target_id)`
- `event.participants: [participant_id]` -> `(participant_id, participates_in, event_id)`
- `event.locations: [place_id]` -> `(event_id, occurred_at, place_id)`
- `affiliation`은 strict membership shorthand다. 느슨한 관계는 `relationships[]`의 `affiliated_with`를 직접 사용한다.

convenience field는 matching explicit `relationships[]` edge가 없어도 graph fact로 normalize된다. explicit `relationships[]`는 extra note/edge metadata가 필요할 때만 작성하면 되고, convenience-normalized fact와 같은 fact로 normalize되면 dedupe된다. 편의 필드가 배열이라면 각 값은 별도 edge로 normalize된다. 잘못된 shape와 비문자열 id는 error다. convenience-normalized fact와 explicit relationship fact가 서로 다른 의미를 가지면 conflict다. duplicate exact fact는 dedupe한다.

### VR-303: relationship target 존재성
draft validation에서는 relationships[].target이 content 또는 active draft에 없으면 warning이다. strict mode에서는 error로 올릴 수 있다.

accept validation에서는 relationships[].target이 content에 있어야 한다. active draft에만 있는 target은 conflict이며 accept는 blocked다. batch accept가 생기면 같은 batch 안에서 함께 accept되는 target만 예외로 둘 수 있다.
`--force`라도 relationship target이 content가 아니라 active draft에만 있거나 아직 존재하지 않으면 bypass하지 못한다.

### VR-304: relationship type 허용값
정적 검증 가능한 relationship type은 [schema.md](schema.md)의 allowlist로 관리한다.

MVP 기본 모드에서 알 수 없는 type은 draft validation에서는 conflict, accept validation에서는 blocking conflict다. graph builder가 generic edge로 처리하는 것은 별도 `graph rebuild --allow-generic` 같은 후속 기능에서만 허용한다.

### VR-305: relationship domain/range 검사
allowlist에 정의된 domain type과 range type이 맞지 않으면 draft validation에서는 conflict, accept validation에서는 blocking conflict다.

예: `capital_of`의 domain은 place이고 range는 nation이다. 따라서 `nation.capital: place_id`는 `(place_id, capital_of, nation_id)`로 정규화한 뒤 domain/range와 비교한다. `event.participants: [character_id]`는 `(character_id, participates_in, event_id)`, `organization.headquarters: place_id`는 `(place_id, headquarters_of, organization_id)`로 정규화한다.

### VR-306: inverse/symmetric contradiction 검사
`parent_of`/`child_of`, `predecessor_of`/`successor_of`는 inverse metadata로 normalize한다. `sibling_of`, `ally_of`, `rival_of`는 symmetric edge다.

- `ally_of`와 `rival_of`는 같은 source/target pair에서 동시에 존재하면 conflict다.
- `parent_of`와 `child_of`는 같은 source/target pair에서 동시에 존재하면 conflict다.
- `predecessor_of`와 `successor_of`는 같은 source/target pair에서 동시에 존재하면 conflict다.
- symmetric edge는 한쪽 문서에만 있어도 되지만, 반대편 문서에서 정반대 관계가 발견되면 conflict 후보로 기록한다.

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
`change_type: create` draft는 기존 canon의 핵심 사실을 덮어쓸 수 없다. 기존 canon을 수정하려면 `change_type: update`와 `target_id`를 사용한다.

### VR-503: retcon 표시
기존 canon을 의도적으로 수정하는 `change_type: update` 또는 `change_type: deprecate` draft는 frontmatter `retcon_reason`을 필수로 가진다. 본문 `Canon Notes`는 보강 설명용이고, frontmatter를 대체하지 않는다. `retcon_reason`이 비어 있으면 error다.

### VR-504: archive 비활성화
archive 아래 문서는 canon 또는 pending draft로 취급하지 않는다. accepted draft 원본은 추적용 보관물이며 content 문서가 canon이다.

### VR-505: OpenCrabs index 비권위성
OpenCrabs DB나 search index와 content가 불일치하면 content를 우선한다. OpenCrabs 쪽 데이터는 재색인 대상으로 기록한다.

### VR-506: target path와 base hash
accept 대상 content path와 active draft path는 아래 규칙으로 deterministic하게 계산한다.

content type to directory:
- `character` -> `content/characters/`
- `nation` -> `content/nations/`
- `organization` -> `content/organizations/`
- `place` -> `content/places/`
- `event` -> `content/events/`
- `timeline` -> `content/timeline/`
- `magic` -> `content/magic/`
- `glossary` -> `content/glossary/`

file name:
- MVP에서는 title slug를 사용하지 않는다.
- file name은 `<id>.md`다.
- id는 이미 영문 소문자, 숫자, 언더스코어만 허용하므로 한글 title 처리와 transliteration은 필요하지 않다.

draft type to directory:
- `character` -> `drafts/characters/`
- `nation` -> `drafts/nations/`
- `organization` -> `drafts/organizations/`
- `place` -> `drafts/places/`
- `event` -> `drafts/events/`
- `timeline` -> `drafts/timeline/`
- `magic` -> `drafts/magic/`
- `glossary` -> `drafts/glossary/`
- `storylet` -> `drafts/storylets/`

`change_type: create`는 위 규칙으로 새 target path를 계산한다. 이미 다른 id의 content 문서가 같은 target path를 점유하면 conflict다.

`change_type: update`와 `change_type: deprecate`는 기존 content 문서의 path를 유지한다. rename/path move는 MVP 범위 밖이며 별도 migration command로 처리한다.
`change_type: deprecate`가 accept되면 대상 canon 문서는 `content/`에 남아 `status: deprecated`와 deprecation audit metadata를 유지한다. archive로 이동하는 것은 accepted draft 원본뿐이다.

draft diff 결과에는 draft hash, target content base hash, patch hash를 포함한다. draft accept는 해당 binding 값이 없거나 현재 상태와 다르면 `DIFF_BINDING_REQUIRED` 또는 `DIFF_BINDING_MISMATCH`로 blocked다.

### VR-507: storylet canon 금지
MVP에서 `type: storylet` active draft는 `drafts/storylets/` 아래에만 존재할 수 있다. `content/` 아래 storylet 문서 또는 `status: canon` storylet은 content validation `error`이며 accept에서는 blocked다. archive copy는 active draft scope 밖의 보관물이다.

### VR-508: force accept 비우회 경계
`--force`는 semantic, timeline, relationship conflict 후보만 우회할 수 있다. 이때도 referenced target이 모두 content에 이미 존재해야 한다.

- missing target, missing related target, missing relationship target, missing update/deprecate target은 `--force`로 우회할 수 없다.
- related target 또는 relationship target이 active draft에만 존재하면 `--force`로 우회할 수 없다.
- path/type/id/schema 불일치, id conflict, target path conflict, diff binding mismatch, storylet canon 승격 금지는 `--force`로 우회할 수 없다.
- `--force`는 오직 all-referenced-targets-in-canon 상태의 semantic/timeline/relationship conflict 후보에만 적용한다.

## 9. Migration Workflow
legacy/import 문서처럼 schema가 완전히 정리되지 않은 대상은 warning-only로 끝내지 않고 migration artifact로 묶는다.

### migration report
각 migration run은 최소 다음 산출물을 남긴다.

- `runs/<run-id>/migration.json`
- `runs/<run-id>/migration.md`
- `runs/<run-id>/migration-actions.jsonl`

report에는 다음이 포함되어야 한다.

- source 문서 목록
- warning, conflict, error, blocked 항목 분리
- 자동 수정 가능 항목과 수동 수정 필요 항목
- path move 필요 여부
- change_type/target_id/retcon_reason 보정 여부
- before/after hash 또는 diff binding 정보

### migration milestone boundary
- `schema_version` 누락 같은 legacy/import 호환 이슈는 migration warning으로 시작할 수 있다.
- `change_type` 누락, invalid enum, `target_id` 누락/불일치, `retcon_reason` 누락은 migration report에서 blocker로 분리한다.
- migration command는 report-only이며 content를 직접 변경하지 않는다. 실행 결과는 report와 artifact만 남긴다.
- migration 완료 기준은 warning이 0이 아니라, warning이 action item으로 분류되고 blocked 항목이 없어졌는지로 판단한다.

## 10. LLM-assisted Validation
정적 rule로 잡기 어려운 부분은 OpenCrabs/Codex가 후보 검토를 도울 수 있다.

검토 항목:
- 세계관 톤 불일치
- 기존 설정과 미묘한 모순
- 설정 과잉
- 이미 존재하는 설정의 중복 변형
- canon을 과도하게 확정하는 문장

LLM-assisted 결과는 warning 또는 conflict candidate로만 취급한다.

OpenCrabs/Codex는 semantic validation 후보를 생성할 수 있지만, validation status 확정과 accept 차단 여부는 `world-tool` validator가 결정한다. OpenCrabs가 반환한 structured output은 schema-valid처럼 보이더라도 신뢰하지 않고 재파싱한다.

## 11. Accept Blocking Rules
기본적으로 다음 상태는 accept를 차단한다.
- validation status = error
- validation status = conflict
- 필수 frontmatter 누락
- `change_type` 누락 또는 invalid enum
- `change_type: create`의 id 중복
- `change_type: update` 또는 `change_type: deprecate`의 target_id 누락 또는 target_id 불일치
- `change_type: update` 또는 `change_type: deprecate`의 retcon_reason 누락
- content target path 충돌
- 대상 draft가 active draft path/type 규칙을 위반하는 경우
- MVP에서 `type: storylet` draft를 content canon으로 accept하려는 경우
- diff binding 누락 또는 불일치
- relationship/related target이 content가 아니라 active draft에만 있는 경우

warning은 기본 accept를 차단하지 않는다. 단, accept reason에는 warning을 확인했다는 맥락을 남겨야 한다.

`--force` 사용 시에도 reason과 trusted approval attestation이 필요하며 runs log에 기록한다. `--force`가 우회할 수 있는 것은 semantic/timeline/relationship conflict 후보에 한정한다.

`--force`로도 우회할 수 없는 조건:
- path boundary violation
- inactive draft
- malformed markdown 또는 YAML parse failure
- 필수 frontmatter 누락
- `change_type: create` id 중복
- `change_type: update` 또는 `change_type: deprecate` target_id 누락 또는 target_id 불일치
- `change_type: update` 또는 `change_type: deprecate` retcon_reason 누락
- content target path 충돌
- diff binding 누락 또는 불일치
- storylet canon 승격
- atomic write 실패
- lock 획득 실패

## 12. Validation Report 형식
```markdown
# Validation Report

## Status
conflict

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
