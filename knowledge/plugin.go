package knowledge

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/liweiming-nova/open_code/config"
	"github.com/redis/go-redis/v9"
)

const (
	defaultChunkSize  = 1000
	defaultOverlap    = 100
	defaultBatchSize  = 32
	defaultStateFile  = "indexed_files.json"
	defaultKeyPrefix  = "oc_kb"
	defaultMetricName = "COSINE"
)

type Plugin struct {
	cfg      config.KnowledgePlugin
	stateMu  sync.Mutex
	state    persistentState
	stateDir string
	rdb      *redis.Client
	httpc    *http.Client
}

type FileRecord struct {
	FilePath     string    `json:"file_path"`
	Checksum     string    `json:"checksum"`
	ChunkCount   int       `json:"chunk_count"`
	VectorDim    int       `json:"vector_dim"`
	RedisKeys    []string  `json:"redis_keys"`
	VectorizedAt time.Time `json:"vectorized_at"`
}

type persistentState struct {
	Files map[string]FileRecord `json:"files"`
}

func NewPlugin(cfg config.KnowledgePlugin) (*Plugin, error) {
	applyDefaults(&cfg)
	if !cfg.Enabled {
		return nil, errors.New("knowledge plugin is disabled")
	}
	if strings.TrimSpace(cfg.Embedding.BaseURL) == "" || strings.TrimSpace(cfg.Embedding.Model) == "" {
		return nil, errors.New("knowledge embedding config is incomplete")
	}
	if strings.TrimSpace(cfg.Redis.Addr) == "" {
		return nil, errors.New("knowledge redis.addr is required")
	}

	if err := os.MkdirAll(cfg.Dir, 0o755); err != nil {
		return nil, fmt.Errorf("create knowledge dir failed: %w", err)
	}

	statePath := cfg.StateFile
	if !filepath.IsAbs(statePath) {
		statePath = filepath.Join(cfg.Dir, statePath)
	}

	p := &Plugin{
		cfg:      cfg,
		stateDir: statePath,
		state: persistentState{
			Files: make(map[string]FileRecord),
		},
		rdb: redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		}),
		httpc: &http.Client{Timeout: 60 * time.Second},
	}

	if err := p.loadState(); err != nil {
		return nil, err
	}
	if err := p.rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return p, nil
}

