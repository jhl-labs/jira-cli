# jira-cli

`jira-cli`는 `jira.example.com` 같은 **Self-hosted Jira (Server/Data Center)** 환경에서
AI Agent와 자동화 스크립트가 이슈를 다루기 위한 가볍고 의존성 없는 CLI입니다.

> **상태: 초기 동작 버전 (alpha).**
> Go로 구현된 CLI가 동작합니다: `search`(JQL) / `get` / `create` / `update` / `comment` /
> `transition` / `assign` / `labels` / `delete` / `projects`.
> 표준 라이브러리만 사용하며(외부 의존성 0), 단일 바이너리로 크로스플랫폼 배포가 가능합니다.

> `confluence-cli`의 형제 프로젝트입니다 — 동일한 설계·인증·배포 방식을 공유합니다.
> (<https://github.com/jhl-labs/confluence-cli>)

## 왜 이 프로젝트가 필요한가

Atlassian 공식 Remote MCP(Rovo MCP)는 **Cloud 중심**이며, Server/Data Center 환경은
인증 방식·네트워크 제약·권한 모델이 다릅니다. `jira-cli`는 셀프호스팅 Jira를 AI Agent가
안전하게 호출할 수 있는 전용 CLI 계층으로, 인증(PAT/Basic)·재시도·JQL 검색·워크플로 전이를
일관되게 처리합니다.

## 설치

소개 페이지: <https://jhl-labs.github.io/jira-cli>

```bash
# 스크립트 설치 (Linux/macOS) — 최신 릴리스 바이너리를 받습니다
curl -fsSL https://jhl-labs.github.io/jira-cli/install.sh | sh
```

또는 [Releases](https://github.com/jhl-labs/jira-cli/releases/latest)에서 플랫폼별 단일
바이너리를 직접 내려받을 수 있습니다 (Linux/macOS amd64·arm64, Windows amd64).

## 소스에서 빌드

Go 1.26+ 환경이 필요합니다.

```bash
make build          # ./jira-cli 생성
make dist           # 전체 플랫폼 릴리스 바이너리 (dist/)
make test
```

## 인증 (Server/Data Center 기준)

Jira Server/Data Center는 Cloud와 인증 방식이 다릅니다.

- **Personal Access Token (권장, Jira 8.14+)** — `Authorization: Bearer <token>`
- **Basic 인증** — 사용자명 + 비밀번호/토큰

### 환경변수

| 변수 | 설명 | 예시 |
|---|---|---|
| `JIRA_BASE_URL` | 사이트 베이스 URL | `https://jira.example.com` |
| `JIRA_TOKEN` | Personal Access Token | `NjA2M...` |
| `JIRA_USER` / `JIRA_PASSWORD` | Basic 인증용 | `agent-bot` / `••••` |
| `JIRA_PROJECT` | 기본 프로젝트 키 (`search`/`create`에서 `--project` 생략 가능) | `PROJ` |

설정 우선순위: **설정 파일 < 환경변수 < 커맨드라인 플래그**.
설정 파일은 `$JIRA_CONFIG` 또는 `~/.config/jira-cli/config.json` ([예시](./config.example.json)).

## 사용법

```bash
# 검색 (JQL 또는 간편 플래그)
jira-cli search --project PROJ --status "In Progress" --max 20
jira-cli search --jql 'assignee = currentUser() AND resolution = Unresolved ORDER BY updated DESC'

# 조회
jira-cli get --key PROJ-123 --output text
jira-cli get --key PROJ-123 --description       # 설명만 출력

# 생성 (Server/DC 설명은 위키 마크업 텍스트, 마크다운/ADF 아님)
jira-cli create --project PROJ --summary "Fix login bug" --type Bug --priority High
echo "재현 절차..." | jira-cli create --summary "From stdin" --description-file -

# 수정
jira-cli update --key PROJ-123 --summary "새 제목" --priority Low

# 워크플로 전이 (인자 없으면 가능한 전이 목록)
jira-cli transition --key PROJ-123
jira-cli transition --key PROJ-123 --to "Done" --comment "1.2.0에 반영"

# 담당자 지정/해제
jira-cli assign --key PROJ-123 --assignee alice
jira-cli assign --key PROJ-123 --unassign

# 댓글 / 라벨 / 삭제 / 프로젝트
jira-cli comment --key PROJ-123 --body "확인 중입니다."
jira-cli labels --key PROJ-123 --add "backend,urgent" --remove "triage"
jira-cli delete --key PROJ-123 --yes
jira-cli projects --output text

# 에이전트용 스킬 문서 생성 (jira-skill.md)
jira-cli generate-skill claude   # 또는 codex / gemini / opencode / (생략 시 generic)
```

모든 명령은 기본 **JSON** 출력(에이전트 친화), `--output text`로 사람용 요약을 제공합니다.

## 필드 형식 주의

Server/DC에서 이슈 **설명·댓글 본문은 위키 마크업 텍스트 문자열**입니다 — 마크다운도,
Cloud의 ADF JSON도 아닙니다. 일반 문자열로 보내면 됩니다.

## 프로젝트 구조

```
.
├── main.go              # 엔트리포인트, 서브커맨드 디스패치
├── common.go            # 공통 플래그(인증/출력) + 클라이언트 생성
├── output.go            # JSON / text 출력
├── cmd_*.go             # search / get / create / update / comment / transition / ...
├── internal/
│   ├── config/          # 설정 로딩 (파일 < 환경변수 < 플래그)
│   └── jira/            # REST 클라이언트 (auth, 재시도, 에러), 이슈·전이 API
└── Makefile             # build / test / dist(크로스컴파일)
```

## 로드맵

- [x] 인증 계층 (PAT / Basic) + 설정 로딩
- [x] `search`(JQL) / `get` 읽기
- [x] `create` / `update` / `comment` / `delete` 쓰기
- [x] `transition` / `assign` 워크플로
- [x] `labels` / `projects`
- [x] JSON 출력 + 재시도(429·5xx) + httptest 단위 테스트
- [x] `generate-skill` (claude/codex/gemini/opencode/generic)
- [ ] `boards` / `sprints` (Agile API), 첨부, 워크로그, 페이지네이션 자동 순회

## 라이선스

[JHL License](./LICENSE) — 개인·교육·비상업적 용도로 자유롭게 사용/수정/배포할 수 있습니다.
**바이너리와 소스 코드 모두 상업적 사용은 금지**되며, 상업적 사용은 개발자(Licensor)의
사전 서면 허가가 있는 경우에만 허용됩니다.
