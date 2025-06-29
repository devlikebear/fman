/*
Copyright © 2025 changheonshin
*/
package daemon

import (
	"context"
	"runtime"
	"sync"
	"time"
)

// ResourceLimits 데몬의 리소스 사용 제한 설정
type ResourceLimits struct {
	MaxMemoryMB   uint64        // 최대 메모리 사용량 (MB)
	MaxCPUPercent float64       // 최대 CPU 사용률 (%)
	CheckInterval time.Duration // 모니터링 주기
	ThrottleDelay time.Duration // 제한 시 지연 시간
}

// ResourceMonitor 시스템 리소스 모니터링 및 제어
type ResourceMonitor struct {
	limits        ResourceLimits
	isThrottling  bool
	lastCPUTime   time.Duration
	lastCheckTime time.Time
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// ResourceStatus 현재 리소스 사용 상태
type ResourceStatus struct {
	MemoryUsageMB float64
	CPUPercent    float64
	IsThrottling  bool
	LastChecked   time.Time
}

// NewResourceMonitor 새로운 리소스 모니터 생성
func NewResourceMonitor(limits ResourceLimits) *ResourceMonitor {
	if limits.MaxMemoryMB == 0 {
		limits.MaxMemoryMB = 500 // 기본값: 500MB
	}
	if limits.MaxCPUPercent == 0 {
		limits.MaxCPUPercent = 25.0 // 기본값: 25%
	}
	if limits.CheckInterval == 0 {
		limits.CheckInterval = 5 * time.Second // 기본값: 5초
	}
	if limits.ThrottleDelay == 0 {
		limits.ThrottleDelay = 100 * time.Millisecond // 기본값: 100ms
	}

	return &ResourceMonitor{
		limits:        limits,
		lastCheckTime: time.Now(),
	}
}

// Start 리소스 모니터링 시작
func (rm *ResourceMonitor) Start(ctx context.Context) {
	rm.mu.Lock()
	rm.ctx, rm.cancel = context.WithCancel(ctx)
	rm.mu.Unlock()

	go rm.monitorLoop()
}

// Stop 리소스 모니터링 중지
func (rm *ResourceMonitor) Stop() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.cancel != nil {
		rm.cancel()
	}
}

// GetStatus 현재 리소스 상태 반환
func (rm *ResourceMonitor) GetStatus() ResourceStatus {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryMB := float64(m.Alloc) / 1024 / 1024

	return ResourceStatus{
		MemoryUsageMB: memoryMB,
		CPUPercent:    rm.getCurrentCPUUsage(),
		IsThrottling:  rm.isThrottling,
		LastChecked:   rm.lastCheckTime,
	}
}

// ShouldThrottle 현재 쓰로틀링이 필요한지 확인
func (rm *ResourceMonitor) ShouldThrottle() bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.isThrottling
}

// WaitIfThrottling 쓰로틀링 중이면 대기
func (rm *ResourceMonitor) WaitIfThrottling(ctx context.Context) error {
	if !rm.ShouldThrottle() {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(rm.limits.ThrottleDelay):
		return nil
	}
}

// ForceGC 메모리 정리 강제 실행
func (rm *ResourceMonitor) ForceGC() {
	runtime.GC()
	runtime.GC() // 두 번 실행하여 확실히 정리
}

// monitorLoop 모니터링 메인 루프
func (rm *ResourceMonitor) monitorLoop() {
	ticker := time.NewTicker(rm.limits.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			rm.checkAndUpdateStatus()
		}
	}
}

// checkAndUpdateStatus 리소스 상태 확인 및 업데이트
func (rm *ResourceMonitor) checkAndUpdateStatus() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryMB := float64(m.Alloc) / 1024 / 1024
	cpuPercent := rm.getCurrentCPUUsage()

	// 메모리 또는 CPU 사용량이 제한을 초과하는지 확인
	memoryExceeded := memoryMB > float64(rm.limits.MaxMemoryMB)
	cpuExceeded := cpuPercent > rm.limits.MaxCPUPercent

	rm.isThrottling = memoryExceeded || cpuExceeded
	rm.lastCheckTime = time.Now()

	// 메모리 사용량이 높으면 가비지 컬렉션 실행
	if memoryExceeded {
		runtime.GC()
	}
}

// getCurrentCPUUsage 현재 CPU 사용률 계산 (간단한 추정)
func (rm *ResourceMonitor) getCurrentCPUUsage() float64 {
	// 고루틴 수를 기반으로 한 간단한 CPU 사용률 추정
	// 실제 프로덕션에서는 더 정확한 CPU 모니터링이 필요할 수 있음
	numGoroutines := float64(runtime.NumGoroutine())
	numCPU := float64(runtime.NumCPU())

	// 기본적인 추정: 고루틴 수 / CPU 코어 수 * 100
	// 실제로는 더 복잡한 계산이 필요하지만, 데몬 환경에서의 기본적인 제어용
	estimatedUsage := (numGoroutines / numCPU) * 10.0

	if estimatedUsage > 100.0 {
		estimatedUsage = 100.0
	}

	return estimatedUsage
}
