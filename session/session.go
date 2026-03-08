package session

import (
	"sync"
	"time"

	"github.com/cloudwego/eino/adk"
)

// CLI 模式会话常量
const (
	CLISessionID  = "cli"
	CLIHistoryDir = "./history/chat_history/"
)

// activeSessionID 当前活跃的会话 ID，每次启动时生成新 ID
var activeSessionID = CLISessionID

// resumeSessionID 恢复会话的 ID，由 -r 参数设置
var resumeSessionID string

var sessionIDMu sync.RWMutex

// NewDirectSessionID 生成基于时间戳的唯一会话 ID
func NewDirectSessionID() string {
	return time.Now().Format("20060102_150405")
}

// SetResumeSessionID 设置要恢复的会话 ID
func SetResumeSessionID(sessionID string) {
	sessionIDMu.Lock()
	defer sessionIDMu.Unlock()

	resumeSessionID = sessionID
}

// GetResumeSessionID 获取要恢复的会话 ID
func GetResumeSessionID() string {
	sessionIDMu.RLock()
	defer sessionIDMu.RUnlock()

	return resumeSessionID
}

// SetActiveSessionID 设置当前活跃会话 ID
func SetActiveSessionID(sessionID string) {
	sessionIDMu.Lock()
	defer sessionIDMu.Unlock()

	activeSessionID = sessionID
}

// GetActiveSessionID 获取当前活跃会话 ID
func GetActiveSessionID() string {
	sessionIDMu.RLock()
	defer sessionIDMu.RUnlock()

	return activeSessionID
}

type Session struct {
	ID   string
	Name string

	Message []adk.Message
}
