# commands.md

# World-Building Harness Commands

## 1. 개요
world-harness CLI는 모든 adapter가 호출하는 공통 인터페이스다. OpenCrab, Telegram, Codex, Claude, Web UI는 직접 파일을 수정하지 않고 CLI 명령을 호출한다.

`content/` Markdown은 canon source of truth다. OpenCrab은 여러 world root를 관리하고 같은 배포 단위에 포함된 `world` CLI를 argv subprocess로 호출할 수 있지만, canon write 규칙은 CLI/core가 집행한다.

## 2. 기본 명령 구조
```bash
world <command> [args] [flags]
```

공통 플래그:
```bash
--root <path>          world root path
--json                 machine-readable JSON output
--dry-run              파일 변경 없이 실행 계획과 diff만 출력
--run-id <id>          기존 run에 후속 event/artifact 추가
--verbose              상세 로그 출력
```

OpenCrab은 world id를 자체 registry에서 root path로 해석한 뒤 항상 `--root`를 명시해 CLI를 호출한다.

## 3. init
```bash
world init --root ./world-lore
```

### 동작
- 지정한 world root에 기본 디렉토리 생성
- harness.yaml 생성
- prompts 기본 파일 생성
- schema 기본 파일 생성
- graph 초기 파일 생성

`world-lore`는 예시 이름일 뿐이며 harness 코드 저장소가 아니다. 여러 세계관은 각각 독립적인 world root를 가진다.

## 4. genesis
```bash
world genesis "북부 제국 설정을 만들어줘"
world genesis "마법 체계 초안 작성" --type magic
world genesis "붉은 수도의 건국 신화" --type event --related nation_red_capital
```

### 동작
- 자연어 요청 분석
- 관련 context 로딩
- draft markdown 생성
- validation 실행
- runs log 저장

### 출력
```json
{
  "status": "created",
  "draft_path": "drafts/nations/northern-empire.md",
  "validation_status": "warning",
  "run_id": "20260529-001"
}
```

## 5. validate
```bash
world validate drafts/nations/northern-empire.md
world validate drafts/nations/northern-empire.md --level strict
```

### 동작
- frontmatter 검사
- id 중복 검사
- timeline 충돌 검사
- relationship 충돌 검사
- validation report 생성

### 출력 상태
- pass
- warning
- conflict
- error

## 6. accept
```bash
world accept drafts/nations/northern-empire.md
world accept drafts/nations/northern-empire.md --commit
world accept drafts/nations/northern-empire.md --force --reason "의도적인 역사 왜곡 설정"
```

### 동작
- validate 재실행
- conflict 시 중단
- draft를 content로 승격
- content frontmatter status를 canon으로 갱신
- accepted draft 원본을 archive/accepted/로 이동
- graph 업데이트
- runs log 업데이트
- OpenCrab index/cache가 재색인할 수 있는 결과 JSON 반환
- optional git commit

## 7. reject
```bash
world reject drafts/nations/northern-empire.md --reason "기존 제국 설정과 중복"
```

### 동작
- draft를 archive/rejected/로 이동
- runs log에 반려 사유 기록

## 8. storylet
```bash
world storylet "잿빛 제국과 남부 도시국가의 무역 갈등"
world storylet --related nation_ashen_empire --related city_south_gate
```

### 동작
- 기존 canon을 기반으로 짧은 사건 후보 생성
- drafts/storylets/에 저장
- canon 변경 없음

## 9. export
```bash
world export raw/note-001.md --type character
world export raw/note-002.md --type nation --out drafts/nations/foo.md
```

### 동작
- raw note를 schema에 맞는 Markdown으로 변환
- frontmatter 생성
- 관련 링크 후보 생성

## 10. graph
```bash
world graph rebuild
world graph check
world graph show character_aria
```

### 동작
- content 기준 graph 재생성
- orphan link 검사
- 특정 entity 관계 출력

## 11. status
```bash
world status
world status --drafts
world status --runs
```

### 출력
- 최근 draft 목록
- validation 상태
- 최근 run 목록
- pending approval 목록

## 12. adapter-friendly JSON 출력
OpenCrab 같은 adapter는 `--json`을 기본으로 사용한다.

```bash
world genesis "북부 제국 설정" --json
```

Adapter는 stdout JSON만 해석하고 내부 파일 구조를 직접 변경하지 않는다. OpenCrab은 accept 이후 content Markdown을 다시 읽어 자체 DB나 search index에 흡수할 수 있지만, 그 DB/index는 canon source of truth가 아니다.
