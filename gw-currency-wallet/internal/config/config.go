package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DbConfig     dbConfig
	RabbitConfig rabbitConfig
}

type dbConfig struct {
	Address string
}

type rabbitConfig struct {
	Address        string
	RabbitExchange string
	RabbitQueue    string
	RabbitRouting  string
}

func NewConfig() (*Config, error) {

	err := godotenv.Load("config.env")

	if err != nil {
		return nil, err
	}

	storage := Config{
		DbConfig: dbConfig{
			Address: os.Getenv("DB_CONN"),
		},
		RabbitConfig: rabbitConfig{
			Address:        os.Getenv("RMQ_CONN"),
			RabbitExchange: os.Getenv("RMQ_EXCHANGE"),
			RabbitQueue:    os.Getenv("RMQ_QUEUE"),
			RabbitRouting:  os.Getenv("RMQ_ROUTINGKEY"),
		},
	}
	return &storage, nil
}
