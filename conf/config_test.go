package conf_test

import (
	"testing"

	"t-api/conf"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	config, err := conf.ReadConfig("config.yaml")
	if err != nil {
		t.Errorf("ReadConfig() error = %v", err)
	}

	if config == nil {
		t.Errorf("ReadConfig() config is nil")
	}
}
