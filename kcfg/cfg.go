package kcfg

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

func Load(path string) (Config, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return DefaultConfig, err
	}

	cfg := DefaultConfig
	err = yaml.Unmarshal(bytes, &cfg)
	if err != nil {
		return DefaultConfig, err
	}

	return cfg, nil
}

func GenerateConfig() error {
	bytes, err := yaml.Marshal(DefaultConfig)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile("kconfig.yaml", bytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

var DefaultConfig = Config{
	Net: Net{
		Scheme: "http",
		Host:   "127.0.0.1",
		Port:   8080,
	},
	Db: DB{
		Driver: "postgres",

		Name:    "keeper",
		User:    "postgres",
		Pass:    "postgres",
		Host:    "localhost",
		Port:    5432,
		SSLMode: "disable",
	},
	Log: Log{
		Level:        "info",
		LogToConsole: true,
		StacktraceDepth: StacktraceDepth{
			Error: 1,
			Warn:  0,
			Info:  0,
			Debug: 1,
		},
	},
}

type Config struct {
	Db  DB  `yaml:"db"`
	Net Net `yaml:"net"`
	Log Log `yaml:"log"`
}

type Net struct {
	Scheme string `yaml:"scheme"`
	Host   string `yaml:"host"`
	Port   uint16 `yaml:"port"`

	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

func (n Net) GetConnectionString() string {
	return fmt.Sprintf("%s:%d", n.Host, n.Port)
}

func (n Net) GetHttpConnectionString() string {
	return fmt.Sprintf("%s:%d", n.Host, n.Port+1)
}

type DB struct {
	Driver   string `yaml:"driver"`
	SSLMode  string `yaml:"ssl_mode"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`

	CustomConnectionString string `yaml:"custom_connection_string"`

	Name string `yaml:"name"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
	Host string `yaml:"host"`
	Port uint16 `yaml:"port"`
}

func (d DB) GetConnectionString() string {
	if d.CustomConnectionString != "" {
		return d.CustomConnectionString
	}

	return fmt.Sprintf(
		"user='%s' password='%s' host='%s' port=%d database='%s' sslmode='%s' sslcert='%s' sslkey='%s'",
		d.User, d.Pass, d.Host, d.Port, d.Name, d.SSLMode, d.CertFile, d.KeyFile,
	)
}

type Log struct {
	Level           string          `yaml:"level"`
	LogToConsole    bool            `yaml:"log_to_console"`
	StacktraceDepth StacktraceDepth `yaml:"stacktrace_depth"`
}

type StacktraceDepth struct {
	Error int `yaml:"error"`
	Warn  int `yaml:"warn"`
	Info  int `yaml:"info"`
	Debug int `yaml:"debug"`
}

func (s StacktraceDepth) Slice() []int {
	return []int{s.Error, s.Warn, s.Info, s.Debug}
}
