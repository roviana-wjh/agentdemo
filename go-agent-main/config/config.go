package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	ServerConf ServerConfig

	// 模型类型配置
	ChatModelType      string
	IntentModelType    string
	EmbeddingModelType string
	VectorDBType       string

	ArkConf      ArkConfig
	OpenAIConf   OpenAIConfig
	QwenConf     QwenConfig
	DeepSeekConf DeepSeekConfig
	GeminiConf   GeminiConfig

	MilvusConf MilvusConfig
	ESConf     ESConfig

	LangSmithConf LangSmithConfig

	MySQLConf        MySQLConfig
	RedisConf        RedisConfig
	MemoryConf       MemoryConfig
	HybridSearchConf HybridSearchConfig
	BlogAPIConf      BlogAPIConfig
	AutoPublish      AutoPublishConfig

	FileDir   string
	CLSMCPURL string
}

type ServerConfig struct {
	Address     string
	OpenAPIPath string
	SwaggerPath string
}

type ArkConfig struct {
	ArkKey            string
	ArkEmbeddingModel string
	ArkChatModel      string
}

type OpenAIConfig struct {
	OpenAIKey       string
	OpenAIChatModel string
	OpenAIEmbedding string
}

type QwenConfig struct {
	BaseUrl       string
	QwenKey       string
	QwenChatModel string
	QwenEmbedding string
}

type DeepSeekConfig struct {
	BaseUrl           string
	DeepSeekKey       string
	DeepSeekChatModel string
	DeepSeekEmbedding string
	DeepSeekTimeout   string
}

type GeminiConfig struct {
	GeminiKey       string
	GeminiChatModel string
	GeminiEmbedding string
}

type MilvusConfig struct {
	MilvusAddr          string
	MilvusUserName      string
	MilvusPassword      string
	SimilarityThreshold string
	CollectionName      string
	TopK                string
}

type ESConfig struct {
	Addresses []string
	Username  string
	Password  string
	CloudID   string
	APIKey    string
	Index     string
}

type LangSmithConfig struct {
	APIKey string
	APIUrl string
}

type MySQLConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	Database string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       string
}

type MemoryConfig struct {
	KeyPrefix          string
	HistoryTTL         time.Duration
	SummaryTTL         time.Duration
	RecentMessageCount int
}

type HybridSearchConfig struct {
	ESTopK                 int
	MilvusTopK             int
	RRFK                   int
	RerankMode             string
	RerankCandidateK       int
	FinalTopK              int
	RerankMaxContentLength int
}

type BlogAPIConfig struct {
	BaseURL       string
	InternalToken string
	Timeout       time.Duration
}

type AutoPublishConfig struct {
	Enabled bool
	Cron    string
}

var Cfg *Config

type yamlModelConfig struct {
	APIKey     string `yaml:"api_key"`
	BaseURL    string `yaml:"base_url"`
	Model      string `yaml:"model"`
	Dimensions int    `yaml:"dimensions"`
}

