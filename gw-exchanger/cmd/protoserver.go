package main

import (
	"context"
	"errors"
	"gw-exchanger/internal/repository"
	"log/slog"
	"time"

	proto "github.com/lynxbites/proto-grpc/proto"
)

type server struct {
	proto.UnimplementedExchangeServiceServer
	repository.ExchangeRepo
	cachedRates map[string]float64
	LastUpdate  time.Time
}

func (server *server) GetExchangeRates(ctx context.Context, request *proto.Empty) (*proto.ExchangeRatesResponse, error) {
	slog.Info("new request: received GetRates request")
	if !time.Now().After(server.LastUpdate.Add(time.Minute * 5)) {
		slog.Info("ok: sending cached rates")
		return &proto.ExchangeRatesResponse{Rates: server.cachedRates}, nil
	}

	newRates, err := server.ExchangeRepo.GetRates()
	if err != nil {
		return nil, err
	}
	server.LastUpdate = time.Now()
	server.cachedRates = newRates
	slog.Info("ok: get rates request fulfilled")
	return &proto.ExchangeRatesResponse{Rates: newRates}, nil
}

func (server *server) GetExchangeRateForCurrency(ctx context.Context, request *proto.CurrencyRequest) (*proto.ExchangeRateResponse, error) {

	slog.Info("new request: received GetRates request, %v to %v\n", request.FromCurrency, request.ToCurrency)
	if request.FromCurrency != "USD" && request.FromCurrency != "RUB" && request.FromCurrency != "EUR" {
		slog.Info("bad request: invalid currency")
		return nil, errors.New("Invalid currency")
	}
	if request.ToCurrency != "USD" && request.ToCurrency != "RUB" && request.ToCurrency != "EUR" {
		slog.Info("bad request: invalid currency")
		return nil, errors.New("Invalid currency")
	}

	rate, err := server.ExchangeRepo.Exchange(request.FromCurrency, request.ToCurrency)
	if err != nil {
		return nil, err
	}
	slog.Info("ok: get exchange rate for currency request fulfilled")
	return &proto.ExchangeRateResponse{
		FromCurrency: request.FromCurrency,
		ToCurrency:   request.ToCurrency,
		Rate:         rate,
	}, nil
}
