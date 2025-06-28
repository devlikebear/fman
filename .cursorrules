# 개발 인스트럭션 (v1.1)

## 1. 프로젝트 기획서

### 1.1. 프로젝트명

`fman` (File Manager)

### 1.2. 한 줄 요약

Go로 개발된, AI를 통해 로컬 파일을 지능적으로 정리하고 관리하는 강력한 CLI(Command-Line Interface) 도구

### 1.3. 타겟 사용자

*   터미널 환경에 익숙한 개발자 및 파워 유저
*   수많은 파일을 효율적으로 정리하고 싶은 모든 사용자

### 1.4. 핵심 기능

#### MVP (Minimum Viable Product) 범위

*   **설정 관리**: `~/.fman/config.yml` 파일을 통해 AI 공급자(Gemini/Ollama)와 API 키 등 설정 관리
*   **수동 인덱싱**: `fman scan <디렉토리>` 명령어로 특정 디렉토리의 파일 메타데이터를 SQLite DB에 저장
*   **AI 기반 정리**: `fman organize --ai <디렉토리>` 명령어로 AI에게 파일 정리를 요청하고, 제안을 받아 사용자 확인 후 실행
*   **기본 검색**: `fman find <파일명>` 명령어로 인덱싱된 파일 검색

#### 최종 버전 목표 기능

*   **실시간 인덱싱**: 데몬(Daemon)을 통해 파일 시스템 변경을 실시간으로 감지하여 자동으로 인덱스 업데이트
*   **고급 검색**: 파일 크기, 수정 날짜, 내용 기반(AI 활용) 검색
*   **중복 파일 찾기**: 파일 해시(hash)를 비교하여 중복 파일 탐색 및 제거 제안
*   **규칙 기반 정리**: 사용자가 직접 "30일 지난 스크린샷은 'old-screenshots' 폴더로 이동"과 같은 규칙을 설정하는 기능

-----

## 2. MVP 태스크 계획서

아래 태스크를 순서대로 하나씩 완료하여 MVP를 개발합니다.

1.  **Task 1: 프로젝트 구조 설정**
    *   Go 모듈 초기화: `go mod init github.com/user/fman`
    *   CLI 프레임워크 `cobra`를 사용하여 기본 명령어 구조 생성
        *   `fman` (root)
        *   `fman scan <directory>`
        *   `fman organize <directory>`
        *   `fman find <pattern>`
    *   의존성 정리: `go mod tidy`

2.  **Task 2: 설정 관리 기능 구현**
    *   `viper` 라이브러리를 사용하여 `~/.fman/config.yml` 파일 읽기/쓰기 기능 구현
    *   설정 파일이 없으면 아래 **기본 템플릿**으로 자동 생성하는 로직 추가
        ```yaml
        # ~/.fman/config.yml
        # 사용할 AI 공급자를 선택합니다. (gemini 또는 ollama)
        ai_provider: "gemini"

        gemini:
          # Gemini API 키를 입력하세요.
          api_key: "YOUR_GEMINI_API_KEY"
          # 사용할 모델을 지정합니다.
          model: "gemini-1.5-flash"

        ollama:
          # Ollama 서버 주소를 입력하세요.
          base_url: "http://localhost:11434"
          # 사용할 모델을 지정합니다.
          model: "llama3"
        ```

3.  **Task 3: AI Provider 인터페이스 설계**
    *   `internal/ai/provider.go` 파일에 `AIProvider` 인터페이스 정의
    *   **`context.Context`를 첫 번째 인자로 추가**하여 타임아웃, 취소 등 컨텍스트 제어 기능을 지원하도록 개선합니다.
        ```go
        package ai

        import "context"

        type AIProvider interface {
            SuggestOrganization(ctx context.Context, filePaths []string) (string, error)
        }
        ```

4.  **Task 4: AI Provider 구현 (Ollama & Gemini)**
    *   `internal/ai/ollama.go`: Ollama의 REST API와 통신하는 `OllamaProvider` 구현
    *   `internal/ai/gemini.go`: Gemini API Go 클라이언트를 사용하는 `GeminiProvider` 구현

