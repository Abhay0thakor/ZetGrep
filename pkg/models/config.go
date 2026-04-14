package models

import (
	"strings"

	"github.com/Abhay0thakor/ZetGrep/pkg/utils"
	"github.com/spf13/viper"
)

func LoadConfig(configFiles []string) (Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("patterns_dir", "~/.config/gf/patterns")
	v.SetDefault("tools_dir", "~/.config/gf/tools")

	// Environment variables
	v.SetEnvPrefix("ZetGrep")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Load files in order
	for _, cf := range configFiles {
		v.SetConfigFile(utils.ExpandPath(cf))
		if err := v.MergeInConfig(); err != nil {
			// If file not found, it's okay if it's the default one, 
			// but here we only pass files that were explicitly requested or exist.
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return cfg, err
	}

	// Expand paths
	cfg.PatternsDir = utils.ExpandPath(cfg.PatternsDir)
	cfg.ToolsDir = utils.ExpandPath(cfg.ToolsDir)

	return cfg, nil
}
