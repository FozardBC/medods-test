package config

import (
	"log"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Log          string `env:"LOG_MODE"`
	ServerHost   string `env:"SRV_HOST"`
	ServerPort   string `env:"SRV_PORT" env-default:"8080"`
	DbConnString string `env:"DB_CONN_STRING, required"`
}

func MustRead() *Config {

	if err := godotenv.Load(); err != nil { // DEBUG: "../../.env"
		log.Print("WARN: CAN'T LOAD .env  TRY TO USE env variabels:%w", err.Error())
	}

	cfg := Config{}
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		help, _ := cleanenv.GetDescription(cfg, nil)
		log.Print(help)
		log.Fatal(err)
	}

	return &cfg
}
