package main

import (
	"context"
	"fmt"
	"gw-wallet/internal/handler"
	"gw-wallet/internal/repository/postgres"
	"gw-wallet/internal/service"
	"gw-wallet/internal/types"
	"log"
	"os"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/golang-migrate/migrate/v4/database/pgx"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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
	var newservice service.Service
	dbType := os.Getenv("DB_TYPE")
	switch dbType {
	case "postgres":
		repo, err := postgres.NewPostgresRepo(connStr)
		if err != nil {
			panic(err)
		}
		newservice = *service.NewService(repo)
	default:
		panic("database unsupported")
	}

	handler := handler.NewHandler(&newservice)

	config := echojwt.Config{
		SigningKey: []byte(os.Getenv("SIGNING_KEY")),
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return new(types.JwtClaims)
		},
	}

	e := echo.New()
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "INFO ${method}:${path} ${status} ${error} ${latency_human} \n",
	}))

	e.POST("/api/v1/register", handler.Register)
	e.POST("/api/v1/login", handler.Login)
	e.GET("/api/v1/balance", handler.GetBalance, echojwt.WithConfig(config))
	e.POST("/api/v1/wallet/deposit", handler.Deposit, echojwt.WithConfig(config))
	e.POST("/api/v1/wallet/withdraw", handler.Withdraw, echojwt.WithConfig(config))

	e.GET("/api/v1/exchange/rates", handler.GetExchangeRates, echojwt.WithConfig(config))
	e.POST("/api/v1/exchange", handler.Exchange, echojwt.WithConfig(config))

	e.Start(":8000")
}
