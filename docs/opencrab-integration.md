# opencrab-integration.md

# OpenCrab Integration

## 1. 역할 정의
OpenCrab은 world-harness의 core가 아니라 adapter다. OpenCrab은 사용자 입력을 받고, world-harness CLI를 호출하고, 결과를 사용자에게 반환한다.

## 2. 책임 분리
### OpenCrab이 책임지는 것
- Telegram 또는 외부 채널 입력 수신
- 사용자 인증 및 명령 라우팅
- world-harness CLI 호출
- stdout JSON 파싱
- 결과 요약 메시지 반환
- 승인/반려 인터랙션 제공

### world-harness가 책임지는 것
- 세계관 context 로딩
- draft 생성
- canon validation
- content 승격
- graph 갱신
- runs log 작성
- 파일 권한 경계 유지

OpenCrab은 content 파일을 직접 수정하지 않는다.

## 3. 기본 호출 방식
OpenCrab은 world-harness를 subprocess로 호출한다.

```bash
world genesis "북부 제국 설정을 만들어줘" --root /workspace/world-lore --json
```

응답 예시:
```json
{
  "status": "created",
  "run_id": "20260528-001",
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
OpenCrab: world genesis 호출
world-harness: draft 생성 + validation
OpenCrab: 결과 요약 반환
사용자: /world_accept drafts/nations/northern-empire.md
OpenCrab: world accept 호출
world-harness: validate 재실행 후 content 승격
```

## 6. 메시지 포맷
### 생성 완료 메시지
```text
초안 생성 완료
- Draft: drafts/nations/northern-empire.md
- Validation: warning
- Run: 20260528-001

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
```yaml
services:
  opencrab:
    image: opencrab:latest
    volumes:
      - ./world-lore:/workspace/world-lore
    environment:
      WORLD_ROOT: /workspace/world-lore
    command: opencrab serve

  world-harness:
    image: world-harness:latest
    volumes:
      - ./world-lore:/workspace/world-lore
    working_dir: /workspace/world-lore
```

## 8. 보안 주의
- OpenCrab 컨테이너에 docker.sock을 마운트하지 않는다.
- world-lore 외부 host path를 마운트하지 않는다.
- API key는 runs log에 기록하지 않는다.
- OpenCrab은 command allowlist만 실행한다.
- 사용자 입력을 shell string으로 직접 연결하지 않는다. argv 배열로 실행한다.

## 9. 실패 처리
### CLI timeout
OpenCrab은 timeout 발생 시 run id가 생성되었는지 확인하고 status 명령을 안내한다.

### validation conflict
OpenCrab은 accept 버튼 또는 명령을 숨기지 말고, conflict 상태와 force 필요성을 명확히 보여준다.

### malformed JSON
OpenCrab은 raw stdout을 그대로 사용자에게 보내지 않고 오류 메시지와 run id를 제공한다.

## 10. 확장
MVP 이후 OpenCrab은 버튼 기반 approval, diff 미리보기, 최근 draft 목록, validation report 보기, graph 관계 조회를 제공할 수 있다.
