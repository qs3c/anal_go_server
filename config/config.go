package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server       ServerConfig       `mapstructure:"server"`
	Database     DatabaseConfig     `mapstructure:"database"`
	Redis        RedisConfig        `mapstructure:"redis"`
	JWT          JWTConfig          `mapstructure:"jwt"`
	OSS          OSSConfig          `mapstructure:"oss"`
	OAuth        OAuthConfig        `mapstructure:"oauth"`
	Email        EmailConfig        `mapstructure:"email"`
	Queue        QueueConfig        `mapstructure:"queue"`
	CORS         CORSConfig         `mapstructure:"cors"`
	Subscription SubscriptionConfig `mapstructure:"subscription"`
	Models       []ModelConfig      `mapstructure:"models"`
	Upload       UploadConfig       `mapstructure:"upload"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

type OSSConfig struct {
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	BucketName      string `mapstructure:"bucket_name"`
	CDNDomain       string `mapstructure:"cdn_domain"`
}

type OAuthConfig struct {
	Github GithubOAuthConfig `mapstructure:"github"`
	Wechat WechatOAuthConfig `mapstructure:"wechat"`
}

type GithubOAuthConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURI  string `mapstructure:"redirect_uri"`
}

type WechatOAuthConfig struct {
	AppID       string `mapstructure:"app_id"`
	AppSecret   string `mapstructure:"app_secret"`
	RedirectURI string `mapstructure:"redirect_uri"`
}

type EmailConfig struct {
	SMTPHost string `mapstructure:"smtp_host"`
	SMTPPort int    `mapstructure:"smtp_port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
}

type QueueConfig struct {
	AnalysisQueue string `mapstructure:"analysis_queue"`
	MaxWorkers    int    `mapstructure:"max_workers"`
}

type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers"`
}

type SubscriptionConfig struct {
	Levels map[string]SubscriptionLevel `mapstructure:"levels"`
}

type SubscriptionLevel struct {
	DailyQuota int     `mapstructure:"daily_quota"`
	MaxDepth   int     `mapstructure:"max_depth"`
	Price      float64 `mapstructure:"price"`
}

type ModelConfig struct {
	Name          string `mapstructure:"name"`
	DisplayName   string `mapstructure:"display_name"`
	RequiredLevel string `mapstructure:"required_level"`
	APIKey        string `mapstructure:"api_key"`
	APIProvider   string `mapstructure:"api_provider"`
	Description   string `mapstructure:"description"`
}

type UploadConfig struct {
	MaxSize           int64    `mapstructure:"max_size"`           // 最大文件大小（字节）
	TempDir           string   `mapstructure:"temp_dir"`           // 临时目录
	ExpireHours       int      `mapstructure:"expire_hours"`       // 过期时间（小时）
	AllowedExtensions []string `mapstructure:"allowed_extensions"` // 允许的扩展名
}

func Load(configPath string) (*Config, error) {
	// 优先尝试读取 config.local.yaml（包含真实密钥，不提交到git）
	dir := filepath.Dir(configPath)
	localConfigPath := filepath.Join(dir, "config.local.yaml")

	// 检查 config.local.yaml 是否存在
	if _, err := os.Stat(localConfigPath); err == nil {
		configPath = localConfigPath
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 环境变量覆盖
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
