package main

import (
	"gw-wallet/internal/config"
	"gw-wallet/internal/handler"
	"gw-wallet/internal/repository/postgres"
	"gw-wallet/internal/service"
	"gw-wallet/internal/types"
	"log/slog"
	"os"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/golang-migrate/migrate/v4/database/pgx"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		slog.Error("Error loading .env file", "error", err.Error())
		return
	}

	var newservice service.Service
	dbType := os.Getenv("DB_TYPE")
	switch dbType {
	case "postgres":
		repo, err := postgres.NewPostgresRepo(cfg.DbConfig.Address)
		if err != nil {
			panic(err)
		}
		newservice = *service.NewService(repo, cfg)
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