type yamlConfig struct {
	Server struct {
		Address     string `yaml:"address"`
		OpenAPIPath string `yaml:"openapiPath"`
		SwaggerPath string `yaml:"swaggerPath"`
	} `yaml:"server"`

	ChatModelType      string `yaml:"chat_model_type"`
	IntentModelType    string `yaml:"intent_model_type"`
	EmbeddingModelType string `yaml:"embedding_model_type"`
	VectorDBType       string `yaml:"vector_db_type"`

	DSThinkChatModel  yamlModelConfig `yaml:"ds_think_chat_model"`
	DSQuickChatModel  yamlModelConfig `yaml:"ds_quick_chat_model"`
	DoubaoEmbedding   yamlModelConfig `yaml:"doubao_embedding_model"`
	ArkChatModel      yamlModelConfig `yaml:"ark_chat_model"`
	ArkEmbeddingModel yamlModelConfig `yaml:"ark_embedding_model"`
	QwenChatModel     yamlModelConfig `yaml:"qwen_chat_model"`
	QwenEmbedding     yamlModelConfig `yaml:"qwen_embedding_model"`

	Milvus struct {
		Address             string `yaml:"address"`
		Username            string `yaml:"username"`
		Password            string `yaml:"password"`
		Collection          string `yaml:"collection"`
		CollectionName      string `yaml:"collection_name"`
		TopK                int    `yaml:"top_k"`
		SimilarityThreshold string `yaml:"similarity_threshold"`
	} `yaml:"milvus"`

	Elasticsearch struct {
		Addresses []string `yaml:"addresses"`
		Address   string   `yaml:"address"`
		Username  string   `yaml:"username"`
		Password  string   `yaml:"password"`
		CloudID   string   `yaml:"cloud_id"`
		APIKey    string   `yaml:"api_key"`
		Index     string   `yaml:"index"`
	} `yaml:"elasticsearch"`

	MySQL struct {
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
	} `yaml:"mysql"`

	Redis struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       string `yaml:"db"`
	} `yaml:"redis"`

	Memory struct {
		KeyPrefix          string `yaml:"key_prefix"`
		HistoryTTL         string `yaml:"history_ttl"`
		SummaryTTL         string `yaml:"summary_ttl"`
		RecentMessageCount int    `yaml:"recent_message_count"`
	} `yaml:"memory"`

	HybridSearch struct {
		ESTopK                 int    `yaml:"es_top_k"`
		MilvusTopK             int    `yaml:"milvus_top_k"`
		RRFK                   int    `yaml:"rrf_k"`
		RerankMode             string `yaml:"rerank_mode"`
		RerankCandidateK       int    `yaml:"rerank_candidate_k"`
		FinalTopK              int    `yaml:"final_top_k"`
		RerankMaxContentLength int    `yaml:"rerank_max_content_length"`
	} `yaml:"hybrid_search"`

	BlogAPI struct {
		BaseURL       string `yaml:"base_url"`
		InternalToken string `yaml:"internal_token"`
		Timeout       string `yaml:"timeout"`
	} `yaml:"blog_api"`

	AutoPublish struct {
		Enabled bool   `yaml:"enabled"`
		Cron    string `yaml:"cron"`
	} `yaml:"auto_publish"`

	LangSmith struct {
		APIKey string `yaml:"api_key"`
		APIURL string `yaml:"api_url"`
	} `yaml:"langsmith"`

	FileDir   string `yaml:"file_dir"`
	CLSMCPURL string `yaml:"cls_mcp_url"`
}

