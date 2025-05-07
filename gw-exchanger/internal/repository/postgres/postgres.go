package postgres

import (
	"context"
	"log"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	db *pgxpool.Pool
}

type Rates struct {
	USD float64
	RUB float64
	EUR float64
}

func NewPostgresRepo(connStr string) (*PostgresRepo, error) {

	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return &PostgresRepo{}, err
	}
	return &PostgresRepo{db: pool}, nil

}

func (repo *PostgresRepo) GetRates() (map[string]float64, error) {

	rates := new(Rates)
	err := repo.db.QueryRow(context.Background(), "select (usd, rub, eur) from rates").Scan(rates)
	if err != nil {
		slog.Error("internal server error: cannot scan into Rates")
		return nil, err
	}
	return map[string]float64{
		"USD": rates.USD,
		"RUB": rates.RUB,
		"EUR": rates.EUR,
	}, nil
}

func (repo *PostgresRepo) Exchange(from string, to string) (float64, error) {

	rates := new(Rates)
	err := repo.db.QueryRow(context.Background(), "select (usd, rub, eur) from rates").Scan(rates)
	if err != nil {
		slog.Error("internal server error: cannot scan into Rates")
		return 0, err
	}

	ratesMap := map[string]float64{
		"USD": rates.USD,
		"RUB": rates.RUB,
		"EUR": rates.EUR,
	}

	rate := ratesMap[to] / ratesMap[from]
	log.Printf("Converted %v to %v, exchange rate - %v", from, to, rate)
	return rate, nil
}