func (p *Plugin) VectorizeFile(ctx context.Context, inputPath string) (*FileRecord, error) {
	mdPath, err := p.resolveMarkdownPath(inputPath)
	if err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(mdPath)
	if err != nil {
		return nil, fmt.Errorf("read markdown failed: %w", err)
	}
	content := string(raw)
	if strings.TrimSpace(content) == "" {
		return nil, errors.New("markdown file is empty")
	}

	checksum := sha256hex(raw)
	p.stateMu.Lock()
	existing, ok := p.state.Files[mdPath]
	p.stateMu.Unlock()
	if ok && existing.Checksum == checksum {
		return &existing, nil
	}

	chunks := splitMarkdown(content, p.cfg.ChunkSize, p.cfg.Overlap)
	if len(chunks) == 0 {
		return nil, errors.New("no chunks generated from markdown")
	}

	embeddings, err := p.embed(ctx, chunks)
	if err != nil {
		return nil, err
	}
	if len(embeddings) != len(chunks) {
		return nil, fmt.Errorf("embedding result count mismatch: got %d, want %d", len(embeddings), len(chunks))
	}
	vectorDim := len(embeddings[0])
	if p.cfg.Embedding.Dimension > 0 && p.cfg.Embedding.Dimension != vectorDim {
		return nil, fmt.Errorf("embedding dimension mismatch: got %d, config %d", vectorDim, p.cfg.Embedding.Dimension)
	}

	if err := p.ensureIndex(ctx, vectorDim); err != nil {
		return nil, err
	}

	if ok && len(existing.RedisKeys) > 0 {
		if err := p.rdb.Del(ctx, existing.RedisKeys...).Err(); err != nil {
			return nil, fmt.Errorf("delete old vectors failed: %w", err)
		}
	}

	redisKeys := make([]string, 0, len(chunks))
	for i, chunk := range chunks {
		key := p.chunkRedisKey(mdPath, checksum, i)
		vec := toFloat32Bytes(embeddings[i])
		fields := map[string]any{
			"file_path":  mdPath,
			"chunk_id":   i,
			"content":    chunk,
			"checksum":   checksum,
			"vector_dim": vectorDim,
			"embedding":  vec,
		}
		if err := p.rdb.HSet(ctx, key, fields).Err(); err != nil {
			return nil, fmt.Errorf("save vector to redis failed: %w", err)
		}
		redisKeys = append(redisKeys, key)
	}

	record := FileRecord{
		FilePath:     mdPath,
		Checksum:     checksum,
		ChunkCount:   len(chunks),
		VectorDim:    vectorDim,
		RedisKeys:    redisKeys,
		VectorizedAt: time.Now(),
	}

	p.stateMu.Lock()
	p.state.Files[mdPath] = record
	err = p.saveStateLocked()
	p.stateMu.Unlock()
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (p *Plugin) ListFiles() []FileRecord {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()

	out := make([]FileRecord, 0, len(p.state.Files))
	for _, v := range p.state.Files {
		out = append(out, v)
	}
	return out
}

func (p *Plugin) resolveMarkdownPath(inputPath string) (string, error) {
	var full string
	if filepath.IsAbs(inputPath) {
		full = filepath.Clean(inputPath)
	} else {
		full = filepath.Clean(filepath.Join(p.cfg.Dir, inputPath))
	}
	if strings.ToLower(filepath.Ext(full)) != ".md" {
		return "", errors.New("only .md files are supported")
	}
	return full, nil
}

func (p *Plugin) chunkRedisKey(path, checksum string, idx int) string {
	h := sha1.Sum([]byte(fmt.Sprintf("%s:%s:%d", path, checksum, idx)))
	return fmt.Sprintf("%s:chunk:%s", p.cfg.Redis.KeyPrefix, hex.EncodeToString(h[:]))
}

func (p *Plugin) embed(ctx context.Context, chunks []string) ([][]float64, error) {
	url := strings.TrimRight(p.cfg.Embedding.BaseURL, "/") + "/embeddings"

	batchSize := p.cfg.Embedding.BatchSize
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}

	result := make([][]float64, 0, len(chunks))
	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}

		reqBody := map[string]any{
			"model": p.cfg.Embedding.Model,
			"input": chunks[i:end],
		}
		body, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("marshal embedding request failed: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("create embedding request failed: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if strings.TrimSpace(p.cfg.Embedding.APIKey) != "" {
			req.Header.Set("Authorization", "Bearer "+p.cfg.Embedding.APIKey)
		}

		resp, err := p.httpc.Do(req)
		if err != nil {
			return nil, fmt.Errorf("embedding request failed: %w", err)
		}
		raw, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("embedding request failed with status %d: %s", resp.StatusCode, string(raw))
		}

		var parsed struct {
			Data []struct {
				Index     int       `json:"index"`
				Embedding []float64 `json:"embedding"`
			} `json:"data"`
		}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil, fmt.Errorf("parse embedding response failed: %w", err)
		}
		if len(parsed.Data) == 0 {
			return nil, errors.New("embedding response contains no data")
		}
		for _, item := range parsed.Data {
			result = append(result, item.Embedding)
		}
	}

	return result, nil
}

