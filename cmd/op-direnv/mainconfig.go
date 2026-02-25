package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

func init() {
	viper.SetConfigName("op-direnv")
	viper.SetConfigType("toml")
	viper.AddConfigPath("${HOME}/.config")
	viper.AddConfigPath("/etc/op-direnv")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); !ok {
			fmt.Fprintf(os.Stderr, "error: reading config file: %v\n", err)
		}
	}

	viper.SetDefault("cache.path", "${HOME}/.cache/op-direnv")
	_ = viper.BindEnv("cache.path", "OP_DIRENV_CACHE_PATH")
}
