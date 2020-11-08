package cli

import (
	"context"
	"testing"

	"github.com/WOo0W/bowerbird/cli/log"

	"github.com/WOo0W/bowerbird/config"
)

func TestConfig(t *testing.T) {
	ctx := log.NewContext(context.Background(), log.New())

	conf := config.New()
	err := loadConfigFile(ctx, conf, "")
	if err != nil {
		t.Error(err)
	}

	// config.test.json
	// conf = config.New()
	// t.Error(loadConfigFile(conf, path))

}
