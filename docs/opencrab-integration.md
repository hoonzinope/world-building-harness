# opencrab-integration.md

# OpenCrab Integration

## 1. 역할 정의
OpenCrab은 사용자 입장에서는 세계관 빌딩 하네스의 얼굴이지만, 구현 관점에서는 world-harness의 core가 아니라 host/backend + adapter다. OpenCrab은 사용자 입력을 받고, world-harness CLI를 호출하고, 결과를 사용자에게 반환한다.

권장 배포 방식은 OpenCrab server와 `world` CLI를 같은 이미지나 배포 아티팩트에 두는 것이다. 개발 환경에서는 같은 컨테이너에서 argv subprocess로 호출할 수 있고, 운영 환경에서는 OpenCrab이 target world root만 마운트한 job container를 실행하는 방식을 권장한다.

## 2. 책임 분리
### OpenCrab이 책임지는 것
- Telegram 또는 외부 채널 입력 수신
- 사용자 인증 및 명령 라우팅
- 여러 world root registry 관리
- job queue와 실행 상태 관리
- world-harness CLI 호출
- stdout JSON 파싱
- 결과 요약 메시지 반환
- 승인/반려 인터랙션 제공
- content accept 이후 search/index/cache 재색인

### world-harness가 책임지는 것
- 세계관 context 로딩
- draft 생성
- canon validation
- content 승격
- graph 갱신
- runs log 작성
- 파일 권한 경계 유지

OpenCrab은 content 파일을 직접 수정하지 않는다.
OpenCrab DB, search index, cache는 canon source of truth가 아니다. canon은 world root의 content Markdown이다.

## 3. 기본 호출 방식
OpenCrab은 같은 배포 아티팩트에 포함된 world-harness CLI를 subprocess로 호출한다. 사용자 입력은 shell string으로 조합하지 않고 argv 배열로 전달한다.

```bash
world genesis "북부 제국 설정을 만들어줘" --root /workspace/world --json
```

OpenCrab 내부 world registry 예시:
```yaml
worlds:
  ashen-continent:
    title: 잿빛 대륙
    root: /workspace/worlds/ashen-continent
  glass-sea:
    title: 유리해
    root: /workspace/worlds/glass-sea
```

응답 예시:
```json
{
  "status": "created",
  "run_id": "20260529-001",
  "draft_path": "drafts/nations/northern-empire.md",
  "validation_status": "warning",
  "summary": "북부 제국 draft가 생성되었습니다. 기존 북부 왕국 멸망 시점과 충돌 후보가 있습니다."
}
```

## 4. OpenCrab 명령 매핑
```text
/world_genesis <request>
→ world genesis <request> --json

/world_validate <draft_path>
→ world validate <draft_path> --json

/world_accept <draft_path>
→ world accept <draft_path> --json

/world_reject <draft_path> <reason>
→ world reject <draft_path> --reason <reason> --json

/world_status
→ world status --json
```

## 5. 승인 플로우
```text
사용자: /world_genesis 북부 제국 설정 만들어줘
OpenCrab: world id를 root path로 해석하고 world genesis 호출
world-harness: draft 생성 + validation
OpenCrab: 결과 요약 반환
사용자: /world_accept drafts/nations/northern-empire.md
OpenCrab: world accept 호출
world-harness: validate 재실행 후 content 승격 + draft archive
OpenCrab: content Markdown을 재색인하고 승인 상태 갱신
```

## 6. 메시지 포맷
### 생성 완료 메시지
```text
초안 생성 완료
- Draft: drafts/nations/northern-empire.md
- Validation: warning
- Run: 20260529-001

요약:
북부 산맥을 기반으로 한 군사 제국 설정입니다.

검토 필요:
기존 북부 왕국 멸망 시점과 현재 통치 표현이 충돌할 수 있습니다.

명령:
/world_accept drafts/nations/northern-empire.md
/world_validate drafts/nations/northern-empire.md
/world_reject drafts/nations/northern-empire.md <reason>
```

## 7. Docker 구성
권장 구성은 OpenCrab과 world CLI를 같은 이미지에 넣되, 실제 harness job은 target world root 하나만 마운트한 컨테이너에서 실행하는 방식이다.

```yaml
services:
  opencrab:
    image: opencrab-world:latest
    environment:
      WORLD_REGISTRY_PATH: /app/config/worlds.yaml
    command: opencrab serve
```

OpenCrab이 실행하는 job container 예시:
```bash
docker run --rm \
  --network none \
  -v /host/worlds/ashen-continent:/workspace/world \
  opencrab-world:latest \
  world genesis "북부 제국 설정을 만들어줘" --root /workspace/world --json
```

이미지 내부에는 `opencrab` server와 `world` CLI가 모두 포함되지만, job container에는 선택된 world root만 마운트한다.

## 8. 보안 주의
- OpenCrab 컨테이너에 docker.sock을 마운트하지 않는다.
- 등록된 world root 외부 host path를 마운트하지 않는다.
- API key는 runs log에 기록하지 않는다.
- OpenCrab은 command allowlist만 실행한다.
- 사용자 입력을 shell string으로 직접 연결하지 않는다. argv 배열로 실행한다.
- OpenCrab은 content를 직접 write하지 않고 `world accept` 결과만 반영한다.

## 9. 실패 처리
### CLI timeout
OpenCrab은 timeout 발생 시 run id가 생성되었는지 확인하고 status 명령을 안내한다.

### validation conflict
OpenCrab은 accept 버튼 또는 명령을 숨기지 말고, conflict 상태와 force 필요성을 명확히 보여준다.

### malformed JSON
OpenCrab은 raw stdout을 그대로 사용자에게 보내지 않고 오류 메시지와 run id를 제공한다.

## 10. 확장
MVP 이후 OpenCrab은 버튼 기반 approval, diff 미리보기, 최근 draft 목록, validation report 보기, graph 관계 조회를 제공할 수 있다.

Codex SDK를 사용할 경우 OpenCrab은 long-running Codex job을 생성하고 상태를 추적할 수 있다. 단, Codex job은 draft 생성과 semantic validation 후보 생성에만 사용하며, accept와 content write는 world-harness CLI가 수행한다.
