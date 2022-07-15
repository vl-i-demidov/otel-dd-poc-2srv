package main

import (
	"github.com/spf13/viper"
	"log"
	"os"
	"otel-dd-poc-2srv/internal/config"
	"otel-dd-poc-2srv/internal/server"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("Not enough args")
	}
	cfgFile := os.Args[1]
	cfg := ReadConfig(cfgFile)
	server.StartMain(cfg)
}

func ReadConfig(cfgFile string) config.Config {
	var cfg config.Config
	viper.SetConfigFile(cfgFile)

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalln("Couldn't read config at", cfgFile, ".", "Error:", err)
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalln("Couldn't read config at", cfgFile, ".", "Error:", err)
	}

	return cfg
}
