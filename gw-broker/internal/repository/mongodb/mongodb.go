package mongodb

import (
	"context"

	"gw-broker/internal/config"
	"gw-broker/internal/repository"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

type OperationStorage struct {
	Collection *mongo.Collection
}

func NewOperationStorage(cfg *config.Config) (*OperationStorage, error) {
	//client, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoConn))
	client, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoConn.Address))
	if err != nil {
		panic(err)
	}

	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		panic(err)
	}

	storage := client.Database("transactions").Collection("operations")
	return &OperationStorage{
		Collection: storage,
	}, nil
}

func (storage *OperationStorage) StoreOperation(data *repository.InsertStruct) error {
	_, err := storage.Collection.InsertOne(context.Background(), &data)
	if err != nil {
		return err
	}
	return nil
}