func LoadConfig() (*Config, error) {
	rawYAML, err := loadYAMLConfig("config.yaml")
	if err != nil {
		return nil, err
	}

	// Environment variables have highest priority. For local development load
	// a service-local .env first, then the workspace-level shared .env.
	for _, envFile := range []string{".env", "../.env"} {
		if err := godotenv.Load(envFile); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Printf("warning: failed to load %s, continuing with other config sources: %v", envFile, err)
		}
	}

	rawAddr := firstNonEmpty(getEnv("ES_ADDRESS", ""), rawYAML.Elasticsearch.Address, "http://localhost:9200")

	// 按逗号分割成 []string
	esAddresses := strings.Split(rawAddr, ",")
	if len(rawYAML.Elasticsearch.Addresses) > 0 {
		esAddresses = rawYAML.Elasticsearch.Addresses
	}

	deepSeekChat := firstModel(rawYAML.DSQuickChatModel, rawYAML.DSThinkChatModel)
	doubaoEmbedding := rawYAML.DoubaoEmbedding

	chatModelType := firstNonEmpty(getEnv("CHAT_MODEL_TYPE", ""), rawYAML.ChatModelType)
	if chatModelType == "" && deepSeekChat.APIKey != "" {
		chatModelType = "deepseek"
	}
	intentModelType := firstNonEmpty(getEnv("INTENT_MODEL_TYPE", ""), rawYAML.IntentModelType, chatModelType)
	embeddingModelType := firstNonEmpty(getEnv("EMBEDDING_MODEL_TYPE", ""), rawYAML.EmbeddingModelType)
	if embeddingModelType == "" && doubaoEmbedding.APIKey != "" {
		// 当前 qwen embedder 走 OpenAI-compatible 协议，可直接适配 DashScope embedding。
		embeddingModelType = "qwen"
	}
	vectorDBType := firstNonEmpty(getEnv("VECTOR_DB_TYPE", ""), rawYAML.VectorDBType)
	if vectorDBType == "" {
		if hasMilvusConfig(rawYAML) || getEnv("MILVUS_ADDR", "") != "" {
			vectorDBType = "milvus"
		} else if hasESConfig(rawYAML) || getEnv("ES_ADDRESS", "") != "" {
			vectorDBType = "es"
		} else {
			vectorDBType = "none"
		}
	}

	config := &Config{
		ServerConf: ServerConfig{
			Address:     firstNonEmpty(getEnv("SERVER_ADDRESS", ""), rawYAML.Server.Address, ":8081"),
			OpenAPIPath: firstNonEmpty(rawYAML.Server.OpenAPIPath, "/api.json"),
			SwaggerPath: firstNonEmpty(rawYAML.Server.SwaggerPath, "/swagger"),
		},

		ChatModelType:      firstNonEmpty(chatModelType, "ark"),
		IntentModelType:    firstNonEmpty(intentModelType, "ark"),
		EmbeddingModelType: firstNonEmpty(embeddingModelType, "ark"),
		VectorDBType:       vectorDBType,

		ArkConf: ArkConfig{
			ArkKey:            firstNonEmpty(getEnv("ARK_KEY", ""), rawYAML.ArkChatModel.APIKey, rawYAML.ArkEmbeddingModel.APIKey),
			ArkEmbeddingModel: firstNonEmpty(getEnv("ARK_EMBEDDING_MODEL", ""), rawYAML.ArkEmbeddingModel.Model, "doubao-embedding-text-240715"),
			ArkChatModel:      firstNonEmpty(getEnv("ARK_CHAT_MODEL", ""), rawYAML.ArkChatModel.Model, "doubao-seed-1-8-251228"),
		},
		OpenAIConf: OpenAIConfig{
			OpenAIKey:       getEnv("OPENAI_KEY", ""),
			OpenAIChatModel: getEnv("OPENAI_CHAT_MODEL", "gpt-4"),
			OpenAIEmbedding: getEnv("OPENAI_EMBEDDING_MODEL", ""),
		},
		QwenConf: QwenConfig{
			BaseUrl:       firstNonEmpty(getEnv("QWEN_BASE_URL", ""), rawYAML.QwenChatModel.BaseURL, rawYAML.QwenEmbedding.BaseURL, doubaoEmbedding.BaseURL),
			QwenKey:       firstNonEmpty(getEnv("QWEN_KEY", ""), rawYAML.QwenChatModel.APIKey, rawYAML.QwenEmbedding.APIKey, doubaoEmbedding.APIKey),
			QwenEmbedding: firstNonEmpty(getEnv("QWEN_EMBEDDING_MODEL", ""), rawYAML.QwenEmbedding.Model, doubaoEmbedding.Model),
			QwenChatModel: firstNonEmpty(getEnv("QWEN_CHAT_MODEL", ""), rawYAML.QwenChatModel.Model),
		},
		DeepSeekConf: DeepSeekConfig{
			BaseUrl:           firstNonEmpty(getEnv("DEEPSEEK_BASE_URL", ""), getEnv("DeepSeek_BASE_URL", ""), deepSeekChat.BaseURL),
			DeepSeekKey:       firstNonEmpty(getEnv("DEEPSEEK_KEY", ""), getEnv("DeepSeek_KEY", ""), deepSeekChat.APIKey),
			DeepSeekTimeout:   firstNonEmpty(getEnv("DEEPSEEK_TIMEOUT", ""), getEnv("DeepSeek_TIMEOUT", "")),
			DeepSeekChatModel: firstNonEmpty(getEnv("DEEPSEEK_CHAT_MODEL", ""), getEnv("DeepSeek_CHAT_MODEL", ""), deepSeekChat.Model),
			DeepSeekEmbedding: firstNonEmpty(getEnv("DEEPSEEK_EMBEDDING_MODEL", ""), getEnv("DeepSeek_EMBEDDING_MODEL", "")),
		},
		GeminiConf: GeminiConfig{
			GeminiKey:       getEnv("GEMINI_KEY", ""),
			GeminiChatModel: getEnv("GEMINI_CHAT_MODEL", ""),
			GeminiEmbedding: getEnv("GEMINI_EMBEDDING_MODEL", ""),
		},
		MilvusConf: MilvusConfig{
			MilvusAddr:          firstNonEmpty(getEnv("MILVUS_ADDR", ""), rawYAML.Milvus.Address, "localhost:19530"),
			MilvusUserName:      firstNonEmpty(getEnv("MILVUS_USERNAME", ""), rawYAML.Milvus.Username),
			MilvusPassword:      firstNonEmpty(getEnv("MILVUS_PASSWORD", ""), rawYAML.Milvus.Password),
			SimilarityThreshold: firstNonEmpty(getEnv("MILVUS_SIMILARITY_THRESHOLD", ""), rawYAML.Milvus.SimilarityThreshold, "0.7"),
			CollectionName:      firstNonEmpty(getEnv("MILVUS_COLLECTION_NAME", ""), rawYAML.Milvus.CollectionName, rawYAML.Milvus.Collection, "GoAgent"),
			TopK:                firstNonEmpty(getEnv("TOPK", ""), intToString(rawYAML.Milvus.TopK), "10"),
		},
		ESConf: ESConfig{
			Addresses: esAddresses,
			Username:  firstNonEmpty(getEnv("ES_USERNAME", ""), rawYAML.Elasticsearch.Username),
			Password:  firstNonEmpty(getEnv("ES_PASSWORD", ""), rawYAML.Elasticsearch.Password),
			CloudID:   firstNonEmpty(getEnv("ES_CLOUD_ID", ""), rawYAML.Elasticsearch.CloudID),
			APIKey:    firstNonEmpty(getEnv("ES_API_KEY", ""), rawYAML.Elasticsearch.APIKey),
			Index:     firstNonEmpty(getEnv("ES_INDEX", ""), rawYAML.Elasticsearch.Index, "blog_article_chunks_v1"),
		},
		LangSmithConf: LangSmithConfig{
			APIKey: firstNonEmpty(getEnv("LANG_SMITH_KEY", ""), rawYAML.LangSmith.APIKey),
			APIUrl: firstNonEmpty(getEnv("LANG_SMITH_URL", ""), rawYAML.LangSmith.APIURL),
		},
		MySQLConf: MySQLConfig{
			Host:     firstNonEmpty(getEnv("MYSQL_HOST", ""), rawYAML.MySQL.Host, "localhost"),
			Port:     firstNonEmpty(getEnv("MYSQL_PORT", ""), rawYAML.MySQL.Port, "3306"),
			Username: firstNonEmpty(getEnv("MYSQL_USERNAME", ""), rawYAML.MySQL.Username),
			Password: firstNonEmpty(getEnv("MYSQL_PASSWORD", ""), rawYAML.MySQL.Password),
			Database: firstNonEmpty(getEnv("MYSQL_DATABASE", ""), rawYAML.MySQL.Database),
		},
		RedisConf: RedisConfig{
			Addr:     firstNonEmpty(getEnv("REDIS_ADDR", ""), rawYAML.Redis.Addr, "localhost:6379"),
			Password: firstNonEmpty(getEnv("REDIS_PASSWORD", ""), rawYAML.Redis.Password),
			DB:       firstNonEmpty(getEnv("REDIS_DB", ""), rawYAML.Redis.DB, "0"),
		},
		MemoryConf: MemoryConfig{
			KeyPrefix: firstNonEmpty(getEnv("MEMORY_KEY_PREFIX", ""), rawYAML.Memory.KeyPrefix, "ai-blog"),
			HistoryTTL: durationOrDefault(
				firstNonEmpty(getEnv("MEMORY_HISTORY_TTL", ""), rawYAML.Memory.HistoryTTL),
				24*time.Hour,
			),
			SummaryTTL: durationOrDefault(
				firstNonEmpty(getEnv("MEMORY_SUMMARY_TTL", ""), rawYAML.Memory.SummaryTTL),
				7*24*time.Hour,
			),
			RecentMessageCount: positiveIntOrDefault(
				firstNonEmpty(getEnv("MEMORY_RECENT_MESSAGE_COUNT", ""), intToString(rawYAML.Memory.RecentMessageCount)),
				8,
			),
		},
		HybridSearchConf: HybridSearchConfig{
			ESTopK: positiveIntOrDefault(
				firstNonEmpty(getEnv("HYBRID_ES_TOP_K", ""), intToString(rawYAML.HybridSearch.ESTopK)),
				20,
			),
			MilvusTopK: positiveIntOrDefault(
				firstNonEmpty(getEnv("HYBRID_MILVUS_TOP_K", ""), intToString(rawYAML.HybridSearch.MilvusTopK)),
				20,
			),
			RRFK: positiveIntOrDefault(
				firstNonEmpty(getEnv("HYBRID_RRF_K", ""), intToString(rawYAML.HybridSearch.RRFK)),
				60,
			),
			RerankMode: firstNonEmpty(getEnv("HYBRID_RERANK_MODE", ""), rawYAML.HybridSearch.RerankMode, "llm"),
			RerankCandidateK: positiveIntOrDefault(
				firstNonEmpty(getEnv("HYBRID_RERANK_CANDIDATE_K", ""), intToString(rawYAML.HybridSearch.RerankCandidateK)),
				10,
			),
			FinalTopK: positiveIntOrDefault(
				firstNonEmpty(getEnv("HYBRID_FINAL_TOP_K", ""), intToString(rawYAML.HybridSearch.FinalTopK)),
				5,
			),
			RerankMaxContentLength: positiveIntOrDefault(
				firstNonEmpty(getEnv("HYBRID_RERANK_MAX_CONTENT_LENGTH", ""), intToString(rawYAML.HybridSearch.RerankMaxContentLength)),
				800,
			),
		},
		BlogAPIConf: BlogAPIConfig{
			BaseURL:       firstNonEmpty(getEnv("BLOG_API_BASE_URL", ""), rawYAML.BlogAPI.BaseURL, "http://127.0.0.1:8080"),
			InternalToken: firstNonEmpty(getEnv("BLOG_INTERNAL_TOKEN", ""), rawYAML.BlogAPI.InternalToken),
			Timeout: durationOrDefault(
				firstNonEmpty(getEnv("BLOG_API_TIMEOUT", ""), rawYAML.BlogAPI.Timeout),
				10*time.Second,
			),
		},
		AutoPublish: AutoPublishConfig{
			Enabled: rawYAML.AutoPublish.Enabled,
			Cron:    firstNonEmpty(rawYAML.AutoPublish.Cron, "0 9 * * *"),
		},

		FileDir:   rawYAML.FileDir,
		CLSMCPURL: rawYAML.CLSMCPURL,
	}

	return config, nil
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func loadYAMLConfig(path string) (yamlConfig, error) {
	var cfg yamlConfig
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse %s: %w", path, err)
	}
	return cfg, nil
}

func firstModel(models ...yamlModelConfig) yamlModelConfig {
	for _, model := range models {
		if model.APIKey != "" || model.Model != "" || model.BaseURL != "" {
			return model
		}
	}
	return yamlModelConfig{}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func intToString(value int) string {
	if value <= 0 {
		return ""
	}
	return fmt.Sprintf("%d", value)
}

func durationOrDefault(value string, fallback time.Duration) time.Duration {
	duration, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil || duration <= 0 {
		return fallback
	}
	return duration
}

func positiveIntOrDefault(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func hasMilvusConfig(cfg yamlConfig) bool {
	return cfg.Milvus.Address != "" ||
		cfg.Milvus.Collection != "" ||
		cfg.Milvus.CollectionName != "" ||
		cfg.Milvus.Username != "" ||
		cfg.Milvus.Password != ""
}

func hasESConfig(cfg yamlConfig) bool {
	return cfg.Elasticsearch.Address != "" ||
		len(cfg.Elasticsearch.Addresses) > 0 ||
		cfg.Elasticsearch.Username != "" ||
		cfg.Elasticsearch.Password != "" ||
		cfg.Elasticsearch.Index != ""
}
