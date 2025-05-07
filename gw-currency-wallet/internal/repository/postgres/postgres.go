package postgres

import (
	"context"
	"fmt"
	"gw-wallet/internal/repository"
	"gw-wallet/internal/types"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type PostgresRepo struct {
	db *pgxpool.Pool
}

func NewPostgresRepo(connStr string) (*PostgresRepo, error) {
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return &PostgresRepo{}, err
	}
	return &PostgresRepo{db: pool}, nil
}

func (repo *PostgresRepo) RegisterUser(ctx echo.Context, request *repository.RegisterRequest) error {

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), 12)
	if err != nil {
		slog.Error("internal error: cannot hash password: " + err.Error())
		return echo.ErrInternalServerError
	}

	_, err = repo.db.Exec(context.Background(), "insert into wallets (username, password_hash, email) values ($1, $2, $3)", request.Username, string(hashedPassword[:]), request.Email)
	if err != nil {
		slog.Error("internal error: cannot insert in db: " + err.Error())
		return err
	}

	return nil
}

func (repo *PostgresRepo) LoginUser(ctx echo.Context, request *repository.LoginRequest) (string, error) { //select exists (select username from wallets where (username = 'floppa' and password_hash = ''))
	var hashedPassword string

	repo.db.QueryRow(context.Background(), "select password_hash from wallets where username = $1", request.Username).Scan(&hashedPassword)
	// fmt.Printf("%v\n", hashedPassword)

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(request.Password)); err != nil {
		slog.Info("unauthorized: invalid password")
		return "", echo.ErrUnauthorized
	}

	newClaims := &types.JwtClaims{
		Username: request.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 1)),
		},
	}

	unsignedToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	secret := "secret_key0000000000000000000000"
	token, err := unsignedToken.SignedString([]byte(secret))
	if err != nil {
		slog.Error("internal server error: cannot sign string")
		return "", echo.ErrInternalServerError
	}

	return token, nil
}

func (repo *PostgresRepo) GetBalance(ctx echo.Context) (*repository.BalanceResponse, error) {

	user := ctx.Get("user").(*jwt.Token)
	if !user.Valid {
		slog.Info("unauthorized: invalid token")
		return nil, echo.ErrUnauthorized
	}

	claims := user.Claims.(*types.JwtClaims)

	balanceResponse := new(repository.BalanceResponse)
	err := repo.db.QueryRow(context.Background(), "select balance_usd, balance_rub, balance_eur from wallets where username = $1", claims.Username).Scan(&balanceResponse.Balance.USD, &balanceResponse.Balance.RUB, &balanceResponse.Balance.EUR)
	if err != nil {
		slog.Error("internal server error: cannot scan into repository.BalanceResponse.Balance")
		return nil, err
	}
	return balanceResponse, nil
}

func (repo *PostgresRepo) Deposit(ctx echo.Context, request *repository.DepositRequest) (*repository.DepositResponse, error) {
	slog.Info("new request: received request for deposit")
	user := ctx.Get("user").(*jwt.Token)
	if !user.Valid {
		slog.Info("unauthorized: invalid token")
		return nil, echo.ErrUnauthorized
	}
	if request.Amount <= 0 {
		slog.Info("bad request: invalid amount")
		return nil, echo.ErrBadRequest
	}

	claims := user.Claims.(*types.JwtClaims)

	switch request.Currency {
	case "USD":
		_, err := repo.db.Exec(context.Background(), "update wallets set balance_usd = balance_usd + $1 where username = $2", request.Amount, claims.Username)
		if err != nil {
			slog.Error("internal server error: cannot update postgres db")
			return nil, err
		}
	case "RUB":
		_, err := repo.db.Exec(context.Background(), "update wallets set balance_rub = balance_rub + $1 where username = $2", request.Amount, claims.Username)
		if err != nil {
			slog.Error("internal server error: cannot update postgres db")
			return nil, err
		}
	case "EUR":
		_, err := repo.db.Exec(context.Background(), "update wallets set balance_eur = balance_eur + $1 where username = $2", request.Amount, claims.Username)
		if err != nil {
			slog.Error("internal server error: cannot update postgres db")
			return nil, err
		}
	default:
		slog.Info("bad request: invalid currency")
		return nil, echo.ErrBadRequest
	}

	depositResponse := new(repository.DepositResponse)
	err := repo.db.QueryRow(context.Background(), "select balance_usd, balance_rub, balance_eur from wallets where username = $1", claims.Username).Scan(&depositResponse.NewBalance.USD, &depositResponse.NewBalance.RUB, &depositResponse.NewBalance.EUR)
	if err != nil {
		slog.Error("internal server error: cannot scan into repository.DepositResponse.Newbalance")
		return nil, err
	}
	depositResponse.Message = "Funds successfully added"
	return depositResponse, nil
}

