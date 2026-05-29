# security-boundary.md

# OpenCrabs World Tools Security Boundary

## 1. 목적
OpenCrabs/Codex가 세계관 파일을 다루는 구조에서는 명확한 보안 경계가 필요하다. 목표는 world root 밖 파일, secret, host system이 tool 호출로 노출되거나 변경되지 않게 하는 것이다.

## 2. 기본 원칙
- 세계관 파일 작업은 `world_*` dynamic tools로 수행한다.
- dynamic tools는 `world-tool` Go CLI를 호출한다.
- `world-tool`은 선택된 world root 밖 파일을 읽거나 쓰지 않는다.
- `content/`는 accept tool에서만 변경된다.
- draft 생성과 canon 승격은 분리한다.
- OpenCrabs/Codex 출력은 후보이며 tool validation을 통과해야 한다.
- 모든 write 작업은 runs log에 기록한다.

## 3. 파일 시스템 경계
### 허용 경로
선택된 world root 내부에서만 허용한다.

- content/
- drafts/
- raw/
- graph/
- schema/
- runs/
- archive/
- harness.yaml

### 금지 경로
- world root 상위 디렉토리
- 다른 world root
- 사용자의 home directory
- `.ssh/`
- `.git-credentials`
- secret이 담긴 `.env`
- Docker socket
- system path

## 4. Path Traversal 방지
사용자 입력으로 들어온 path는 반드시 normalize 후 world root 내부인지 검사한다.

금지 예시:

```text
../../.ssh/id_rsa
~/private.txt
/var/run/docker.sock
```

## 5. Dynamic Tool Boundary
나쁜 tool:

```toml
[[tools]]
name = "world_exec_shell"
executor = "shell"
command = "{{command}}"
```

좋은 tool:

```toml
[[tools]]
name = "world_accept_draft"
executor = "shell"
command = "world-tool accept draft --world {{world_id}} --draft {{draft_path}} --reason {{reason}} --json"
```

tool은 의미 단위 작업이어야 하며 shell 권한을 넓게 열지 않는다.

## 6. Network Boundary
`world-tool` MVP는 임의 네트워크 요청을 수행하지 않는다.

OpenCrabs provider가 Codex/OpenAI/Claude/Gemini 등으로 네트워크를 사용하는 것은 OpenCrabs 설정의 책임이다. world 파일 작업 tool은 외부 URL fetch, repo clone, arbitrary curl을 수행하지 않는다.

## 7. Secret Handling
API key, OAuth token, bot token은 다음 위치에 저장하지 않는다.

- runs/
- drafts/
- content/
- validation report
- graph/
- world root 내부

OpenCrabs credential은 OpenCrabs의 credential store나 별도 secret mount로 관리한다. `world-tool`은 provider API key를 필요로 하지 않는다.

## 8. LLM Output Boundary
OpenCrabs/Codex는 다음을 직접 수행하지 않는다.

- content 직접 수정
- accept 우회
- validation 생략
- graph 확정 업데이트
- 다른 world root 접근
- OpenCrabs 보안 설정 변경

LLM은 후보 markdown과 판단을 만들 수 있지만, 파일 상태 변경은 `world-tool`이 수행한다.

## 9. Approval Boundary
draft가 content로 승격되려면 `world_accept_draft`를 통과해야 한다.

기본 차단 조건:
- validation error
- validation conflict
- id 중복
- content target path 충돌
- required field 누락
- draft가 active drafts/ 밖에 있음

force accept는 가능하지만 reason이 필수다.

## 10. Docker Boundary
권장 컨테이너 실행 원칙:
- per-world tool container에는 선택된 world root 하나만 마운트한다.
- 여러 world root를 한 컨테이너에 동시에 마운트하지 않는다.
- docker.sock을 마운트하지 않는다.
- read-only root filesystem을 고려한다.
- 네트워크 권한은 최소화한다.
- 컨테이너 유저는 root가 아닌 전용 유저를 사용한다.

예시:

```bash
docker run --rm \
  --user 1000:1000 \
  --network none \
  --read-only \
  --tmpfs /tmp \
  -v /host/worlds/ashen-continent:/workspace/world \
  world-tool:latest \
  world-tool validate draft --root /workspace/world --draft drafts/nations/northern-empire.md --json
```

## 11. Audit Log
모든 write tool은 다음을 기록한다.

- run id
- tool name
- input summary
- modified files
- validation status
- actor
- timestamp
- force 여부와 reason

## 12. 위험 시나리오
### LLM이 canon을 오염시키는 경우
방어:
- content 직접 write tool 제공 금지
- accept tool 강제
- validation report 생성

### path traversal
방어:
- path normalize
- root 내부 검사
- symlink resolution

### dynamic tool이 너무 넓은 권한을 갖는 경우
방어:
- `world_exec_shell` 금지
- 의미 단위 `world_*` tools만 제공
- command template에서 path와 인자를 제한

### runs log에 secret이 남는 경우
방어:
- secret masking
- env dump 금지
- 에러 메시지 scrub
