package repository

import "github.com/labstack/echo/v4"

type WalletRepo interface {
	RegisterUser(ctx echo.Context, request *RegisterRequest) error
	LoginUser(ctx echo.Context, request *LoginRequest) (string, error)
	GetBalance(ctx echo.Context) (*BalanceResponse, error)
	Deposit(ctx echo.Context, request *DepositRequest) (*DepositResponse, error)
	Withdraw(ctx echo.Context, request *WithdrawRequest) (*WithdrawResponse, error)
	Exchange(ctx echo.Context, request *ExchangeRequest) (*ExchangeResponse, error)
}

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type BalanceRequest struct {
	Token string
}

type BalanceResponse struct {
	Balance struct {
		USD float64 `json:"USD"`
		RUB float64 `json:"RUB"`
		EUR float64 `json:"EUR"`
	} `json:"balance"`
}

type DepositRequest struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type DepositResponse struct {
	Message    string `json:"message"`
	NewBalance struct {
		USD float64 `json:"USD"`
		RUB float64 `json:"RUB"`
		EUR float64 `json:"EUR"`
	} `json:"new_balance"`
}

type WithdrawRequest struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type WithdrawResponse struct {
	Message    string `json:"message"`
	NewBalance struct {
		USD float64 `json:"USD"`
		RUB float64 `json:"RUB"`
		EUR float64 `json:"EUR"`
	} `json:"new_balance"`
}

type ExchangeRequest struct {
	FromCurrency string  `json:"from_currency"`
	ToCurrency   string  `json:"to_currency"`
	Amount       float64 `json:"amount"`
	Rate         float64 `json:"rate"`
}
type ExchangeRequestClient struct {
	FromCurrency string  `json:"from_currency"`
	ToCurrency   string  `json:"to_currency"`
	Amount       float64 `json:"amount"`
}

type ExchangeResponse struct {
	Message         string             `json:"message"`
	ExchangedAmount float64            `json:"exchanged_amount"`
	NewBalance      map[string]float64 `json:"new_balance"`
}

type Balance struct {
	USD float64 `json:"USD"`
	RUB float64 `json:"RUB"`
	EUR float64 `json:"EUR"`
}
