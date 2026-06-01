# AGENTS.md - 세계관 설정 작업 지침

이 레포에서 세계관 설정을 읽고, 정리하고, 문서화하는 에이전트를 위한 실무 가이드입니다. 기존 변경은 함부로 되돌리지 말고, 현재 작업 위에 조심스럽게 덧붙이십시오.

## 기본 원칙

- 별도 지시가 없으면 읽기, 쓰기, 작업은 서브 에이전트를 우선 사용합니다.
- 새 설정은 먼저 `raw/`에 원본 메모를 만들거나 갱신한 뒤, `drafts/`에서 구조화합니다.
- `raw/`는 원본 설정 메모 저장소입니다. 공개 위키 초안은 여기서 직접 완성하지 말고 `drafts/`로 옮겨 정리합니다.
- 도시와 시스템은 얇은 아이디어보다 실제로 작동하는 모델, 의존망, 주기, 병목, 실패 반응까지 포함해 확장합니다.

## 설정 플로우

1. `raw/`의 메모와 기존 `drafts/` 문서를 함께 읽어 현재 상태를 파악합니다.
2. 새 설정이 생기면 `raw/`에 원본 메모를 먼저 보강하고, 그다음 `drafts/` 문서로 옮겨 구조화합니다.
3. 관련 문서는 반드시 링크로 연결하고, 필요하면 `drafts/index.md`도 갱신합니다.
4. 문서 간 충돌이 보이면 기존 문서의 용어와 구조를 우선 맞추고, 새 문서는 그 위에 얹습니다.

## 반복 작업 워크플로우

세계관 보강 요청을 반복 수행할 때는 아래 순서를 기본값으로 삼습니다.

1. 보강 후보 문서를 확인합니다.
   - 현재 `main` 브랜치 상태와 기존 변경을 먼저 확인합니다.
   - `raw/`, `drafts/`, `contents/`의 관련 문서를 함께 읽어 이미 충분히 정리된 요소와 빈틈을 구분합니다.
2. 웹검색으로 세계관 빌딩 체크리스트나 관련 참고 요소를 확인합니다.
   - 외부 자료는 설정을 그대로 가져오기보다, 정치·경제·사회·문화·예술·종교·철학 중 빠진 축을 점검하는 용도로 씁니다.
   - 최신성이나 정확성이 필요한 항목은 검색 결과를 확인한 뒤 작업합니다.
3. `raw/`, `drafts/`, `contents/`를 차례로 보강합니다.
   - 새 설정은 먼저 `raw/`에 원본 메모로 남깁니다.
   - 구조화된 내용은 `drafts/`에 반영하고, 공개 가능한 문서는 `contents/`에도 함께 승격합니다.
   - 밝은/중간 톤 요소는 체제를 이기는 해결책이 아니라, 사람이 판정·물량·채권·의례·위험값으로 너무 빨리 닫히지 않게 하는 낮은 마찰로 둡니다.
4. 문서를 점검합니다.
   - `drafts`와 `contents` 링크, 엔티티 추출, 타임라인, 공백 오류를 확인합니다.
   - `contents`에는 `status: draft`가 남지 않게 점검합니다.
5. 요청이 있을 경우 `main`에 커밋하고 푸시합니다.
   - 검증을 통과한 변경만 staging합니다.
   - 커밋 메시지는 보강한 설정의 핵심을 짧게 적습니다.
6. `contents`가 바뀌었으면 Quartz를 재빌드하고 재기동한 뒤 URL을 확인합니다.
   - 현재 Quartz 컨테이너와 nginx `/world/` 프록시 설정을 확인합니다.
   - Quartz build를 실행하고 오류 문자열을 점검합니다.
   - `quartz-site` 컨테이너를 재기동하거나 재생성합니다.
   - `https://urrrm.com/world/cities/seongun-si`와 변경된 문서 URL이 200으로 열리는지 확인하고, 새 문구가 렌더되는지 확인합니다.

## 네이밍과 설정 고정

- `백일몽`은 공식 draft 설정으로 되살리지 않습니다.
- 도시명, 시스템명, 문서명은 기존 인덱스와 slug 체계를 유지합니다.

## 문서 배치

- 공통 시스템 문서는 `drafts/systems/`에 둡니다.
- 도시별 문서는 `drafts/cities/`에 둡니다.
- 도시 간 연결, 운영 모델, 시간 주기, 병목, 임계점 같은 횡단 설정은 별도 시스템 문서로 만들고, 도시 문서에는 짧게 링크만 둡니다.
- 이상현상과 도시괴담은 `drafts/anomalies/`에 둡니다.
- 모든 새 설정은 관련 도시, 시스템, 이상현상, 세력 문서와 서로 연결합니다.

## 문서 스타일

- 한국어 중심으로 작성합니다.
- frontmatter는 유지하고, 필요한 필드만 임의로 지우지 않습니다.
- 표보다 설명문과 절차가 흐름 있게 드러나도록 씁니다.
- 도시와 시스템은 인물이나 장면보다 먼저, 도시가 스스로 어떻게 작동하는지 보이게 씁니다.
- 이상현상 문서는 규칙, 단계, 대응, 실패 반응이 보이도록 정리합니다.

## 검증

문서 변경 후에는 아래 명령을 순서대로 실행합니다.

```bash
python3 tools/check-links.py --content-dir drafts
python3 tools/extract-entities.py --content-dir drafts >/tmp/world-lore-draft-entities.json
python3 tools/check-timeline.py --content-dir drafts
python3 tools/check-links.py --content-dir contents
python3 tools/extract-entities.py --content-dir contents >/tmp/world-lore-content-entities.json
python3 tools/check-timeline.py --content-dir contents
rg -n "status: draft" contents || true
git diff --check
```

## Quartz 공개 확인

`contents/`가 변경되면 커밋과 푸시 뒤에 아래 순서로 공개 위키를 갱신하고 확인합니다.

```bash
rg -n "location /world/|location = /world|proxy_pass http://host.docker.internal:8088/" /Users/hoonzi/Documents/docker_v/nginx/default.conf
docker compose ps
docker compose --profile build run --rm quartz-build
docker compose up -d --force-recreate quartz-site
curl -I https://urrrm.com/world/cities/seongun-si
curl -I https://urrrm.com/world/<changed-path>
```

- nginx는 `/Users/hoonzi/Documents/docker_v/nginx/default.conf`의 `/world/` 프록시 설정을 기준으로 확인합니다.
- Quartz build 로그에서 `invalid date`, `Error`, `Failed` 같은 오류 문자열이 없는지 확인합니다.
- 변경된 문서 URL은 HTTP 200뿐 아니라 새 문구가 HTML에 렌더되는지도 확인합니다.

## Git 작업

- 먼저 `main` 브랜치 상태와 기존 변경을 확인합니다.
- 다른 작업자의 변경은 되돌리지 않습니다.
- 사용자가 명시적으로 요청할 때만 `git commit`과 `git push`를 합니다.
- 요청이 있어 커밋이나 푸시를 진행할 때는 `검증 -> staging -> commit -> push -> final directive` 순서를 지킵니다.

## 유지 기준

- 기존 문서와의 링크를 끊지 말고, 새 문서는 반드시 관련 위치로 연결합니다.
- `drafts/index.md`는 새 문서가 생기거나 분류가 바뀔 때 함께 점검합니다.
- 작업 중 발견한 설정 충돌은 숨기지 말고 문서에 드러내어 정리합니다.
