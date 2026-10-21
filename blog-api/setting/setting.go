package setting

import (
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Database struct {
	User     string
	Password string
	Host     string
	Name     string
}

var MySQLSetting = &Database{
	User:     "root",
	Password: "lwnznxf5555",
	Host:     "127.0.0.1:3306",
	Name:     "db05",
}

type Agent struct {
	BaseURL               string
	ChatStreamPath        string
	UnifiedChatStreamPath string
	ArticleIndexPath      string
	ResponseHeaderTimeout time.Duration
	ArticleIndexTimeout   time.Duration
}

var AgentSetting = &Agent{
	BaseURL:               envOrDefault("AGENT_BASE_URL", "http://127.0.0.1:8081"),
	ChatStreamPath:        envOrDefault("AGENT_CHAT_STREAM_PATH", "/v1/rag/chat/stream"),
	UnifiedChatStreamPath: envOrDefault("AGENT_UNIFIED_CHAT_STREAM_PATH", "/v1/blog/assistant/stream"),
	ArticleIndexPath:      envOrDefault("AGENT_ARTICLE_INDEX_PATH", "/internal/v1/index/articles"),
	ResponseHeaderTimeout: 15 * time.Second,
	ArticleIndexTimeout:   90 * time.Second,
}

// InternalTools contains the shared secret used by the Agent when calling the
// Blog domain-tool endpoints. There is deliberately no default: an
// unconfigured deployment must fail closed.
type InternalTools struct {
	Token string
}

var InternalToolSetting = &InternalTools{
	Token: sharedEnvValue("BLOG_INTERNAL_TOKEN"),
}

const (
	OSSAccessKeyID     = ""
	OSSAccessKeySecret = ""
	OSSEndpoint        = ""
	OSSBucketName      = ""
	OSSBaseURL         = ""
)

var Mode = gin.ReleaseMode

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

// sharedEnvValue reads the process environment first, then a service-local
// or workspace-level .env file. It does not mutate the process environment.
func sharedEnvValue(key string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	for _, path := range []string{".env", "../.env"} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, rawLine := range strings.Split(string(data), "\n") {
			line := strings.TrimSpace(rawLine)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			name, value, ok := strings.Cut(line, "=")
			if ok && strings.TrimSpace(name) == key {
				return strings.Trim(strings.TrimSpace(value), `"'`)
			}
		}
	}
	return ""
}
