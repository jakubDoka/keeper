package cfg

import (
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

func Load(path string) (*Config, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	config := &Config{}
	err = yaml.Unmarshal(bytes, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

type Config struct {
	Email           Email `yaml:"email"`
	SessionDuration int   `yaml:"session_duration"`
}

type Email struct {
	Value    string `yaml:"value"`
	Password string `yaml:"password"`
}
