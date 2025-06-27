# fman (File Manager)

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go)
![Cobra](https://img.shields.io/badge/Cobra-CLI-blue?style=for-the-badge)
![SQLite](https://img.shields.io/badge/SQLite-07405E?style=for-the-badge&logo=sqlite&logoColor=white)
![AI Powered](https://img.shields.io/badge/AI%20Powered-FF69B4?style=for-the-badge&logo=openai&logoColor=white)

`fman`은 Go로 개발된 강력한 CLI(Command-Line Interface) 도구로, AI를 활용하여 로컬 파일을 지능적으로 정리하고 관리합니다.

## ✨ 주요 기능

*   **설정 관리**: `~/.fman/config.yml` 파일을 통해 AI 공급자(Gemini/Ollama) 및 API 키 설정
*   **수동 인덱싱**: `fman scan <디렉토리>` 명령어로 특정 디렉토리의 파일 메타데이터를 SQLite DB에 저장
*   **AI 기반 정리**: `fman organize --ai <디렉토리>` 명령어로 AI에게 파일 정리 제안을 받고, 사용자 확인 후 실행
*   **기본 검색**: `fman find <파일명>` 명령어로 인덱싱된 파일 검색

## 🚀 시작하기

### 📋 전제 조건

*   [Go 1.22+](https://go.dev/doc/install)
*   [Docker](https://docs.docker.com/get-docker/) (선택 사항, Docker를 사용하여 실행할 경우)

### 🛠️ 설치

1.  **저장소 클론**: 
    ```bash
    git clone https://github.com/devlikebear/fman.git
    cd fman
    ```

2.  **의존성 설치 및 빌드**: 
    ```bash
    go mod tidy
    go build -o fman .
    ```
    이제 `fman` 실행 파일이 현재 디렉토리에 생성됩니다.

### ⚙️ 설정

`fman`을 처음 실행하면 `~/.fman/config.yml` 파일이 자동으로 생성됩니다. 이 파일을 열어 AI 공급자(Gemini 또는 Ollama)를 선택하고 필요한 API 키를 설정해야 합니다.

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

### 💡 사용법

#### 파일 스캔

특정 디렉토리의 파일 메타데이터를 인덱싱합니다. 이 과정에서 파일의 경로, 이름, 크기, 수정일, 그리고 내용 해시가 데이터베이스에 저장됩니다.

```bash
./fman scan /path/to/your/directory
```

#### 파일 검색

인덱싱된 파일 중에서 이름 패턴에 일치하는 파일을 검색합니다.

```bash
./fman find <file-name-pattern>
# 예시:
# ./fman find report
```

#### AI 기반 파일 정리

AI의 제안을 받아 파일을 정리합니다. AI는 지정된 디렉토리의 파일 목록을 분석하여 `mv` 또는 `mkdir`와 같은 셸 명령어를 제안합니다. 사용자의 확인 없이는 실행되지 않습니다.

```bash
./fman organize --ai /path/to/your/directory
# 예시:
# ./fman organize --ai ~/Downloads
```

### 🐳 Docker로 실행

Docker를 사용하여 `fman`을 컨테이너 환경에서 실행할 수 있습니다. 이는 시스템에 Go를 설치하지 않고도 `fman`을 사용하거나, 격리된 환경에서 테스트할 때 유용합니다.

1.  **Docker 이미지 빌드**: 
    ```bash
    make docker-build
    ```

2.  **Docker 컨테이너 실행**: 
    `docker-run` 명령어는 `~/.fman` 디렉토리와 현재 디렉토리의 `test_data` 디렉토리를 컨테이너에 마운트하여 설정 파일과 스캔할 데이터를 공유할 수 있도록 합니다.
    ```bash
    make docker-run ARGS="scan /app/test_data"
    # 또는 AI 정리:
    # make docker-run ARGS="organize --ai /app/test_data"
    ```

## 🧪 테스트

단위 테스트를 실행하고 코드 커버리지를 확인할 수 있습니다.

```bash
make test
```

## 🧹 클린업

빌드된 실행 파일과 테스트 커버리지 파일을 삭제합니다.

```bash
make clean
```

## 🤝 기여

기여를 환영합니다! 버그 리포트, 기능 제안, 풀 리퀘스트 등 어떤 형태의 기여든 좋습니다. 기여하기 전에 `CONTRIBUTING.md` (아직 없음) 파일을 참조해 주세요.

## 📄 라이선스

이 프로젝트는 MIT 라이선스 하에 배포됩니다. 자세한 내용은 `LICENSE` (아직 없음) 파일을 참조하세요.
