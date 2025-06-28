# fman - AI-Powered File Manager

Go로 개발된, AI를 통해 로컬 파일을 지능적으로 정리하고 관리하는 강력한 CLI(Command-Line Interface) 도구입니다.

## 주요 기능

- **지능적 파일 스캔**: 권한 오류를 우아하게 처리하고 시스템 디렉토리를 자동으로 스킵
- **AI 기반 파일 정리**: Gemini 또는 Ollama AI를 사용한 스마트 파일 정리
- **빠른 파일 검색**: 인덱스된 파일에 대한 빠른 검색
- **크로스 플랫폼 지원**: macOS, Linux, Windows 지원

## 설치

```bash
git clone https://github.com/devlikebear/fman.git
cd fman
go build -o bin/fman
```

## 사용법

### 1. 파일 스캔

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

- **자동 스킵**: 시스템 디렉토리 (.Trash, .fseventsd, System/Library 등)를 자동으로 스킵
- **권한 오류 처리**: 접근 권한이 없는 파일/디렉토리를 우아하게 스킵
- **통계 리포트**: 스캔 완료 후 인덱싱된 파일 수, 스킵된 디렉토리 수, 권한 오류 수 표시
- **선택적 sudo**: `--force-sudo` 플래그로 필요시에만 관리자 권한 사용

#### 스캔 시 스킵되는 디렉토리

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

### 2. 파일 검색

인덱싱된 파일을 검색합니다:

```bash
./bin/fman find "pattern"
```

### 3. AI 기반 파일 정리

AI를 사용하여 파일을 정리합니다:

```bash
./bin/fman organize --ai /path/to/directory
```

## 설정

첫 실행 시 `~/.fman/config.yml` 파일이 자동으로 생성됩니다:

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

## 데이터 저장

- **인덱스 데이터베이스**: `~/.fman/fman.db` (SQLite3)
- **설정 파일**: `~/.fman/config.yml`

## 개발

### 테스트 실행

```bash
go test ./...
```

### 테스트 커버리지

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

### 빌드

```bash
go build -o bin/fman
```

## 보안 고려사항

- `--force-sudo` 플래그는 신중하게 사용하세요
- 시스템 디렉토리는 기본적으로 스킵되어 안전합니다
- 권한 오류는 우아하게 처리되어 전체 스캔을 중단하지 않습니다

## 라이선스

MIT License
