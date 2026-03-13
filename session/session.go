package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
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
	ID   string `json:"id"`
	Name string `json:"name"`

	Messages  []adk.Message `json:"messages"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

var (
	sessionIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	// ErrSessionNotFound indicates the target session does not exist.
	ErrSessionNotFound = errors.New("session not found")
)

type JSONStore struct {
	dir string
	mu  sync.RWMutex
}

func NewJSONStore(dir string) (*JSONStore, error) {
	if strings.TrimSpace(dir) == "" {
		dir = CLIHistoryDir
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create session dir failed: %w", err)
	}

	return &JSONStore{dir: dir}, nil
}

func (s *JSONStore) Save(_ context.Context, session *Session) error {
	if session == nil {
		return errors.New("session is nil")
	}
	if err := validateSessionID(session.ID); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}
	session.UpdatedAt = now

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session failed: %w", err)
	}

	return writeAtomic(s.filePath(session.ID), data)
}

func (s *JSONStore) Get(_ context.Context, id string) (*Session, error) {
	if err := validateSessionID(id); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	raw, err := os.ReadFile(s.filePath(id))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("read session failed: %w", err)
	}

	var ss Session
	if err = json.Unmarshal(raw, &ss); err != nil {
		return nil, fmt.Errorf("unmarshal session failed: %w", err)
	}
	return &ss, nil
}

func (s *JSONStore) List(ctx context.Context) ([]*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("read session dir failed: %w", err)
	}

	sessions := make([]*Session, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), ".json")
		ss, getErr := s.getNoLock(ctx, id)
		if getErr != nil {
			continue
		}
		sessions = append(sessions, ss)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})
	return sessions, nil
}

func (s *JSONStore) Delete(_ context.Context, id string) error {
	if err := validateSessionID(id); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.Remove(s.filePath(id)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrSessionNotFound
		}
		return fmt.Errorf("delete session failed: %w", err)
	}
	return nil
}

func (s *JSONStore) AppendMessages(ctx context.Context, id string, messages ...adk.Message) (*Session, error) {
	if err := validateSessionID(id); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ss, err := s.getNoLock(ctx, id)
	if err != nil {
		if !errors.Is(err, ErrSessionNotFound) {
			return nil, err
		}
		ss = &Session{
			ID:        id,
			Name:      id,
			Messages:  make([]adk.Message, 0, len(messages)),
			CreatedAt: time.Now(),
		}
	}

	ss.Messages = append(ss.Messages, messages...)
	ss.UpdatedAt = time.Now()
	if ss.Name == "" {
		ss.Name = id
	}

	data, err := json.MarshalIndent(ss, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal session failed: %w", err)
	}
	if err = writeAtomic(s.filePath(id), data); err != nil {
		return nil, err
	}
	return ss, nil
}

func (s *JSONStore) ClearMessages(ctx context.Context, id string) (*Session, error) {
	if err := validateSessionID(id); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ss, err := s.getNoLock(ctx, id)
	if err != nil {
		return nil, err
	}

	ss.Messages = ss.Messages[:0]
	ss.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(ss, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal session failed: %w", err)
	}
	if err = writeAtomic(s.filePath(id), data); err != nil {
		return nil, err
	}
	return ss, nil
}

func (s *JSONStore) getNoLock(_ context.Context, id string) (*Session, error) {
	raw, err := os.ReadFile(s.filePath(id))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("read session failed: %w", err)
	}

	var ss Session
	if err = json.Unmarshal(raw, &ss); err != nil {
		return nil, fmt.Errorf("unmarshal session failed: %w", err)
	}
	return &ss, nil
}

func (s *JSONStore) filePath(id string) string {
	return filepath.Join(s.dir, id+".json")
}

func validateSessionID(id string) error {
	if !sessionIDPattern.MatchString(id) {
		return fmt.Errorf("invalid session id: %q", id)
	}
	return nil
}

func writeAtomic(path string, data []byte) error {
	tmpFile := path + ".tmp"

	if err := os.WriteFile(tmpFile, data, 0o644); err != nil {
		return fmt.Errorf("write temp file failed: %w", err)
	}
	if err := os.Rename(tmpFile, path); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("replace session file failed: %w", err)
	}
	return nil
}
