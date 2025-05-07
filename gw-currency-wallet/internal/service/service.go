package service

import (
	"context"
	"gw-wallet/internal/repository"
	"log/slog"

	"github.com/labstack/echo/v4"
	proto "github.com/lynxbites/proto-grpc/proto"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Service struct {
	repo repository.WalletRepo
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

func NewService(repo repository.WalletRepo) *Service {
	return &Service{repo: repo}
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
	return service.repo.Deposit(ctx, request)
}

func (service *Service) Withdraw(ctx echo.Context, request *repository.WithdrawRequest) (*repository.WithdrawResponse, error) {
	return service.repo.Withdraw(ctx, request)
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
