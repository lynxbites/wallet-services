package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gw-broker/internal/config"
	"gw-broker/internal/logger"
	"gw-broker/internal/repository"
	"gw-broker/internal/repository/mongodb"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {

	// init logger
	logger, err := logger.InitLogger("Rabbit-Broker")
	if err != nil {
		log.Fatal(err)
	}

	// init config
	cfg, err := config.NewConfig()
	if err != nil {
		logger.Error("Error loading .env file", "error", err.Error())
		return
	}

	// init mongoDB
	storage, err := mongodb.NewOperationStorage(cfg)
	if err != nil {
		logger.Error("Error connecting to MongoDB", "error", err.Error())
		return
	}
	defer storage.Collection.Database().Client().Disconnect(context.Background())

	logger.Info("Successfully connected to MongoDB")
	fmt.Printf("cfg.RabbitConfig.Address: %v\n", cfg.RabbitConfig)
	conn, err := amqp.Dial(cfg.RabbitConfig.Address)
	if err != nil {
		logger.Error("main: couldn't connect to amqp")
		panic(err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		logger.Error("main: couldn't open channel")
		panic(err)
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(
		cfg.RabbitConfig.RabbitExchange, // name
		"topic",                         // type
		true,                            // durable
		false,                           // auto-deleted
		false,                           // internal
		false,                           // no-wait
		nil,                             // arguments
	)
	if err != nil {
		logger.Error("main: failed to declare exchange")
		panic(err)
	}

	q, err := ch.QueueDeclare(
		cfg.RabbitConfig.RabbitQueue, // name
		false,                        // durable
		false,                        // delete when unused
		false,                        // exclusive
		false,                        // no-wait
		nil,                          // arguments
	)
	if err != nil {
		logger.Error("main: failed to declare queue")
		panic(err)
	}

	err = ch.QueueBind(
		q.Name,                         // queue name ("wallet_processor")
		cfg.RabbitConfig.RabbitRouting, // routing key pattern (topic filter)
		"wallet_events",                // exchange name (messages originate here)
		false,                          // no-wait
		nil,                            // arguments (optional)
	)
	if err != nil {
		logger.Error("main: failed to bind queue")
		panic(err)
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		logger.Error("main: failed to register a consumer")
		panic(err)
	}

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
			var data *repository.InsertStruct
			json.Unmarshal(d.Body, &data)
			fmt.Printf("data: %v\n", data)
			err := storage.StoreOperation(data)
			if err != nil {
				logger.Error("message receive: couldn't insert data")
			}
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	log.Println("Waiting for messages. To exit press CTRL+C. Press CTRL+C to exit.")
	<-sigChan
	log.Println("Shutting down consumer")
}
