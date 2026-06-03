package config

import (
	"fmt"
	"sync"

	"github.com/clodoaldomarques/core-sdk/pkg/env"
)

type Config struct {
	AppPort            int
	MySqlDBUser        string
	MySqlDBPass        string
	MySqlDBHost        string
	MySqlDBPort        string
	MysqlDBName        string
	AwsAddress         string
	AwsRegion          string
	AwsAccessKeyID     string
	AwsSecretAccessKey string
	BalancesSNSTopic   string
	BalancesSQSQueue   string
}

type Option func(*Config)

var (
	singleton sync.Once
	instance  *Config
)

func New(options ...Option) *Config {
	singleton.Do(func() {
		instance = &Config{
			AppPort:            env.GetInt("APP_PORT", 5000),
			MySqlDBUser:        env.GetString("MYSQL_USER", ""),
			MySqlDBPass:        env.GetString("MYSQL_PASSWORD", ""),
			MySqlDBHost:        env.GetString("MYSQL_HOST", "192.168.49.2"),
			MySqlDBPort:        env.GetString("MYSQL_PORT", "30001"),
			MysqlDBName:        env.GetString("MYSQL_DATABASE", "balances"),
			AwsAddress:         env.GetString("AWS_ADDRESS", ""),
			AwsRegion:          env.GetString("AWS_REGION", ""),
			AwsAccessKeyID:     env.GetString("AWS_ACCESS_KEY_ID", ""),
			AwsSecretAccessKey: env.GetString("AWS_SECRET_ACCESS_KEY", ""),
			BalancesSNSTopic:   env.GetString("BALANCES_SNS_TOPIC", ""),
			BalancesSQSQueue:   env.GetString("BALANCES_SQS_QUEUE", ""),
		}
	})

	for _, optFunc := range options {
		optFunc(instance)
	}

	return instance
}

func WithAppPort(appPort int) Option {
	return func(c *Config) {
		c.AppPort = appPort
	}
}

func WithMySqlDBUser(mySqlDBUser string) Option {
	return func(c *Config) {
		c.MySqlDBUser = mySqlDBUser
	}
}

func WithMySqlDBPass(MySqlDBPass string) Option {
	return func(c *Config) {
		c.MySqlDBPass = MySqlDBPass
	}
}

func WithMySqlDBHost(MySqlDBHost string) Option {
	return func(c *Config) {
		c.MySqlDBHost = MySqlDBHost
	}
}
func WithMySqlDBPort(MySqlDBPort string) Option {
	return func(c *Config) {
		c.MySqlDBPort = MySqlDBPort
	}
}
func WithMysqlDBName(MysqlDBName string) Option {
	return func(c *Config) {
		c.MysqlDBName = MysqlDBName
	}
}
func WithAwsAddress(AwsAddress string) Option {
	return func(c *Config) {
		c.AwsAddress = AwsAddress
	}
}
func WithAwsRegion(AwsRegion string) Option {
	return func(c *Config) {
		c.AwsRegion = AwsRegion
	}
}

func WithSnsAccountTopic(SnsAccountTopic string) Option {
	return func(c *Config) {
		c.BalancesSNSTopic = SnsAccountTopic
	}
}

func (c Config) Region() string {
	return c.AwsRegion
}

func (c Config) Address() string {
	return c.AwsAddress
}
func (c Config) AccessKeyID() string {
	return c.AwsAccessKeyID
}
func (c Config) SecretAccessKey() string {
	return c.AwsSecretAccessKey
}

func (c Config) TopicARN() string {
	return c.BalancesSNSTopic
}

func (c Config) GetMySQLConnectionString() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
		c.MySqlDBUser,
		c.MySqlDBPass,
		c.MySqlDBHost,
		c.MySqlDBPort,
		c.MysqlDBName,
	)
}
