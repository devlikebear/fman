# fman 사용자 가이드

이 가이드는 fman을 처음 사용하는 사용자부터 고급 사용자까지 모든 기능을 효과적으로 활용할 수 있도록 도와줍니다.

## 목차

- [설치 및 초기 설정](#설치-및-초기-설정)
- [기본 사용법](#기본-사용법)
- [고급 기능](#고급-기능)
- [AI 기반 파일 정리](#ai-기반-파일-정리)
- [규칙 기반 자동화](#규칙-기반-자동화)
- [데몬 모드 활용](#데몬-모드-활용)
- [문제 해결](#문제-해결)
- [팁과 요령](#팁과-요령)

## 설치 및 초기 설정

### 1. 설치

#### 소스에서 빌드 (권장)

```bash
# 저장소 클론
git clone https://github.com/devlikebear/fman.git
cd fman

# 빌드
make build

# 실행 파일 확인
ls -la bin/fman
```

#### 직접 빌드

```bash
go build -o bin/fman
```

### 2. 첫 실행 및 설정

```bash
# 첫 실행 (설정 파일 자동 생성)
./bin/fman scan ~/Documents

# 설정 파일 위치 확인
ls -la ~/.fman/
```

설정 파일 (`~/.fman/config.yml`)이 자동으로 생성됩니다:

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

### 3. AI 설정 (선택사항)

#### Gemini 사용

1. [Google AI Studio](https://aistudio.google.com/)에서 API 키 발급
2. 설정 파일에 API 키 입력:
   ```bash
   vi ~/.fman/config.yml
   ```
   ```yaml
   ai_provider: "gemini"
   gemini:
     api_key: "your-actual-api-key-here"
     model: "gemini-1.5-flash"
   ```

#### Ollama 사용

1. [Ollama](https://ollama.ai/) 설치
2. 모델 다운로드:
   ```bash
   ollama pull llama3
   ```
3. 설정 파일 수정:
   ```yaml
   ai_provider: "ollama"
   ollama:
     base_url: "http://localhost:11434"
     model: "llama3"
   ```

## 기본 사용법

### 파일 스캔

fman의 핵심 기능인 파일 스캔부터 시작합니다.

```bash
# 기본 스캔
./bin/fman scan ~/Documents

# 상세 출력과 함께 스캔
./bin/fman scan ~/Documents --verbose

# 여러 디렉토리 동시 스캔
./bin/fman scan ~/Documents ~/Downloads ~/Pictures
```

#### 스캔 결과 예시

```
🔍 Scanning directory: /Users/user/Documents
📁 Directories processed: 45
📄 Files indexed: 1,247
⏭️  Directories skipped: 3
🚫 Permission errors: 0
⏱️  Total time: 2.3s
✅ Scan completed successfully
```

### 파일 검색

인덱싱된 파일을 빠르게 검색할 수 있습니다.

```bash
# 기본 검색 (파일명)
./bin/fman find "*.pdf"

# 특정 확장자 검색
./bin/fman find "*.jpg"

# 부분 이름 검색
./bin/fman find "report"
```

#### 고급 검색 옵션

```bash
# 크기 기반 검색 (10MB 이상)
./bin/fman find --size "+10MB"

# 날짜 기반 검색 (최근 7일)
./bin/fman find --modified "-7d"

# 특정 경로에서만 검색
./bin/fman find "document" --path "/Users/user/Documents"

# 복합 조건 검색
./bin/fman find "*.jpg" --size "+1MB" --modified "-30d"
```

### 중복 파일 찾기

중복 파일을 효율적으로 관리할 수 있습니다.

```bash
# 모든 중복 파일 찾기
./bin/fman duplicate

# 특정 디렉토리에서만 검색
./bin/fman duplicate ~/Pictures

# 최소 크기 지정 (1MB 이상)
./bin/fman duplicate --min-size 1048576

# 대화형 모드로 선택적 삭제
./bin/fman duplicate --interactive
```

#### 대화형 모드 예시

```
🔍 Found 3 groups of duplicate files:

Group 1: IMG_1234.jpg (2 files, 2.5MB each)
  1. /Users/user/Pictures/IMG_1234.jpg
  2. /Users/user/Downloads/IMG_1234.jpg

Which file would you like to keep? (1-2, s=skip, q=quit): 1
✅ Deleted: /Users/user/Downloads/IMG_1234.jpg
```

## 고급 기능

### 백그라운드 데몬 사용

대용량 디렉토리 스캔이나 장시간 작업을 위해 데몬 모드를 활용할 수 있습니다.

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

#### 데몬 상태 정보

```
✅ Daemon is running
📊 Status: Active
🔧 PID: 12345
📍 Socket: /Users/user/.fman/daemon.sock
⏰ Started: 2024-01-15 10:30:25
📈 Uptime: 2h 15m 30s
🔄 Jobs processed: 15
📋 Queue size: 0
```

### 작업 큐 관리

백그라운드 작업을 효율적으로 관리할 수 있습니다.

```bash
# 모든 작업 목록 보기
./bin/fman queue list

# 특정 작업 상태 확인
./bin/fman queue status abc123

# 작업 취소
./bin/fman queue cancel abc123

# 완료된 작업 정리
./bin/fman queue clear
```

#### 작업 목록 예시

```
📋 Job Queue Status:

ID       Status     Path                    Progress  Created
abc123   Running    /Users/user/Documents   45%       2024-01-15 10:30
def456   Pending    /Users/user/Pictures    0%        2024-01-15 10:35
ghi789   Completed  /Users/user/Downloads   100%      2024-01-15 10:25

Total: 3 jobs (1 running, 1 pending, 1 completed)
```

## AI 기반 파일 정리

AI를 활용하여 파일을 지능적으로 정리할 수 있습니다.

### 기본 사용법

```bash
# AI 제안 받기 (미리보기)
./bin/fman organize --ai ~/Downloads --dry-run

# AI 제안 적용
./bin/fman organize --ai ~/Downloads

# 특정 파일 타입만 정리
./bin/fman organize --ai ~/Downloads --type "image"
```

### AI 제안 예시

```
🤖 AI Analysis Complete

📁 Suggested Organization for /Users/user/Downloads:

📸 Images (15 files)
  → Create folder: Images/2024/January/
  → Move: IMG_1234.jpg, IMG_1235.jpg, ...

📄 Documents (8 files)
  → Create folder: Documents/PDFs/
  → Move: report.pdf, manual.pdf, ...

🎵 Audio (3 files)
  → Create folder: Audio/Music/
  → Move: song1.mp3, song2.mp3, ...

💾 Archives (2 files)
  → Create folder: Archives/
  → Move: backup.zip, data.tar.gz

Would you like to apply these changes? (y/n): y
✅ Organization completed successfully!
```

### AI 제안 커스터마이징

AI 제안을 더 정확하게 만들기 위한 팁:

1. **파일명 패턴 활용**: 의미 있는 파일명 사용
2. **디렉토리 구조**: 기존 정리 패턴 유지
3. **파일 타입 지정**: 특정 타입만 정리하여 정확도 향상

```bash
# 이미지만 정리
./bin/fman organize --ai ~/Downloads --type "image"

# 문서만 정리
./bin/fman organize --ai ~/Downloads --type "document"

# 특정 날짜 이후 파일만
./bin/fman organize --ai ~/Downloads --modified "-30d"
```

## 규칙 기반 자동화

반복적인 파일 정리 작업을 자동화할 수 있습니다.

### 규칙 초기화

```bash
# 예제 규칙으로 초기화
./bin/fman rules init

# 규칙 목록 확인
./bin/fman rules list
```

### 예제 규칙들

초기화하면 다음과 같은 규칙들이 생성됩니다:

```yaml
# 스크린샷 정리 규칙
- name: screenshot-cleanup
  description: Move old screenshots to archive folder
  enabled: true
  conditions:
    - field: name
      operator: matches
      value: "^(Screenshot|Screen Shot).*\\.(png|jpg)$"
    - field: age_days
      operator: greater_than
      value: 30
  actions:
    - type: move
      params:
        destination: "~/Pictures/Screenshots/Archive/"

# 임시 파일 정리 규칙
- name: temp-file-cleanup
  description: Delete temporary files older than 7 days
  enabled: true
  conditions:
    - field: name
      operator: matches
      value: "\\.(tmp|temp|cache)$"
    - field: age_days
      operator: greater_than
      value: 7
  actions:
    - type: delete
```

### 규칙 관리

```bash
# 규칙 적용 (미리보기)
./bin/fman rules apply --dry-run ~/Downloads

# 규칙 적용
./bin/fman rules apply ~/Downloads

# 특정 규칙만 적용
./bin/fman rules apply ~/Downloads --rule "screenshot-cleanup"

# 규칙 활성화/비활성화
./bin/fman rules enable screenshot-cleanup
./bin/fman rules disable temp-file-cleanup

# 규칙 제거
./bin/fman rules remove screenshot-cleanup
```

### 사용자 정의 규칙 작성

규칙 파일 (`~/.fman/rules.yml`)을 직접 편집하여 사용자 정의 규칙을 만들 수 있습니다:

```yaml
- name: organize-downloads
  description: Organize downloads by file type
  enabled: true
  priority: 1
  conditions:
    - field: path
      operator: contains
      value: "/Downloads/"
    - field: extension
      operator: in
      value: ["pdf", "doc", "docx", "txt"]
  actions:
    - type: move
      params:
        destination: "~/Documents/Downloads/{{ .Extension }}/"
        create_dirs: true

- name: large-file-alert
  description: Alert for large files
  enabled: true
  conditions:
    - field: size
      operator: greater_than
      value: 1073741824  # 1GB
  actions:
    - type: log
      params:
        message: "Large file detected: {{ .Path }} ({{ .SizeHuman }})"
```

## 데몬 모드 활용

### 자동 시작 설정

#### macOS (launchd)

`~/Library/LaunchAgents/com.fman.daemon.plist` 파일 생성:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.fman.daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>/path/to/fman</string>
        <string>daemon</string>
        <string>start</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
```

```bash
# 서비스 등록
launchctl load ~/Library/LaunchAgents/com.fman.daemon.plist

# 서비스 시작
launchctl start com.fman.daemon
```

#### Linux (systemd)

`~/.config/systemd/user/fman-daemon.service` 파일 생성:

```ini
[Unit]
Description=fman daemon
After=network.target

[Service]
Type=forking
ExecStart=/path/to/fman daemon start
ExecStop=/path/to/fman daemon stop
Restart=always
RestartSec=10

[Install]
WantedBy=default.target
```

```bash
# 서비스 등록 및 시작
systemctl --user daemon-reload
systemctl --user enable fman-daemon
systemctl --user start fman-daemon
```

### 데몬 모니터링

```bash
# 데몬 로그 확인 (macOS)
tail -f ~/.fman/daemon.log

# 데몬 성능 모니터링
./bin/fman daemon status --verbose

# 데몬 재시작 (문제 발생 시)
./bin/fman daemon restart
```

## 문제 해결

### 일반적인 문제들

#### 1. 권한 오류

```bash
# 문제: Permission denied 오류
# 해결: 권한 확인 후 필요시 sudo 사용
./bin/fman scan /path/to/directory --force-sudo
```

#### 2. 데몬 시작 실패

```bash
# 문제: 데몬이 시작되지 않음
# 해결: 기존 프로세스 확인 후 정리
ps aux | grep fman
kill -9 <pid>
rm ~/.fman/daemon.pid
./bin/fman daemon start
```

#### 3. AI 요청 실패

```bash
# 문제: AI 제안 실패
# 해결: API 키 및 네트워크 확인
./bin/fman organize --ai ~/Downloads --verbose
```

#### 4. 데이터베이스 오류

```bash
# 문제: 데이터베이스 손상
# 해결: 데이터베이스 재생성
mv ~/.fman/fman.db ~/.fman/fman.db.backup
./bin/fman scan ~/Documents  # 데이터베이스 재생성
```

### 로그 확인

```bash
# 일반 로그
tail -f ~/.fman/fman.log

# 데몬 로그
tail -f ~/.fman/daemon.log

# 상세 로그 활성화
export FMAN_LOG_LEVEL=debug
./bin/fman scan ~/Documents --verbose
```

### 성능 최적화

#### 대용량 디렉토리 스캔

```bash
# 메모리 사용량 제한
export FMAN_MAX_MEMORY=1GB
./bin/fman scan ~/Documents

# 병렬 처리 수 조정
export FMAN_WORKERS=4
./bin/fman scan ~/Documents

# 배치 크기 조정
export FMAN_BATCH_SIZE=1000
./bin/fman scan ~/Documents
```

## 팁과 요령

### 효율적인 워크플로우

#### 1. 정기적인 스캔 스케줄링

```bash
# crontab 설정 (매일 새벽 2시)
0 2 * * * /path/to/fman scan ~/Documents ~/Downloads ~/Pictures
```

#### 2. 규칙 기반 자동 정리

```bash
# 매주 규칙 적용
0 0 * * 0 /path/to/fman rules apply ~/Downloads
```

#### 3. 중복 파일 정기 정리

```bash
# 매월 중복 파일 체크
0 0 1 * * /path/to/fman duplicate --min-size 1048576 > ~/duplicate_report.txt
```

### 고급 사용 패턴

#### 1. 파이프라인 활용

```bash
# 검색 결과를 다른 도구로 전달
./bin/fman find "*.log" | xargs du -sh

# 중복 파일 목록을 파일로 저장
./bin/fman duplicate --min-size 10485760 > large_duplicates.txt
```

#### 2. 스크립트 통합

```bash
#!/bin/bash
# 일일 파일 정리 스크립트

echo "Starting daily file management..."

# 1. 새 파일 스캔
./bin/fman scan ~/Downloads ~/Documents

# 2. 규칙 적용
./bin/fman rules apply ~/Downloads --dry-run

# 3. AI 정리 (선택적)
if [ "$1" == "--ai" ]; then
    ./bin/fman organize --ai ~/Downloads --dry-run
fi

# 4. 중복 파일 체크
./bin/fman duplicate --min-size 1048576

echo "Daily file management completed!"
```

#### 3. 모니터링 및 알림

```bash
# 대용량 파일 알림
./bin/fman find --size "+100MB" | \
while read file; do
    echo "Large file found: $file"
    # 알림 발송 (예: Slack, 이메일 등)
done
```

### 성능 최적화 팁

1. **선택적 스캔**: 필요한 디렉토리만 스캔
2. **배치 처리**: 대량 작업은 데몬 모드 활용
3. **정기적인 정리**: 데이터베이스 크기 관리
4. **규칙 우선순위**: 자주 사용하는 규칙의 우선순위 높이기

### 보안 고려사항

1. **API 키 보호**: 설정 파일 권한 제한 (600)
2. **경로 검증**: 절대 경로 사용 권장
3. **권한 최소화**: sudo 사용 최소화
4. **로그 관리**: 민감한 정보 로그 제외

이 사용자 가이드를 통해 fman의 모든 기능을 효과적으로 활용하시기 바랍니다! 