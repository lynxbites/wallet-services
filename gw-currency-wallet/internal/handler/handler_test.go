package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"gw-wallet/internal/config"
	"gw-wallet/internal/handler"
	"gw-wallet/internal/repository"
	"gw-wallet/internal/repository/postgres"
	"gw-wallet/internal/service"
	"gw-wallet/internal/types"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

var db *pgxpool.Pool
var cfg *config.Config
var connStr string

func TestMain(m *testing.M) {
	var err error
	cfg, err = config.NewConfig()
	if err != nil {
		log.Fatalf("Could not setup config: %s", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Could not get working directory: %s", err)
	}
	fmt.Printf("wd: %v\n", wd)

	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "alpine",
		Env: []string{
			"POSTGRES_PASSWORD=pass",
			"POSTGRES_USER=user",
			"POSTGRES_DB=walletservice",
			"listen_addresses = '*'",
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	hostAndPort := resource.GetHostPort("5432/tcp")
	connStr = fmt.Sprintf("postgres://user:pass@%s/walletservice?sslmode=disable", hostAndPort)

	log.Println("Connecting to database on url: ", connStr)

	resource.Expire(60)

	pool.MaxWait = 20 * time.Second
	if err = pool.Retry(func() error {
		db, err = pgxpool.New(context.Background(), connStr)
		if err != nil {
			return err
		}
		return db.Ping(context.Background())
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	migration, err := migrate.New("file://../migrations/", connStr)
	if err != nil {
		log.Fatal(err)
	}
	err = migration.Up()
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		if err := pool.Purge(resource); err != nil {
			log.Fatalf("Could not purge resource: %s", err)
		}
	}()

	// run tests
	m.Run()
}

type Token struct {
	Token string
}

var token *Token

func TestStuff(t *testing.T) {
	testRepo, err := postgres.NewPostgresRepo(connStr)
	if err != nil {
		t.Fatal(err)
	}
	testService := service.NewService(testRepo, cfg)
	testHandler := handler.NewHandler(testService)

	e := echo.New()
	e.Use()

	// register test
	registerRequestBody := repository.RegisterRequest{
		Username: "newuser",
		Password: "1",
		Email:    "newuser@mail",
	}
	jsonBody, err := json.Marshal(registerRequestBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	context := e.NewContext(req, rec)
	err = testHandler.Register(context)
	if err != nil {
		t.Fatal(err)
	}

	//login test
	loginRequestBody := repository.LoginRequest{
		Username: "user",
		Password: "1",
	}
	jsonBody, err = json.Marshal(loginRequestBody)
	if err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	context = e.NewContext(req, rec)
	err = testHandler.Login(context)
	if err != nil {
		t.Fatal(err)
	}
	token = new(Token)
	err = json.Unmarshal(rec.Body.Bytes(), &token)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("token: %v\n", token.Token)

}

func TestGetBalance(t *testing.T) {

	///////////////////////////////boilerplate
	testRepo, err := postgres.NewPostgresRepo(connStr)
	if err != nil {
		t.Fatal(err)
	}
	testService := service.NewService(testRepo, cfg)
	testHandler := handler.NewHandler(testService)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/balance", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token.Token)
	rec := httptest.NewRecorder()
	context := e.NewContext(req, rec)
	context.Set("user", &jwt.Token{
		Claims: &types.JwtClaims{Username: "user"},
		Valid:  true,
	})
	///////////////////////////////

	err = testHandler.GetBalance(context)
	if err != nil {
		t.Fatal(err)
	}
	balanceResponse := repository.BalanceResponse{}
	err = json.Unmarshal(rec.Body.Bytes(), &balanceResponse)
	if err != nil {
		t.Fatal(err)
	}

	expectedBalance := repository.Balance{
		USD: 100,
		RUB: 100,
		EUR: 100,
	}
	if balanceResponse.Balance != expectedBalance {
		t.Errorf("expected %v, got %v\n", expectedBalance, balanceResponse.Balance)
	}

}

func TestDeposit(t *testing.T) {
	///////////////////////////////boilerplate
	testRepo, err := postgres.NewPostgresRepo(connStr)
	if err != nil {
		t.Fatal(err)
	}
	testService := service.NewService(testRepo,cfg)
	testHandler := handler.NewHandler(testService)

	e := echo.New()
	registerRequestBody := repository.DepositRequest{
		Amount:   50,
		Currency: "USD",
	}
	jsonBody, err := json.Marshal(registerRequestBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet/deposit", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token.Token)
	rec := httptest.NewRecorder()
	context := e.NewContext(req, rec)
	context.Set("user", &jwt.Token{
		Claims: &types.JwtClaims{Username: "user"},
		Valid:  true,
	})
	///////////////////////////////

	err = testHandler.Deposit(context)
	if err != nil {
		t.Fatal(err)
	}
	depositResponse := repository.DepositResponse{}
	err = json.Unmarshal(rec.Body.Bytes(), &depositResponse)
	if err != nil {
		t.Fatal(err)
	}
	expectedBalance := repository.Balance{
		USD: 150,
		RUB: 100,
		EUR: 100,
	}
	if depositResponse.NewBalance != expectedBalance {
		t.Errorf("expected %v, got %v\n", expectedBalance, depositResponse.NewBalance)
	}
}

func TestWithdraw(t *testing.T) {
	///////////////////////////////boilerplate
	testRepo, err := postgres.NewPostgresRepo(connStr)
	if err != nil {
		t.Fatal(err)
	}
	testService := service.NewService(testRepo,cfg)
	testHandler := handler.NewHandler(testService)

	e := echo.New()
	requestBody := repository.WithdrawRequest{
		Amount:   100,
		Currency: "USD",
	}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet/withdraw", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token.Token)
	rec := httptest.NewRecorder()
	context := e.NewContext(req, rec)
	context.Set("user", &jwt.Token{
		Claims: &types.JwtClaims{Username: "user"},
		Valid:  true,
	})
	///////////////////////////////

	err = testHandler.Withdraw(context)
	if err != nil {
		t.Fatal(err)
	}
	withdrawResponse := repository.WithdrawResponse{}
	err = json.Unmarshal(rec.Body.Bytes(), &withdrawResponse)
	if err != nil {
		t.Fatal(err)
	}
	expectedBalance := repository.Balance{
		USD: 50,
		RUB: 100,
		EUR: 100,
	}
	fmt.Printf("withdrawResponse: %v\n", withdrawResponse)
	if withdrawResponse.NewBalance != expectedBalance {
		t.Errorf("expected %v, got %v\n", expectedBalance, withdrawResponse.NewBalance)
	}
}