5.  **Task 5: 데이터베이스 레이어 설정**
    *   `internal/db/database.go` 파일 생성
    *   `mattn/go-sqlite3` 드라이버와 `sqlx` 라이브러리를 사용하여 DB 연결 및 초기화
    *   **명확한 `files` 테이블 스키마**를 정의하고, `file_hash` 필드를 추가하여 향후 중복 파일 찾기 기능을 대비합니다.
        ```sql
        CREATE TABLE IF NOT EXISTS files (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            path TEXT NOT NULL UNIQUE,
            name TEXT NOT NULL,
            size INTEGER NOT NULL,
            modified_at TIMESTAMP NOT NULL,
            indexed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
            file_hash TEXT
        );
        ```
    *   파일 정보를 DB에 저장(Upsert), 조회, 삭제하는 함수 구현

6.  **Task 6: `scan` 명령어 로직 구현**
    *   `cmd/scan.go` 파일에 로직 구현
    *   지정된 디렉토리를 재귀적으로 탐색하며 각 파일의 메타데이터(경로, 크기, 수정일)와 해시(SHA-256)를 계산
    *   Task 5에서 만든 DB 함수를 통해 정보 저장 (이미 존재하면 업데이트)

7.  **Task 7: `organize --ai` 명령어 로직 구현**
    *   `cmd/organize.go` 파일에 로직 구현
    *   설정 파일(`config.yml`)을 읽어 활성화된 AI Provider(Task 4) 선택
    *   지정된 디렉토리의 파일 목록을 AI Provider에게 전달하여 정리 계획(예: `mv "image.jpg" "images/photo_2024.jpg"`)을 텍스트로 제안받음
    *   AI의 제안(실행할 셸 스크립트)을 사용자에게 보여주고, 실행 여부(y/n)를 확인
    *   사용자가 동의하면 제안된 명령을 `os/exec`를 통해 실행

8.  **Task 8: `find` 명령어 로직 구현**
    *   `cmd/find.go` 파일에 로직 구현
    *   사용자 입력(파일명 패턴)을 받아 SQLite DB에 `WHERE name LIKE ?` 쿼리를 실행하여 결과를 테이블 형식으로 출력

-----

## 3. 개발 워크플로우

**각 태스크는 아래의 워크플로우를 반드시 준수하여 개발합니다.**

1.  **STEP 1: 태스크 선정 및 계획**
    *   MVP 태스크 계획서에서 현재 진행할 태스크를 하나 선택합니다.
    *   해당 태스크를 구현하기 위해 어떤 파일을 수정하고 어떤 함수를 작성해야 할지 간략하게 계획을 세웁니다.

2.  **STEP 2: 코드 개발**
    *   계획에 따라 실제 Go 코드를 작성합니다.

3.  **STEP 3: 단위 테스트 작성**
    *   작성한 코드에 대한 단위 테스트(`_test.go`)를 작성합니다.
    *   AI API 호출, DB 접근 등 외부 의존성이 있는 부분은 **Mocking**을 사용하여 테스트합니다. (`testify/mock` 라이브러리 사용 권장)

4.  **STEP 4: 테스트 실행 및 커버리지 확인**
    *   터미널에서 아래 명령어를 실행하여 모든 테스트를 통과하는지 확인합니다.
        ```sh
        go test ./...
        ```
    *   테스트 커버리지를 측정하고 **전체 커버리지가 70% 이상**인지 확인합니다.
        ```sh
        go test -coverprofile=coverage.out ./...
        go tool cover -func=coverage.out
        ```
    *   커버리지가 70% 미만이면, 테스트가 부족한 부분을 찾아 테스트 케이스를 보강합니다. (STEP 3으로 복귀)

5.  **STEP 5: 코드 린팅 (추가)**
    *   `golangci-lint`를 사용하여 코드 스타일과 잠재적인 오류를 점검합니다.
        ```sh
        # 설치 (아직 안했다면): go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
        golangci-lint run ./...
        ```
    *   린터가 지적하는 문제를 모두 해결합니다.

6.  **STEP 6: Git 커밋**
    *   테스트와 린팅을 모두 통과했다면, 변경 사항을 Git에 커밋합니다.
    *   커밋 메시지는 **Conventional Commits** 양식([https://www.conventionalcommits.org](https://www.conventionalcommits.org))을 따릅니다.
        *   `feat(cli): implement 'scan' command to index directory`
        *   `fix(db): correct sql query for finding files`
        *   `test(ai): add unit tests for ollama provider`
        *   `docs(readme): update usage instructions`

7.  **STEP 7: 다음 태스크로 이동**
    *   하나의 태스크가 완전히 완료되었습니다. MVP 태스크 계획서의 다음 태스크로 이동하여 **STEP 1부터 다시 반복**합니다.
