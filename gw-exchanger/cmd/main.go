package main

import (
	"fmt"
	"gw-exchanger/internal/repository/postgres"
	"log"
	"net"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	proto "github.com/lynxbites/proto-grpc/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var db *pgxpool.Pool
var connStr string

func init() {
	err := godotenv.Load("config.env")
	if err != nil {
		log.Fatal(err)
	}

	connStr = fmt.Sprintf("postgres://%v:%v@%v:%v/%v?sslmode=disable", os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_HOST"), os.Getenv("POSTGRES_PORT"), os.Getenv("POSTGRES_DB"))
	db, err = pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatal(err)
	}
	// m, err := migrate.New("file://internal/migrations/", connStr)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// err = m.Up()
	// if err != nil {
	// 	log.Fatal(err)
	// }

}

func main() {
	repo, err := postgres.NewPostgresRepo(connStr)
	if err != nil {
		panic(err)
	}
	lis, err := net.Listen("tcp", ":8020")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	proto.RegisterExchangeServiceServer(s, &server{ExchangeRepo: repo})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
