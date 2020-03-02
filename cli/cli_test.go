package cli

import (
	"testing"

	"github.com/WOo0W/bowerbird/cli/log"

	"github.com/WOo0W/bowerbird/config"
)

func TestConfig(t *testing.T) {
	log.G.ConsoleLevel = log.DEBUG

	conf := config.New()
	err := loadConfigFile(conf, "")
	if err != nil {
		t.Error(err)
	}

	// config.test.json
	// conf = config.New()
	// t.Error(loadConfigFile(conf, path))

}
