# security-boundary.md

# World-Building Harness Security Boundary

## 1. 목적
world-harness는 LLM이 파일을 읽고 쓰는 구조이므로 명확한 보안 경계가 필요하다. 목표는 세계관 저장소 외부 파일, 환경변수, 인증 토큰, 호스트 시스템이 LLM 또는 adapter에 의해 오염되거나 유출되지 않도록 하는 것이다.

## 2. 기본 원칙
- 하네스는 선택된 world root 밖 파일을 읽거나 쓰지 않는다.
- content는 accept workflow에서만 변경된다.
- draft 생성과 canon 승격은 분리한다.
- OpenCrab은 host/backend + adapter이며 content를 직접 수정하지 않는다.
- LLM 출력은 신뢰하지 않고 validator와 approval을 거친다.
- 모든 write 작업은 runs log에 기록한다.
- content Markdown은 canon source of truth다.

## 3. 파일 시스템 경계
### 허용 경로
선택된 world root 내부에서만 허용한다.

- content/
- drafts/
- raw/
- graph/
- prompts/
- schema/
- runs/
- archive/
- harness.yaml

### 금지 경로
- world root 상위 디렉토리
- 다른 world root
- 사용자의 home directory
- .ssh/
- .git-credentials
- .env 중 민감 값
- Docker socket
- system path

## 4. Path Traversal 방지
사용자 입력으로 들어온 파일 경로는 반드시 normalize 후 world root 내부인지 검사한다.

금지 예시:
```text
../../.ssh/id_rsa
~/private.txt
/var/run/docker.sock
```

## 5. Command Execution Boundary
world-harness가 외부 명령을 실행할 경우 allowlist를 사용한다.

허용 후보:
- git diff
- git status
- git add
- git commit
- world-harness 내부 subcommand

금지 후보:
- rm -rf
- curl arbitrary URL
- docker run
- docker socket 접근
- shell eval
- 사용자 입력을 그대로 shell에 붙이는 방식

OpenCrab은 shell string 대신 argv 배열로 실행해야 한다.
OpenCrab과 world CLI가 같은 컨테이너에 있더라도 이 경계는 유지한다.

## 6. Network Boundary
MVP에서는 world-harness core가 임의 네트워크 요청을 수행하지 않는다.

허용:
- 설정된 LLM provider API 호출

금지:
- LLM이 임의 URL을 요청하게 하는 기능
- 사용자가 입력한 URL을 자동 fetch
- 외부 repo 자동 clone

필요한 경우 explicit allowlist를 둔다.

## 7. Secret Handling
API key, bot token, OAuth token은 다음 위치에 저장하지 않는다.
- runs/
- drafts/
- content/
- validation report
- graph/

환경변수를 로그로 출력하지 않는다. 에러 메시지에도 secret 값을 포함하지 않는다.

## 8. LLM Output Boundary
LLM은 다음을 직접 수행할 수 없다.
- content 직접 수정
- accept 자동 실행
- graph 확정 업데이트
- git commit
- OpenCrab 설정 변경
- harness.yaml 보안 옵션 변경
- 다른 world root 접근

LLM은 후보 결과만 생성한다.
Codex SDK runner도 동일한 제한을 받는다.

## 9. Approval Boundary
draft가 content로 승격되려면 accept workflow를 통과해야 한다.

기본 차단 조건:
- validation error
- validation conflict
- id 중복
- content path 충돌
- required field 누락

force accept는 가능하지만 reason이 필수다.

## 10. Docker Boundary
권장 컨테이너 실행 원칙:
- harness job container에는 선택된 world root 하나만 마운트한다.
- 여러 world root를 한 컨테이너에 동시에 마운트하지 않는다.
- docker.sock을 마운트하지 않는다.
- read-only root filesystem을 고려한다.
- OpenCrab과 world CLI는 같은 이미지에 포함할 수 있지만, 운영 환경에서는 OpenCrab이 per-world job container로 CLI를 실행한다.
- 네트워크 권한은 최소화한다.
- 컨테이너 유저는 root가 아닌 전용 유저를 사용한다.

예시:
```yaml
docker run --rm \
  --user 1000:1000 \
  --network none \
  --read-only \
  --tmpfs /tmp \
  -v /host/worlds/ashen-continent:/workspace/world \
  opencrab-world:latest \
  world validate drafts/nations/northern-empire.md --root /workspace/world --json
```

## 11. Audit Log
모든 write workflow는 다음을 기록한다.
- run id
- command
- input summary
- modified files
- validation status
- actor
- timestamp
- force 여부와 reason

## 12. 위험 시나리오
### LLM이 canon을 오염시키는 경우
방어:
- content 직접 write 금지
- accept workflow 강제
- validation report 생성

### 사용자 입력이 path traversal을 시도하는 경우
방어:
- path normalize
- root 내부 검사

### OpenCrab이 너무 많은 권한을 갖는 경우
방어:
- adapter 역할 제한
- command allowlist
- subprocess argv 실행
- content write는 world CLI만 수행

### runs log에 secret이 남는 경우
방어:
- secret masking
- env dump 금지
- 에러 메시지 scrub

## 13. 보안 관련 비목표
MVP에서 완전한 sandbox 보안, RBAC, multi-user permission model, remote execution isolation은 구현하지 않는다. 단, 나중에 확장 가능하도록 core와 adapter를 분리한다.
