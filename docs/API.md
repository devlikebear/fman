# fman API 문서

이 문서는 fman의 내부 API와 인터페이스에 대한 상세한 설명을 제공합니다.

## 목차

- [AI Provider 인터페이스](#ai-provider-인터페이스)
- [데이터베이스 인터페이스](#데이터베이스-인터페이스)
- [데몬 인터페이스](#데몬-인터페이스)
- [큐 인터페이스](#큐-인터페이스)
- [규칙 엔진](#규칙-엔진)

## AI Provider 인터페이스

### AIProvider

AI 공급자를 위한 기본 인터페이스입니다.

```go
type AIProvider interface {
    SuggestOrganization(ctx context.Context, filePaths []string) (string, error)
}
```

#### 메서드

- **SuggestOrganization**: 파일 목록을 받아 AI 기반 정리 제안을 반환합니다.
  - `ctx`: 컨텍스트 (타임아웃, 취소 등)
  - `filePaths`: 정리할 파일 경로 목록
  - 반환값: 정리 제안 텍스트와 에러

### 구현체

#### GeminiProvider

Google Gemini API를 사용하는 구현체입니다.

```go
type GeminiProvider struct {
    client *genai.Client
    model  string
}

func NewGeminiProvider(apiKey, model string) (*GeminiProvider, error)
```

#### OllamaProvider

Ollama 로컬 AI 서비스를 사용하는 구현체입니다.

```go
type OllamaProvider struct {
    baseURL string
    model   string
    client  *http.Client
}

func NewOllamaProvider(baseURL, model string) *OllamaProvider
```

## 데이터베이스 인터페이스

### DatabaseInterface

파일 메타데이터 저장을 위한 데이터베이스 인터페이스입니다.

```go
type DatabaseInterface interface {
    Initialize() error
    InsertFile(file *FileInfo) error
    UpdateFile(file *FileInfo) error
    DeleteFile(path string) error
    GetFile(path string) (*FileInfo, error)
    SearchFiles(criteria SearchCriteria) ([]*FileInfo, error)
    GetFilesByHash(hash string) ([]*FileInfo, error)
    GetStats() (*DatabaseStats, error)
    Close() error
}
```

### FileInfo 구조체

```go
type FileInfo struct {
    ID         int       `db:"id"`
    Path       string    `db:"path"`
    Name       string    `db:"name"`
    Size       int64     `db:"size"`
    ModifiedAt time.Time `db:"modified_at"`
    IndexedAt  time.Time `db:"indexed_at"`
    FileHash   string    `db:"file_hash"`
}
```

### SearchCriteria 구조체

```go
type SearchCriteria struct {
    NamePattern string
    SizeMin     int64
    SizeMax     int64
    ModifiedAfter  time.Time
    ModifiedBefore time.Time
    PathPattern    string
    Limit          int
    Offset         int
}
```

## 데몬 인터페이스

### DaemonInterface

백그라운드 데몬 서비스를 위한 인터페이스입니다.

```go
type DaemonInterface interface {
    Start(ctx context.Context) error
    Stop() error
    Status() (*DaemonStatus, error)
    IsRunning() bool
    EnqueueScan(request *ScanRequest) (*Job, error)
    GetJob(jobID string) (*Job, error)
    CancelJob(jobID string) error
    ListJobs(status JobStatus) ([]*Job, error)
    ClearQueue() error
}
```

### ClientInterface

데몬 클라이언트를 위한 인터페이스입니다.

```go
type ClientInterface interface {
    Connect() error
    Disconnect() error
    IsConnected() bool
    SendRequest(req *Request) (*Response, error)
    IsDaemonRunning() bool
    StartDaemon() error
    StopDaemon() error
    GetStatus() (*DaemonStatus, error)
    EnqueueScan(request *ScanRequest) (*Job, error)
    GetJob(jobID string) (*Job, error)
    CancelJob(jobID string) error
    ListJobs(status JobStatus) ([]*Job, error)
    ClearQueue() error
}
```

### 메시지 프로토콜

#### Request 구조체

```go
type Request struct {
    Type RequestType    `json:"type"`
    Data interface{}    `json:"data,omitempty"`
}

type RequestType string

const (
    RequestTypeScan       RequestType = "scan"
    RequestTypeStatus     RequestType = "status"
    RequestTypeJobStatus  RequestType = "job_status"
    RequestTypeJobList    RequestType = "job_list"
    RequestTypeJobCancel  RequestType = "job_cancel"
    RequestTypeQueueClear RequestType = "queue_clear"
    RequestTypeShutdown   RequestType = "shutdown"
)
```

#### Response 구조체

```go
type Response struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}
```

## 큐 인터페이스

### QueueInterface

작업 큐 관리를 위한 인터페이스입니다.

```go
type QueueInterface interface {
    Add(job *Job) error
    Next(ctx context.Context) (*Job, error)
    Get(jobID string) (*Job, error)
    Update(job *Job) error
    List(status JobStatus) ([]*Job, error)
    Cancel(jobID string) error
    Clear() error
    Size() int
    Stats() map[string]int
}
```

### Job 구조체

```go
type Job struct {
    ID          string           `json:"id"`
    Path        string           `json:"path"`
    Status      JobStatus        `json:"status"`
    Options     *ScanOptions     `json:"options,omitempty"`
    Stats       *ScanStats       `json:"stats,omitempty"`
    Error       string           `json:"error,omitempty"`
    Progress    float64          `json:"progress"`
    CreatedAt   time.Time        `json:"created_at"`
    StartedAt   *time.Time       `json:"started_at,omitempty"`
    CompletedAt *time.Time       `json:"completed_at,omitempty"`
}

type JobStatus string

const (
    JobStatusPending   JobStatus = "pending"
    JobStatusRunning   JobStatus = "running"
    JobStatusCompleted JobStatus = "completed"
    JobStatusFailed    JobStatus = "failed"
    JobStatusCancelled JobStatus = "cancelled"
)
```

## 규칙 엔진

### RuleManager

파일 정리 규칙을 관리하는 인터페이스입니다.

```go
type RuleManager interface {
    LoadRules() error
    SaveRules() error
    AddRule(rule *Rule) error
    RemoveRule(name string) error
    GetRule(name string) (*Rule, error)
    ListRules() []*Rule
    EnableRule(name string) error
    DisableRule(name string) error
    InitializeWithExamples() error
}
```

### Rule 구조체

```go
type Rule struct {
    Name        string      `yaml:"name" json:"name"`
    Description string      `yaml:"description" json:"description"`
    Enabled     bool        `yaml:"enabled" json:"enabled"`
    Conditions  []Condition `yaml:"conditions" json:"conditions"`
    Actions     []Action    `yaml:"actions" json:"actions"`
    Priority    int         `yaml:"priority" json:"priority"`
}
```

### Condition 구조체

```go
type Condition struct {
    Field    string      `yaml:"field" json:"field"`
    Operator string      `yaml:"operator" json:"operator"`
    Value    interface{} `yaml:"value" json:"value"`
}
```

### Action 구조체

```go
type Action struct {
    Type   string                 `yaml:"type" json:"type"`
    Params map[string]interface{} `yaml:"params" json:"params"`
}
```

### RuleEvaluator

규칙 평가를 담당하는 인터페이스입니다.

```go
type RuleEvaluator interface {
    EvaluateFile(file *FileInfo, rules []*Rule) ([]*Rule, error)
    EvaluateCondition(file *FileInfo, condition *Condition) (bool, error)
}
```

### RuleExecutor

규칙 실행을 담당하는 인터페이스입니다.

```go
type RuleExecutor interface {
    ExecuteActions(file *FileInfo, actions []Action, dryRun bool) (*ExecutionResult, error)
    ExecuteAction(file *FileInfo, action *Action, dryRun bool) (*ActionResult, error)
}
```

## 에러 타입

### 공통 에러

```go
var (
    ErrFileNotFound      = errors.New("file not found")
    ErrInvalidPath       = errors.New("invalid path")
    ErrPermissionDenied  = errors.New("permission denied")
    ErrDatabaseError     = errors.New("database error")
)
```

### 데몬 관련 에러

```go
var (
    ErrDaemonNotRunning     = errors.New("daemon is not running")
    ErrDaemonAlreadyRunning = errors.New("daemon is already running")
    ErrConnectionFailed     = errors.New("failed to connect to daemon")
    ErrJobNotFound          = errors.New("job not found")
)
```

### AI 관련 에러

```go
var (
    ErrAIProviderNotConfigured = errors.New("AI provider not configured")
    ErrAIRequestFailed         = errors.New("AI request failed")
    ErrInvalidAPIKey           = errors.New("invalid API key")
)
```

## 사용 예제

### AI Provider 사용

```go
// Gemini 공급자 생성
provider, err := ai.NewGeminiProvider("your-api-key", "gemini-1.5-flash")
if err != nil {
    log.Fatal(err)
}

// 파일 정리 제안 요청
ctx := context.Background()
files := []string{"/path/to/file1.jpg", "/path/to/file2.pdf"}
suggestion, err := provider.SuggestOrganization(ctx, files)
if err != nil {
    log.Fatal(err)
}

fmt.Println("AI 제안:", suggestion)
```

### 데몬 클라이언트 사용

```go
// 클라이언트 생성
client := daemon.NewDaemonClient(nil)
defer client.Disconnect()

// 데몬 시작
if err := client.StartDaemon(); err != nil {
    log.Fatal(err)
}

// 스캔 작업 큐에 추가
request := &daemon.ScanRequest{
    Path: "/home/user/Documents",
    Options: &daemon.ScanOptions{
        Verbose: true,
    },
}

job, err := client.EnqueueScan(request)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("작업 ID: %s\n", job.ID)
```

### 규칙 엔진 사용

```go
// 규칙 매니저 생성
manager := rules.NewRuleManager("~/.fman/rules.yml")

// 규칙 로드
if err := manager.LoadRules(); err != nil {
    log.Fatal(err)
}

// 파일에 적용할 규칙 찾기
evaluator := rules.NewRuleEvaluator()
applicableRules, err := evaluator.EvaluateFile(fileInfo, manager.ListRules())
if err != nil {
    log.Fatal(err)
}

// 규칙 실행
executor := rules.NewRuleExecutor()
for _, rule := range applicableRules {
    result, err := executor.ExecuteActions(fileInfo, rule.Actions, false)
    if err != nil {
        log.Printf("규칙 실행 실패: %v", err)
        continue
    }
    fmt.Printf("규칙 '%s' 실행 완료: %s\n", rule.Name, result.Summary)
}
``` 