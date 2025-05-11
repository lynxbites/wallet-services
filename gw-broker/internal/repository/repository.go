package repository

type Repository interface {
	StoreOperation(id string, amount string) error
}
type InsertStruct struct {
	Id            string  `bson:"_id"`
	User          string  `bson:"user"`
	OperationType string  `bson:"operation_type"`
	Amount        float64 `bson:"amount"`
	Timestamp     string  `bson:"timestamp"`
}