func (repo *PostgresRepo) Withdraw(ctx echo.Context, request *repository.WithdrawRequest) (*repository.WithdrawResponse, error) {
	slog.Info("new request: received request for withdrawal")
	if request.Amount <= 0 {
		slog.Info("bad request: invalid amount")
		return nil, echo.ErrBadRequest
	}

	user := ctx.Get("user").(*jwt.Token)

	if !user.Valid {
		slog.Info("unauthorized: invalid token")
		return nil, echo.ErrUnauthorized
	}

	claims := user.Claims.(*types.JwtClaims)

	switch request.Currency {
	case "USD":
		_, err := repo.db.Exec(context.Background(), "update wallets set balance_usd = balance_usd - $1 where username = $2", request.Amount, claims.Username)
		if err != nil {
			slog.Error("internal server error: cannot update postgres db")
			return nil, err
		}
	case "RUB":
		_, err := repo.db.Exec(context.Background(), "update wallets set balance_rub = balance_rub - $1 where username = $2", request.Amount, claims.Username)
		if err != nil {
			slog.Error("internal server error: cannot update postgres db")
			return nil, err
		}
	case "EUR":
		_, err := repo.db.Exec(context.Background(), "update wallets set balance_eur = balance_eur - $1 where username = $2", request.Amount, claims.Username)
		if err != nil {
			slog.Error("internal server error: cannot update postgres db")
			return nil, err
		}
	default:
		slog.Info("bad request: invalid currency")
		return nil, echo.ErrBadRequest
	}

	withdrawResponse := new(repository.WithdrawResponse)
	err := repo.db.QueryRow(context.Background(), "select balance_usd, balance_rub, balance_eur from wallets where username = $1", claims.Username).Scan(&withdrawResponse.NewBalance.USD, &withdrawResponse.NewBalance.RUB, &withdrawResponse.NewBalance.EUR)
	if err != nil {
		slog.Error("internal server error: cannot scan into withdrawResponse.NewBalance")
		return nil, err
	}
	withdrawResponse.Message = "Withdrawal successful"
	return withdrawResponse, nil
}

func (repo *PostgresRepo) Exchange(ctx echo.Context, request *repository.ExchangeRequest) (*repository.ExchangeResponse, error) {

	if request.Amount <= 0 {
		slog.Info("bad request: invalid amount")
		return nil, echo.ErrBadRequest
	}

	if request.Rate <= 0 {
		slog.Warn("internal server error: invalid rate")
		return nil, echo.ErrInternalServerError
	}

	user := ctx.Get("user").(*jwt.Token)
	if !user.Valid {
		slog.Info("unauthorized: invalid token")
		return nil, echo.ErrUnauthorized
	}

	claims := user.Claims.(*types.JwtClaims)

	oldBalance := new(repository.Balance)
	err := repo.db.QueryRow(context.Background(), "select (balance_usd,balance_rub,balance_eur) from wallets where username = $1", claims.Username).Scan(oldBalance)
	if err != nil {
		slog.Error("internal server error: cannot scan into oldBalance - repository.Balance")
		return nil, echo.ErrInternalServerError
	}

	oldBalanceMap := map[string]float64{
		"USD": oldBalance.USD,
		"RUB": oldBalance.RUB,
		"EUR": oldBalance.EUR,
	}
	fmt.Printf("Old Balance: %v\n", oldBalance)
	if oldBalanceMap[request.FromCurrency] < request.Amount {
		slog.Info("bad request: balance is lower than requested amount")
		return nil, echo.ErrBadRequest
	}

	newBalanceMap := oldBalanceMap
	newBalanceMap[request.FromCurrency] = newBalanceMap[request.FromCurrency] - request.Amount
	newBalanceMap[request.ToCurrency] = newBalanceMap[request.ToCurrency] + request.Amount*request.Rate
	fmt.Printf("New Balance: %v\n", newBalanceMap)
	_, err = repo.db.Exec(context.Background(), "update wallets set balance_usd = $1, balance_rub = $2, balance_eur = $3 where username = $4", newBalanceMap["USD"], newBalanceMap["RUB"], newBalanceMap["EUR"], claims.Username)
	if err != nil {
		slog.Error("internal server error: cannot scan into newBalanceMap")
		return nil, echo.ErrInternalServerError
	}

	return &repository.ExchangeResponse{
		Message:         "Exchange successful",
		ExchangedAmount: request.Amount * request.Rate,
		NewBalance: map[string]float64{
			request.FromCurrency: newBalanceMap[request.FromCurrency],
			request.ToCurrency:   newBalanceMap[request.ToCurrency],
		},
	}, nil
}
