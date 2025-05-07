package postgres_test

import (
	"context"
	"fmt"
	"gw-exchanger/internal/repository/postgres"
	"log"
	"maps"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
)

var db *pgxpool.Pool
var connStr string

func TestMain(m *testing.M) {
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
			"POSTGRES_DB=walletrates",
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
	connStr = fmt.Sprintf("postgres://user:pass@%s/walletrates?sslmode=disable", hostAndPort)

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

func TestGetRates(t *testing.T) {
	repo, err := postgres.NewPostgresRepo(connStr)
	if err != nil {
		t.Fatalf("Couldn't connect to db.")
	}
	expectedRates := map[string]float64{
		"USD": 1,
		"RUB": 80.24,
		"EUR": 0.8851,
	}

	rates, err := repo.GetRates()
	if err != nil {
		t.Fatalf("Couldn't scan rates into a map.")
	}
	if !maps.Equal(rates, expectedRates) {
		t.Fatalf("expected %v, got %v\n", rates, expectedRates)
	}
}
func TestExchange(t *testing.T) {
	repo, err := postgres.NewPostgresRepo(connStr)
	if err != nil {
		t.Fatalf("Couldn't connect to db.")
	}

	var expectedRate float64 = 0.011030658025922234

	rate, err := repo.Exchange("RUB", "EUR")

	if rate != expectedRate {
		t.Fatalf("expected %v, got %v\n", rate, expectedRate)
	}

}
