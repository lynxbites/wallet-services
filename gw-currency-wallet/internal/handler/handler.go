package handler

import (
	"errors"
	"fmt"
	"gw-wallet/internal/repository"
	"gw-wallet/internal/service"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	service *service.Service
}

func NewHandler(service *service.Service) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) Register(ctx echo.Context) error {
	slog.Info("new request: received register request")

	registerRequest := new(repository.RegisterRequest)

	if err := ctx.Bind(registerRequest); err != nil {
		slog.Info("bad request: error when binding to repository.RegisterRequest: " + fmt.Sprint(registerRequest))
		return echo.ErrBadRequest
	}

	if registerRequest.Username == "" || registerRequest.Password == "" || registerRequest.Email == "" {
		slog.Info("bad request: empty field in request: " + fmt.Sprint(registerRequest))
		return echo.ErrBadRequest
	}

	if err := handler.service.RegisterUser(ctx, registerRequest); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			fmt.Println(pgErr.Message)
			fmt.Println(pgErr.Code)
			if pgErr.Code == "23505" { //23514
				slog.Info("bad request: username or email already exists")
				return ctx.JSON(echo.ErrBadRequest.Code, map[string]string{"error": "Username or email already exists"})
			}

		}
		slog.Warn("internal server error")
		return err
	}

	slog.Info("ok: user registered successfully")
	return ctx.JSON(http.StatusCreated, echo.Map{
		"message": "User registered successfully",
	})
}

func (handler *Handler) Login(ctx echo.Context) error {
	slog.Info("new request: received login request")

	loginRequest := new(repository.LoginRequest)
	if err := ctx.Bind(loginRequest); err != nil {
		return echo.ErrBadRequest
	}

	token, err := handler.service.LoginUser(ctx, loginRequest)
	if err != nil {
		if errors.Is(err, echo.ErrUnauthorized) {

			return ctx.JSON(http.StatusUnauthorized, echo.Map{
				"error": "Invalid username or password",
			})
		} else {
			return err
		}
	}
	slog.Info("ok: user logged in successfully")
	return ctx.JSON(http.StatusOK, echo.Map{
		"token": token,
	})
}

func (handler *Handler) GetBalance(ctx echo.Context) error {
	slog.Info("new request: received balance request")
	balance, err := handler.service.GetBalance(ctx)
	if err != nil {
		return err
	}
	slog.Info("ok: balance request fulfilled")
	return ctx.JSON(http.StatusOK, balance)
}

func (handler *Handler) Deposit(ctx echo.Context) error {
	slog.Info("new request: received deposit request")
	depositRequest := new(repository.DepositRequest)
	if err := ctx.Bind(depositRequest); err != nil {
		slog.Info("bad request: invalid request body")
		return echo.ErrBadRequest
	}

	response, err := handler.service.Deposit(ctx, depositRequest)
	if err != nil {
		return err
	}
	slog.Info("ok: deposit request fulfilled")
	return ctx.JSON(http.StatusOK, response)
}

func (handler *Handler) Withdraw(ctx echo.Context) error {
	slog.Info("new request: received withdraw request")
	withdrawRequest := new(repository.WithdrawRequest)
	if err := ctx.Bind(withdrawRequest); err != nil {
		slog.Info("bad request: invalid request body")
		return echo.ErrBadRequest
	}

	response, err := handler.service.Withdraw(ctx, withdrawRequest)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			fmt.Println(pgErr.Message)
			fmt.Println(pgErr.Code)
			if pgErr.Code == "23514" {
				slog.Error("Insufficient balance for withdrawal.")
				return ctx.JSON(echo.ErrBadRequest.Code, map[string]string{"error": "Insufficient balance for withdrawal"})
			}
		}
		return err
	}
	slog.Info("ok: withdraw request fulfilled")
	return ctx.JSON(http.StatusOK, response)
}

func (handler *Handler) GetExchangeRates(ctx echo.Context) error {
	slog.Info("new request: received request for exchange rates")
	response, err := handler.service.GetExchangeRates(ctx)
	if err != nil {
		return err
	}
	slog.Info("ok: get exchange rates request fulfilled")
	return ctx.JSON(http.StatusOK, response.Rates)
}

func (handler *Handler) Exchange(ctx echo.Context) error {
	slog.Info("new request: received request for exchange")
	exchangeRequest := new(repository.ExchangeRequestClient)
	if err := ctx.Bind(exchangeRequest); err != nil {
		return echo.ErrBadRequest
	}
	response, err := handler.service.Exchange(ctx, exchangeRequest)
	if err != nil {
		return err
	}
	slog.Info("ok: exchange request fulfilled")
	return ctx.JSON(http.StatusOK, response)
}
