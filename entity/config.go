package entity

import (
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/russianinvestments/invest-api-go-sdk/investgo"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Invest            investgo.Config `yaml:"Invest"`
	TgBot             TgBotConfig     `yaml:"TgBot"`
	Cache             Cache           `yaml:"Cache"`
	OperationsTimeout time.Duration   `yaml:"OperationsTimeout"`
	AesKey            []byte
}

type TgBotConfig struct {
	Debug bool `yaml:"Debug"`
}

type Cache struct {
	Filepath string `yaml:"Filepath"`
}

func ReadConfig(filename string) (*Config, error) {
	input, err := os.ReadFile(filename)
	if err != nil {
		return nil, errors.WithMessage(err, "read file")
	}

	var cfg Config
	err = yaml.Unmarshal(input, &cfg)
	if err != nil {
		return nil, errors.WithMessage(err, "unmarshal yaml")
	}

	return &cfg, nil
}