func (p *Plugin) ensureIndex(ctx context.Context, dim int) error {
	if strings.TrimSpace(p.cfg.Redis.IndexName) == "" {
		return nil
	}

	listRes, err := p.rdb.Do(ctx, "FT._LIST").Result()
	if err == nil {
		if list, ok := listRes.([]any); ok {
			for _, item := range list {
				if name, ok := item.(string); ok && name == p.cfg.Redis.IndexName {
					return nil
				}
			}
		}
	}

	args := []any{
		"FT.CREATE", p.cfg.Redis.IndexName,
		"ON", "HASH",
		"PREFIX", 1, p.cfg.Redis.KeyPrefix + ":chunk:",
		"SCHEMA",
		"file_path", "TEXT",
		"content", "TEXT",
		"checksum", "TAG",
		"embedding", "VECTOR", "HNSW", 6,
		"TYPE", "FLOAT32",
		"DIM", dim,
		"DISTANCE_METRIC", strings.ToUpper(p.cfg.Redis.DistanceMetric),
	}

	if err := p.rdb.Do(ctx, args...).Err(); err != nil {
		// Index already exists in race scenarios.
		if strings.Contains(strings.ToLower(err.Error()), "index already exists") {
			return nil
		}
		return fmt.Errorf("create redis vector index failed: %w", err)
	}
	return nil
}

func (p *Plugin) loadState() error {
	raw, err := os.ReadFile(p.stateDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read knowledge state failed: %w", err)
	}
	var st persistentState
	if err := json.Unmarshal(raw, &st); err != nil {
		return fmt.Errorf("unmarshal knowledge state failed: %w", err)
	}
	if st.Files == nil {
		st.Files = make(map[string]FileRecord)
	}
	p.state = st
	return nil
}

func (p *Plugin) saveStateLocked() error {
	data, err := json.MarshalIndent(p.state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal knowledge state failed: %w", err)
	}
	tmp := p.stateDir + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write knowledge state temp file failed: %w", err)
	}
	if err := os.Rename(tmp, p.stateDir); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace knowledge state failed: %w", err)
	}
	return nil
}

func applyDefaults(cfg *config.KnowledgePlugin) {
	if strings.TrimSpace(cfg.Dir) == "" {
		cfg.Dir = "./knowlege/"
	}
	if strings.TrimSpace(cfg.StateFile) == "" {
		cfg.StateFile = defaultStateFile
	}
	if cfg.ChunkSize <= 0 {
		cfg.ChunkSize = defaultChunkSize
	}
	if cfg.Overlap < 0 {
		cfg.Overlap = 0
	}
	if cfg.Overlap >= cfg.ChunkSize {
		cfg.Overlap = defaultOverlap
	}
	if strings.TrimSpace(cfg.Redis.KeyPrefix) == "" {
		cfg.Redis.KeyPrefix = defaultKeyPrefix
	}
	if strings.TrimSpace(cfg.Redis.DistanceMetric) == "" {
		cfg.Redis.DistanceMetric = defaultMetricName
	}
	if cfg.Embedding.BatchSize <= 0 {
		cfg.Embedding.BatchSize = defaultBatchSize
	}
}

func splitMarkdown(content string, chunkSize, overlap int) []string {
	if chunkSize <= 0 {
		chunkSize = defaultChunkSize
	}
	if overlap < 0 {
		overlap = 0
	}
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	paras := strings.Split(normalized, "\n\n")
	out := make([]string, 0, len(paras))

	var b strings.Builder
	for _, p := range paras {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if b.Len() > 0 && b.Len()+2+len(p) > chunkSize {
			chunk := strings.TrimSpace(b.String())
			if chunk != "" {
				out = append(out, chunk)
			}
			tail := tailRunes(chunk, overlap)
			b.Reset()
			if tail != "" {
				b.WriteString(tail)
			}
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(p)
	}
	last := strings.TrimSpace(b.String())
	if last != "" {
		out = append(out, last)
	}
	return out
}

func tailRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[len(runes)-n:])
}

func sha256hex(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func toFloat32Bytes(vec []float64) []byte {
	buf := bytes.NewBuffer(make([]byte, 0, len(vec)*4))
	for _, v := range vec {
		bits := math.Float32bits(float32(v))
		_ = binary.Write(buf, binary.LittleEndian, bits)
	}
	return buf.Bytes()
}
