package service

import (
	"context"
	"encoding/json"
	"gw-wallet/internal/config"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	amqp "github.com/rabbitmq/amqp091-go"
)

type InsertStruct struct {
	Id            string  `bson:"_id"`
	User          string  `bson:"user"`
	OperationType string  `bson:"operation_type"`
	Amount        float64 `bson:"amount"`
	Timestamp     string  `bson:"timestamp"`
}

type RabbitConn struct {
	conn *amqp.Connection
}

func (rabbit *RabbitConn) SendData(ctx echo.Context, userName string, operationType string, amount float64) error {
	cfg, err := config.NewConfig()
	if err != nil {
		panic(err)
	}
	channel, err := rabbit.conn.Channel()
	if err != nil {
		slog.Error("rabbitmq send: failed to create channel")
		return err
	}
	defer channel.Close()

	err = channel.ExchangeDeclare(
		cfg.RabbitConfig.RabbitExchange, // name
		"topic",                         // type
		true,                            // durable
		false,                           // auto-deleted
		false,                           // internal
		false,                           // no-wait
		nil,                             // arguments
	)
	if err != nil {
		slog.Error("rabbitmq send: failed to declare exchange")
		return err
	}

	rabbitctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	jsonByte, err := json.Marshal(InsertStruct{
		Id:            uuid.New().String(),
		User:          userName,
		OperationType: operationType,
		Amount:        amount,
		Timestamp:     time.Now().UTC().String(),
	})
	if err != nil {
		return err
	}

	err = channel.PublishWithContext(rabbitctx,
		cfg.RabbitConfig.RabbitExchange, // exchange
		"wallet.event."+operationType,   // routing key
		false,                           // mandatory
		false,                           // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        jsonByte,
		})
	if err != nil {
		slog.Error("rabbitmq send: failed to publish")
		return err
	}
	slog.Info(" [x] Sent " + string(jsonByte))
	return err
}
