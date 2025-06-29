package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRulesCommand(t *testing.T) {
	t.Run("rules command exists", func(t *testing.T) {
		assert.NotNil(t, rulesCmd)
		assert.Equal(t, "rules", rulesCmd.Use)
		assert.Contains(t, rulesCmd.Short, "rules")
	})

	t.Run("rules functions exist", func(t *testing.T) {
		assert.NotNil(t, runRulesList)
		assert.NotNil(t, runRulesApply)
		assert.NotNil(t, runRulesRemove)
		assert.NotNil(t, runRulesEnable)
		assert.NotNil(t, runRulesDisable)
		assert.NotNil(t, runRulesInit)
	})

	t.Run("getConfigDir function", func(t *testing.T) {
		configDir := getConfigDir()
		assert.NotEmpty(t, configDir)
		assert.Contains(t, configDir, ".fman")
	})
}

func TestRunRulesCommands(t *testing.T) {
	// 테스트용 임시 HOME 설정
	setupTestHome := func(t *testing.T) (string, func()) {
		tempDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", tempDir)
		
		// fman 설정 디렉토리 생성
		fmanDir := filepath.Join(tempDir, ".fman")
		err := os.MkdirAll(fmanDir, 0755)
		assert.NoError(t, err)
		
		cleanup := func() {
			os.Setenv("HOME", originalHome)
		}
		return tempDir, cleanup
	}

	t.Run("runRulesList", func(t *testing.T) {
		tempHome, cleanup := setupTestHome(t)
		defer cleanup()
		
		err := runRulesList(rulesCmd, []string{})
		// rules.yml이 없으면 성공하지만 "No rules found" 메시지 출력
		// 성공하는 것이 정상적인 동작
		assert.NoError(t, err)
		_ = tempHome // 사용했음을 표시
	})

	t.Run("runRulesApply", func(t *testing.T) {
		tempHome, cleanup := setupTestHome(t)
		defer cleanup()
		
		err := runRulesApply(rulesCmd, []string{})
		// 활성화된 규칙이 없으면 성공하지만 "No enabled rules found" 메시지 출력
		// 성공하는 것이 정상적인 동작
		assert.NoError(t, err)
		_ = tempHome
	})

	t.Run("runRulesRemove", func(t *testing.T) {
		tempHome, cleanup := setupTestHome(t)
		defer cleanup()
		
		err := runRulesRemove(rulesCmd, []string{"test-rule"})
		// 존재하지 않는 rule이므로 에러
		assert.Error(t, err)
		_ = tempHome
	})

	t.Run("runRulesEnable", func(t *testing.T) {
		tempHome, cleanup := setupTestHome(t)
		defer cleanup()
		
		err := runRulesEnable(rulesCmd, []string{"test-rule"})
		// 존재하지 않는 rule이므로 에러
		assert.Error(t, err)
		_ = tempHome
	})

	t.Run("runRulesDisable", func(t *testing.T) {
		tempHome, cleanup := setupTestHome(t)
		defer cleanup()
		
		err := runRulesDisable(rulesCmd, []string{"test-rule"})
		// 존재하지 않는 rule이므로 에러
		assert.Error(t, err)
		_ = tempHome
	})

	t.Run("runRulesInit", func(t *testing.T) {
		tempHome, cleanup := setupTestHome(t)
		defer cleanup()
		
		err := runRulesInit(rulesCmd, []string{})
		// init 명령은 성공해야 함
		assert.NoError(t, err)
		
		// 파일이 생성되었는지 확인
		rulesFile := filepath.Join(tempHome, ".fman", "rules.yml")
		_, err = os.Stat(rulesFile)
		assert.NoError(t, err, "rules.yml should be created")
	})
}