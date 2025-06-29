# fman 개발자 가이드

이 문서는 fman 프로젝트에 기여하거나 개발하려는 개발자들을 위한 가이드입니다.

## 목차

- [개발 환경 설정](#개발-환경-설정)
- [프로젝트 구조](#프로젝트-구조)
- [개발 워크플로우](#개발-워크플로우)
- [테스트 가이드](#테스트-가이드)
- [코딩 스타일](#코딩-스타일)
- [기여 가이드](#기여-가이드)

## 개발 환경 설정

### 필수 요구사항

- **Go**: 1.21 이상
- **Git**: 버전 관리
- **Make**: 빌드 자동화
- **golangci-lint**: 코드 린팅 (선택사항)

### 개발 도구 설치

```bash
# Go 설치 확인
go version

# golangci-lint 설치
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 프로젝트 클론
git clone https://github.com/devlikebear/fman.git
cd fman

# 의존성 설치
go mod download

# 빌드 테스트
make build
```

### IDE 설정

#### VS Code

권장 확장 프로그램:
- Go (Google)
- Go Test Explorer
- GitLens

설정 (`.vscode/settings.json`):
```json
{
    "go.useLanguageServer": true,
    "go.lintTool": "golangci-lint",
    "go.lintOnSave": "package",
    "go.formatTool": "goimports",
    "go.testFlags": ["-v", "-race"],
    "go.coverOnSave": true,
    "go.coverageDecorator": {
        "type": "gutter"
    }
}
```

## 프로젝트 구조

```
fman/
├── cmd/                    # CLI 명령어 구현
│   ├── daemon.go          # 데몬 관리 명령어
│   ├── duplicate.go       # 중복 파일 관리
│   ├── find.go            # 파일 검색
│   ├── organize.go        # AI 기반 정리
│   ├── queue.go           # 큐 관리
│   ├── root.go            # 루트 명령어
│   ├── rules.go           # 규칙 관리
│   └── scan.go            # 파일 스캔
├── internal/              # 내부 패키지
│   ├── ai/               # AI 공급자
│   │   ├── provider.go   # 인터페이스 정의
│   │   ├── gemini.go     # Gemini 구현
│   │   └── ollama.go     # Ollama 구현
│   ├── daemon/           # 데몬 서비스
│   │   ├── server.go     # 데몬 서버
│   │   ├── client.go     # 데몬 클라이언트
│   │   ├── queue.go      # 작업 큐
│   │   ├── worker.go     # 워커
│   │   └── types.go      # 타입 정의
│   ├── db/               # 데이터베이스
│   │   └── database.go   # SQLite 구현
│   ├── rules/            # 규칙 엔진
│   │   ├── manager.go    # 규칙 관리
│   │   ├── evaluator.go  # 규칙 평가
│   │   ├── executor.go   # 규칙 실행
│   │   └── types.go      # 타입 정의
│   ├── scanner/          # 파일 스캐너
│   │   └── scanner.go    # 스캔 로직
│   └── utils/            # 유틸리티
│       ├── paths.go      # 경로 처리
│       └── permissions.go # 권한 처리
├── docs/                 # 문서
├── main.go               # 진입점
├── Makefile             # 빌드 스크립트
├── Dockerfile           # Docker 이미지
├── go.mod               # Go 모듈
└── go.sum               # 의존성 체크섬
```

### 패키지 설계 원칙

1. **관심사 분리**: 각 패키지는 명확한 책임을 가집니다
2. **인터페이스 우선**: 구체적 구현보다 인터페이스를 우선합니다
3. **테스트 가능성**: 모든 패키지는 단위 테스트가 가능해야 합니다
4. **의존성 최소화**: 외부 의존성을 최소화합니다

## 개발 워크플로우

### 1. 이슈 생성 및 브랜치 생성

```bash
# 기능 브랜치 생성
git checkout -b feature/새로운-기능

# 버그 수정 브랜치 생성
git checkout -b fix/버그-설명

# 문서 업데이트 브랜치 생성
git checkout -b docs/문서-업데이트
```

### 2. 개발 사이클

```bash
# 1. 코드 작성
# 2. 테스트 작성
# 3. 테스트 실행
make test

# 4. 린팅 실행
golangci-lint run ./...

# 5. 빌드 확인
make build

# 6. 커버리지 확인
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

### 3. 커밋 및 푸시

```bash
# 변경사항 스테이징
git add .

# Conventional Commits 규칙에 따른 커밋
git commit -m "feat(scanner): add support for symbolic links"

# 브랜치 푸시
git push origin feature/새로운-기능
```

### 4. Pull Request 생성

PR 템플릿:
```markdown
## 변경사항 요약
- 새로운 기능/버그 수정에 대한 간단한 설명

## 변경 타입
- [ ] 새로운 기능
- [ ] 버그 수정
- [ ] 문서 업데이트
- [ ] 리팩토링
- [ ] 테스트 개선

## 테스트
- [ ] 단위 테스트 추가/수정
- [ ] 통합 테스트 확인
- [ ] 수동 테스트 완료

## 체크리스트
- [ ] 코드가 프로젝트 스타일 가이드를 따름
- [ ] 테스트 커버리지 70% 이상 유지
- [ ] 문서 업데이트 (필요한 경우)
- [ ] Breaking changes 없음 (또는 명시)
```

## 테스트 가이드

### 테스트 구조

```go
func TestFunctionName(t *testing.T) {
    // Arrange (준비)
    // Given
    
    // Act (실행)
    // When
    
    // Assert (검증)
    // Then
}
```

### 단위 테스트 예제

```go
func TestDatabaseInsertFile(t *testing.T) {
    // Arrange
    db := setupTestDB(t)
    defer db.Close()
    
    file := &FileInfo{
        Path:       "/test/file.txt",
        Name:       "file.txt",
        Size:       1024,
        ModifiedAt: time.Now(),
        FileHash:   "abc123",
    }
    
    // Act
    err := db.InsertFile(file)
    
    // Assert
    assert.NoError(t, err)
    assert.NotZero(t, file.ID)
    
    // Verify the file was actually inserted
    retrieved, err := db.GetFile(file.Path)
    assert.NoError(t, err)
    assert.Equal(t, file.Path, retrieved.Path)
}
```

### 모킹 사용

```go
// Mock 생성 (testify/mock 사용)
type MockAIProvider struct {
    mock.Mock
}

func (m *MockAIProvider) SuggestOrganization(ctx context.Context, filePaths []string) (string, error) {
    args := m.Called(ctx, filePaths)
    return args.String(0), args.Error(1)
}

// 테스트에서 사용
func TestOrganizeWithAI(t *testing.T) {
    mockAI := new(MockAIProvider)
    mockAI.On("SuggestOrganization", mock.Anything, mock.Anything).
        Return("mv file1.jpg photos/", nil)
    
    organizer := NewOrganizer(mockAI)
    result, err := organizer.OrganizeFiles([]string{"file1.jpg"})
    
    assert.NoError(t, err)
    assert.Contains(t, result, "photos/")
    mockAI.AssertExpectations(t)
}
```

### 통합 테스트

```go
func TestFullScanWorkflow(t *testing.T) {
    // 임시 디렉토리 생성
    tempDir := t.TempDir()
    
    // 테스트 파일 생성
    testFile := filepath.Join(tempDir, "test.txt")
    err := os.WriteFile(testFile, []byte("test content"), 0644)
    require.NoError(t, err)
    
    // 데이터베이스 설정
    db := setupTestDB(t)
    defer db.Close()
    
    // 스캐너 실행
    scanner := NewScanner(db)
    stats, err := scanner.ScanDirectory(tempDir, &ScanOptions{})
    
    // 검증
    assert.NoError(t, err)
    assert.Equal(t, 1, stats.FilesIndexed)
    
    // 데이터베이스에서 파일 확인
    files, err := db.SearchFiles(SearchCriteria{PathPattern: tempDir})
    assert.NoError(t, err)
    assert.Len(t, files, 1)
}
```

### 테스트 실행

```bash
# 모든 테스트 실행
make test

# 특정 패키지 테스트
go test ./internal/db/

# 특정 테스트 함수 실행
go test -run TestDatabaseInsertFile ./internal/db/

# 벤치마크 테스트
go test -bench=. ./internal/scanner/

# 커버리지와 함께 실행
go test -cover ./...

# 상세한 커버리지 리포트
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 코딩 스타일

### Go 코딩 컨벤션

1. **gofmt**: 코드 포맷팅 자동화
2. **goimports**: import 문 자동 정리
3. **golint**: 린팅 규칙 준수
4. **go vet**: 정적 분석 도구 사용

### 네이밍 컨벤션

```go
// 패키지명: 소문자, 단수형
package scanner

// 상수: 대문자로 시작, CamelCase
const DefaultTimeout = 30 * time.Second

// 변수: camelCase
var configPath string

// 함수: PascalCase (공개), camelCase (비공개)
func NewScanner() *Scanner { }
func validatePath() error { }

// 타입: PascalCase
type FileInfo struct {
    ID   int    `json:"id"`
    Path string `json:"path"`
}

// 인터페이스: -er 접미사 선호
type Scanner interface {
    Scan() error
}
```

### 에러 처리

```go
// 에러 래핑
if err != nil {
    return fmt.Errorf("failed to scan directory %s: %w", path, err)
}

// 사용자 정의 에러
var ErrInvalidPath = errors.New("invalid path")

// 에러 타입 정의
type ScanError struct {
    Path string
    Err  error
}

func (e *ScanError) Error() string {
    return fmt.Sprintf("scan error at %s: %v", e.Path, e.Err)
}
```

### 주석 작성

```go
// Package scanner provides file system scanning functionality.
// It supports recursive directory traversal with permission handling.
package scanner

// Scanner handles file system scanning operations.
// It maintains a database connection for storing file metadata.
type Scanner struct {
    db Database
}

// NewScanner creates a new Scanner instance.
// The database parameter must be initialized and connected.
func NewScanner(db Database) *Scanner {
    return &Scanner{db: db}
}

// ScanDirectory recursively scans a directory and indexes all files.
// It returns statistics about the scan operation.
//
// The path parameter must be an absolute path to an existing directory.
// Options can be nil to use default settings.
//
// Returns an error if the path is invalid or if database operations fail.
func (s *Scanner) ScanDirectory(path string, options *ScanOptions) (*ScanStats, error) {
    // Implementation...
}
```

### 구조체 태그

```go
type FileInfo struct {
    ID         int       `json:"id" db:"id"`
    Path       string    `json:"path" db:"path" validate:"required"`
    Name       string    `json:"name" db:"name"`
    Size       int64     `json:"size" db:"size" validate:"min=0"`
    ModifiedAt time.Time `json:"modified_at" db:"modified_at"`
}
```

## 기여 가이드

### 이슈 리포팅

버그 리포트 템플릿:
```markdown
## 버그 설명
버그에 대한 명확하고 간결한 설명

## 재현 단계
1. '...' 이동
2. '...' 클릭
3. '...' 스크롤
4. 에러 확인

## 예상 동작
예상했던 동작에 대한 설명

## 실제 동작
실제로 일어난 동작에 대한 설명

## 환경
- OS: [예: macOS 14.0]
- Go 버전: [예: 1.21.0]
- fman 버전: [예: v1.0.0]

## 추가 정보
스크린샷, 로그, 기타 관련 정보
```

### 기능 요청

기능 요청 템플릿:
```markdown
## 기능 설명
제안하는 기능에 대한 명확하고 간결한 설명

## 문제점
이 기능이 해결하는 문제점 설명

## 제안 솔루션
원하는 동작에 대한 설명

## 대안
고려한 다른 대안들

## 추가 정보
기타 관련 정보나 스크린샷
```

### 코드 리뷰 가이드

리뷰어를 위한 체크리스트:
- [ ] 코드가 요구사항을 충족하는가?
- [ ] 테스트가 충분한가?
- [ ] 에러 처리가 적절한가?
- [ ] 성능 이슈가 없는가?
- [ ] 보안 취약점이 없는가?
- [ ] 문서가 업데이트되었는가?
- [ ] Breaking changes가 명시되었는가?

### 릴리스 프로세스

1. **버전 태깅**
   ```bash
   git tag -a v1.2.0 -m "Release version 1.2.0"
   git push origin v1.2.0
   ```

2. **체인지로그 업데이트**
   - 새로운 기능
   - 버그 수정
   - Breaking changes
   - 성능 개선

3. **릴리스 노트 작성**
   - 주요 변경사항 요약
   - 마이그레이션 가이드 (필요시)
   - 알려진 이슈

## 성능 최적화

### 프로파일링

```bash
# CPU 프로파일링
go test -cpuprofile=cpu.prof -bench=.

# 메모리 프로파일링
go test -memprofile=mem.prof -bench=.

# 프로파일 분석
go tool pprof cpu.prof
go tool pprof mem.prof
```

### 벤치마크 작성

```go
func BenchmarkScanDirectory(b *testing.B) {
    tempDir := setupBenchmarkDir(b)
    defer os.RemoveAll(tempDir)
    
    scanner := NewScanner(setupTestDB(b))
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := scanner.ScanDirectory(tempDir, nil)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### 메모리 최적화

```go
// 슬라이스 사전 할당
files := make([]*FileInfo, 0, expectedCount)

// 문자열 빌더 사용
var builder strings.Builder
builder.Grow(expectedSize)

// 풀 사용으로 객체 재활용
var fileInfoPool = sync.Pool{
    New: func() interface{} {
        return &FileInfo{}
    },
}
```

## 보안 고려사항

### 입력 검증

```go
func validatePath(path string) error {
    // 절대 경로 확인
    if !filepath.IsAbs(path) {
        return fmt.Errorf("path must be absolute: %s", path)
    }
    
    // 경로 순회 공격 방지
    cleaned := filepath.Clean(path)
    if cleaned != path {
        return fmt.Errorf("invalid path: %s", path)
    }
    
    // 존재 확인
    if _, err := os.Stat(path); err != nil {
        return fmt.Errorf("path does not exist: %s", path)
    }
    
    return nil
}
```

### 권한 처리

```go
func checkPermissions(path string) error {
    info, err := os.Stat(path)
    if err != nil {
        return err
    }
    
    // 읽기 권한 확인
    if info.Mode()&0400 == 0 {
        return fmt.Errorf("no read permission for %s", path)
    }
    
    return nil
}
```

이 개발자 가이드를 따라 fman 프로젝트에 기여해 주세요! 