package service

import (
	"context"
	"gw-wallet/internal/config"
	"gw-wallet/internal/repository"
	"gw-wallet/internal/types"
	"log/slog"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	proto "github.com/lynxbites/proto-grpc/proto"
	amqp "github.com/rabbitmq/amqp091-go"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Service struct {
	repo   repository.WalletRepo
	rabbit *RabbitConn
}

type GetRatesResponse struct {
	Rates struct {
		USD float64 `json:"USD"`
		RUB float64 `json:"RUB"`
		EUR float64 `json:"EUR"`
	} `json:"rates"`
}

type GetExchangeRateRequest struct {
	From string
	To   string
}

type GetExchangeRateResponse struct {
	From string
	To   string
	Rate float64
}

func NewService(repo repository.WalletRepo, cfg *config.Config) *Service {
	conn, err := amqp.Dial(cfg.RabbitConfig.Address)
	if err != nil {
		slog.Error("rabbitmq connection: could not connect to rabbitmq")
	}
	return &Service{repo: repo, rabbit: &RabbitConn{conn: conn}}
}

func (service *Service) RegisterUser(ctx echo.Context, request *repository.RegisterRequest) error {
	return service.repo.RegisterUser(ctx, request)
}

func (service *Service) LoginUser(ctx echo.Context, request *repository.LoginRequest) (string, error) {
	return service.repo.LoginUser(ctx, request)
}

func (service *Service) GetBalance(ctx echo.Context) (*repository.BalanceResponse, error) {
	return service.repo.GetBalance(ctx)
}

func (service *Service) Deposit(ctx echo.Context, request *repository.DepositRequest) (*repository.DepositResponse, error) {
	response, err := service.repo.Deposit(ctx, request)
	if err != nil {
		slog.Info("rabbitmq send: transaction failed, skipping")
		return response, err
	}
	if request.Amount < 30000 {
		return response, err
	}

	user := ctx.Get("user").(*jwt.Token)
	claims := user.Claims.(*types.JwtClaims)

	rabbitErr := service.rabbit.SendData(ctx, claims.Username, "deposit", request.Amount)
	if rabbitErr != nil {
		slog.Error("send data to rabbit: error: " + rabbitErr.Error())

	}
	return response, err
}

func (service *Service) Withdraw(ctx echo.Context, request *repository.WithdrawRequest) (*repository.WithdrawResponse, error) {
	response, err := service.repo.Withdraw(ctx, request)
	if err != nil {
		slog.Info("rabbitmq send: transaction failed, skipping")
		return response, err
	}
	if request.Amount < 30000 {
		return response, err
	}

	user := ctx.Get("user").(*jwt.Token)
	claims := user.Claims.(*types.JwtClaims)

	rabbitErr := service.rabbit.SendData(ctx, claims.Username, "withdraw", request.Amount)
	if rabbitErr != nil {
		slog.Error("send data to rabbit: error: " + rabbitErr.Error())

	}
	return response, err
}

func (service *Service) GetExchangeRates(ctx echo.Context) (*proto.ExchangeRatesResponse, error) {
	conn, err := grpc.NewClient("localhost:8020", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("Could not establish gRPC connection.")
		return nil, err
	}
	client := proto.NewExchangeServiceClient(conn)
	response, err := client.GetExchangeRates(context.Background(), &proto.Empty{})
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (service *Service) Exchange(ctx echo.Context, request *repository.ExchangeRequestClient) (*repository.ExchangeResponse, error) {

	if request.FromCurrency != "USD" && request.FromCurrency != "RUB" && request.FromCurrency != "EUR" {
		return nil, echo.ErrBadRequest
	}
	if request.ToCurrency != "USD" && request.ToCurrency != "RUB" && request.ToCurrency != "EUR" {
		return nil, echo.ErrBadRequest
	}

	conn, err := grpc.NewClient("localhost:8020", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	client := proto.NewExchangeServiceClient(conn)
	grpcResponse, err := client.GetExchangeRateForCurrency(context.Background(), &proto.CurrencyRequest{FromCurrency: request.FromCurrency, ToCurrency: request.ToCurrency})
	if err != nil {
		return nil, err
	}

	repoRequest := &repository.ExchangeRequest{
		FromCurrency: request.FromCurrency,
		ToCurrency:   request.ToCurrency,
		Amount:       request.Amount,
		Rate:         grpcResponse.Rate,
	}

	return service.repo.Exchange(ctx, repoRequest)
}
