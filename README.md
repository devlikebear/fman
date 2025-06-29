# fman - AI-Powered File Manager

Go로 개발된, AI를 통해 로컬 파일을 지능적으로 정리하고 관리하는 강력한 CLI(Command-Line Interface) 도구입니다.

## ✨ 주요 기능

### 🔍 파일 관리
- **지능적 파일 스캔**: 권한 오류를 우아하게 처리하고 시스템 디렉토리를 자동으로 스킵
- **빠른 파일 검색**: 인덱스된 파일에 대한 고급 검색 기능
- **중복 파일 찾기**: 파일 해시 비교를 통한 중복 파일 탐지 및 제거

### 🤖 AI 기반 기능
- **AI 파일 정리**: Gemini 또는 Ollama AI를 사용한 스마트 파일 정리
- **자동화 규칙**: 사용자 정의 규칙을 통한 자동 파일 정리

### ⚡ 백그라운드 처리
- **데몬 모드**: 백그라운드에서 실행되는 데몬을 통한 비동기 작업 처리
- **작업 큐**: 대용량 디렉토리 스캔을 위한 큐 시스템

### 🌐 크로스 플랫폼
- **macOS, Linux, Windows** 지원

## 📦 설치

### 소스에서 빌드

```bash
git clone https://github.com/devlikebear/fman.git
cd fman
make build
```

### 직접 빌드

```bash
go build -o bin/fman
```

## 🚀 빠른 시작

### 1. 설정 초기화

첫 실행 시 설정 파일이 자동으로 생성됩니다:

```bash
./bin/fman scan ~/Documents
```

### 2. AI 설정

`~/.fman/config.yml` 파일에서 AI 공급자를 설정하세요:

```yaml
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

### 3. 파일 정리 시작

```bash
# AI를 사용한 파일 정리
./bin/fman organize --ai ~/Downloads

# 규칙 기반 정리
./bin/fman rules apply ~/Downloads
```

## 📖 사용법

### 🔍 파일 스캔

디렉토리를 스캔하여 파일 메타데이터를 인덱싱합니다:

```bash
# 기본 스캔
./bin/fman scan /path/to/directory

# 상세 출력과 함께 스캔
./bin/fman scan /path/to/directory --verbose

# 권한 에스컬레이션으로 스캔 (주의해서 사용)
./bin/fman scan /path/to/directory --force-sudo
```

#### 스캔 기능 특징

- **자동 스킵**: 시스템 디렉토리를 자동으로 스킵
- **권한 오류 처리**: 접근 권한이 없는 파일/디렉토리를 우아하게 스킵
- **통계 리포트**: 스캔 완료 후 상세한 통계 제공
- **해시 계산**: 중복 파일 찾기를 위한 파일 해시 계산

### 🔎 파일 검색

인덱싱된 파일을 고급 검색 조건으로 검색합니다:

```bash
# 파일명으로 검색
./bin/fman find "*.jpg"

# 크기와 날짜 조건으로 검색
./bin/fman find --size "+10MB" --modified "-30d"

# 특정 디렉토리 내에서만 검색
./bin/fman find "document" --path "/home/user/Documents"
```

### 🤖 AI 기반 파일 정리

AI를 사용하여 파일을 지능적으로 정리합니다:

```bash
# AI 제안 받기 (미리보기)
./bin/fman organize --ai /path/to/directory --dry-run

# AI 제안 적용
./bin/fman organize --ai /path/to/directory

# 특정 파일 타입만 정리
./bin/fman organize --ai /path/to/directory --type "image"
```

### 📏 규칙 기반 자동화

파일 정리 규칙을 생성하고 관리합니다:

```bash
# 예제 규칙 초기화
./bin/fman rules init

# 모든 규칙 목록 보기
./bin/fman rules list

# 규칙 적용 (미리보기)
./bin/fman rules apply --dry-run ~/Downloads

# 규칙 적용
./bin/fman rules apply ~/Downloads

# 특정 규칙 활성화/비활성화
./bin/fman rules enable screenshot-cleanup
./bin/fman rules disable temp-file-cleanup
```

### 🔄 중복 파일 관리

중복 파일을 찾고 관리합니다:

```bash
# 중복 파일 찾기
./bin/fman duplicate

