package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoConn    mongoConfig
	RabbitConfig rabbitConfig
}

type mongoConfig struct {
	Address      string
	DBName       string
	DBCollection string
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
		MongoConn: mongoConfig{
			Address:      os.Getenv("DB_CONN"),
			DBName:       os.Getenv("DB_NAME"),
			DBCollection: os.Getenv("DB_COLLECTION"),
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
