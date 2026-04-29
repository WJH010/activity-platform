package config

import (
	"time"
)

// Config 主配置结构
type Config struct {
	App      AppConfig      `yaml:"app"`
	Database DatabaseConfig `yaml:"database"`
	MinIO    MinIOConfig    `yaml:"minio"`
	Wechat   WechatConfig   `yaml:"wechat"` // 添加 Wechat 字段
	JWT      JWTConfig      `yaml:"jwt"`
	Agent    AgentConfig    `yaml:"agent"`
	RAG      RAGConfig      `yaml:"rag"`
}

// AppConfig 应用配置
type AppConfig struct {
	Name           string   `yaml:"name"`
	Env            string   `yaml:"env"`
	Port           int      `yaml:"port"`
	Debug          bool     `yaml:"debug"`
	AllowedOrigins []string `yaml:"cors.allowed_origins"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver                string        `yaml:"driver"`
	Host                  string        `yaml:"host"`
	Port                  int           `yaml:"port"`
	Username              string        `yaml:"username"`
	Password              string        `yaml:"password"`
	DBName                string        `yaml:"dbname"`
	MaxOpenConnections    int           `yaml:"max_open_connections"`
	MaxIdleConnections    int           `yaml:"max_idle_connections"`
	ConnectionMaxLifetime time.Duration `yaml:"connection_max_lifetime"`
}

// MinIOConfig MinIO配置
type MinIOConfig struct {
	Endpoint        string `yaml:"endpoint"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	UseSSL          bool   `yaml:"use_ssl"`
	BucketName      string `yaml:"bucket_name"`
}

// WechatConfig 微信配置
type WechatConfig struct {
	AppID     string `yaml:"app_id"`
	AppSecret string `yaml:"app_secret"`
}

// JWTConfig JWT 配置
type JWTConfig struct {
	JwtSecret        string `yaml:"jwt_secret"`
	ExpirationHours  int    `yaml:"expiration_hours"`
	RefreshSecret    string `yaml:"refresh_secret"`
	SessionKeySecret string `yaml:"session_key_secret"`
}

// AgentConfig Agent模块配置
type AgentConfig struct {
	DefaultProvider    string `yaml:"default_provider"`
	MaxReactRounds     int    `yaml:"max_react_rounds"`
	MaxHistoryMessages int    `yaml:"max_history_messages"`
	InternalAPIBaseURL string `yaml:"internal_api_base_url"`
}

// RAGConfig RAG语义检索配置
type RAGConfig struct {
	Milvus    MilvusConfig    `yaml:"milvus"`
	Embedding EmbeddingConfig `yaml:"embedding"`
	Sync      SyncConfig      `yaml:"sync"`
}

// MilvusConfig Milvus连接配置
type MilvusConfig struct {
	Host   string `yaml:"host"`
	Port   int    `yaml:"port"`
	DBName string `yaml:"db_name"`
}

// EmbeddingConfig 向量嵌入配置
type EmbeddingConfig struct {
	Provider  string `yaml:"provider"`
	ApiURL    string `yaml:"api_url"`
	Dimension int    `yaml:"dimension"`
	BatchSize int    `yaml:"batch_size"`
}

// SyncConfig 数据同步配置
type SyncConfig struct {
	IncrementInterval int `yaml:"increment_interval"`
}
