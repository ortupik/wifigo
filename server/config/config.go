package config

import (
	"fmt"
	"github.com/spf13/viper"
)

func Config() error {
	// Load general config
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.MergeInConfig(); err != nil {
		return fmt.Errorf("failed to read config.yaml: %w", err)
	}

	// Load mpesa config
	viper.SetConfigName("mpesa")
	viper.AddConfigPath("./config")
	if err := viper.MergeInConfig(); err != nil {
		return fmt.Errorf("failed to read mpesa.yaml: %w", err)
	}

	return nil
}

func GetConfig() *viper.Viper {
	return viper.GetViper()
}