# 특정 디렉토리에서 중복 파일 찾기
./bin/fman duplicate ~/Pictures

# 대화형 모드로 중복 파일 제거
./bin/fman duplicate --interactive

# 최소 크기 지정 (1MB 이상만)
./bin/fman duplicate --min-size 1048576
```

### ⚙️ 데몬 관리

백그라운드 데몬을 관리합니다:

```bash
# 데몬 시작
./bin/fman daemon start

# 데몬 상태 확인
./bin/fman daemon status

# 데몬 중지
./bin/fman daemon stop

# 데몬 재시작
./bin/fman daemon restart
```

### 📋 작업 큐 관리

백그라운드 작업을 관리합니다:

```bash
# 모든 작업 목록 보기
./bin/fman queue list

# 특정 작업 상태 확인
./bin/fman queue status <job-id>

# 작업 취소
./bin/fman queue cancel <job-id>

# 완료된 작업 정리
./bin/fman queue clear
```

## 🗂️ 데이터 저장

- **인덱스 데이터베이스**: `~/.fman/fman.db` (SQLite3)
- **설정 파일**: `~/.fman/config.yml`
- **데몬 소켓**: `~/.fman/daemon.sock`
- **PID 파일**: `~/.fman/daemon.pid`

## 🛡️ 보안 고려사항

### 스킵되는 시스템 디렉토리

**macOS:**
- `.Trash`, `.Trashes`
- `.fseventsd`, `.Spotlight-V100`
- `.DocumentRevisions-V100`, `.TemporaryItems`
- `System/Library`, `Library/Caches`

**Linux:**
- `.cache`, `.local/share/Trash`
- `proc`, `sys`, `dev`
- `tmp`, `var/tmp`

**Windows:**
- `$Recycle.Bin`
- `System Volume Information`
- `pagefile.sys`, `hiberfil.sys`

### 권한 관리

- `--force-sudo` 플래그는 신중하게 사용하세요
- 시스템 디렉토리는 기본적으로 스킵되어 안전합니다
- 권한 오류는 우아하게 처리되어 전체 스캔을 중단하지 않습니다

## 🔧 개발

### 프로젝트 구조

```
fman/
├── cmd/                 # CLI 명령어 구현
├── internal/
│   ├── ai/             # AI 공급자 인터페이스
│   ├── daemon/         # 백그라운드 데몬
│   ├── db/             # 데이터베이스 레이어
│   ├── rules/          # 규칙 엔진
│   ├── scanner/        # 파일 스캐너
│   └── utils/          # 유틸리티 함수
├── main.go             # 메인 진입점
└── Makefile           # 빌드 스크립트
```

### 빌드 및 테스트

```bash
# 빌드
make build

# 테스트 실행
make test

# 테스트 커버리지 확인
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# 린팅
golangci-lint run ./...

# 정리
make clean
```

### Docker 사용

```bash
# Docker 이미지 빌드
make docker-build

# Docker 컨테이너로 실행
make docker-run ARGS="scan /app/test_data"
```

## 🤝 기여하기

1. 이 저장소를 포크하세요
2. 기능 브랜치를 생성하세요 (`git checkout -b feature/amazing-feature`)
3. 변경사항을 커밋하세요 (`git commit -m 'feat: add amazing feature'`)
4. 브랜치에 푸시하세요 (`git push origin feature/amazing-feature`)
5. Pull Request를 생성하세요

### 커밋 메시지 규칙

[Conventional Commits](https://www.conventionalcommits.org/) 규칙을 따릅니다:

- `feat:` 새로운 기능
- `fix:` 버그 수정
- `docs:` 문서 변경
- `test:` 테스트 추가/수정
- `refactor:` 코드 리팩토링

## 📄 라이선스

MIT License - 자세한 내용은 [LICENSE](LICENSE) 파일을 참조하세요.

## 🙏 감사의 말

- [Cobra](https://github.com/spf13/cobra) - CLI 프레임워크
- [Viper](https://github.com/spf13/viper) - 설정 관리
- [SQLite](https://www.sqlite.org/) - 데이터베이스
- [Google Gemini](https://ai.google.dev/) - AI 서비스
- [Ollama](https://ollama.ai/) - 로컬 AI 서비스
