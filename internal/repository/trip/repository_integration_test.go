//go:build integration

package trip_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/antei22/tripgo/internal/database"
	domain "github.com/antei22/tripgo/internal/domain/trip"
	repository "github.com/antei22/tripgo/internal/repository/trip"
	create "github.com/antei22/tripgo/internal/usecase/trip/create"
	finish "github.com/antei22/tripgo/internal/usecase/trip/finish"
)

func TestRepositoryAndTransactions(t *testing.T) {
	ctx := context.Background()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Fatal("DATABASE_URL is required")
	}
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	repo := repository.NewRepository(pool, 5*time.Second)
	tx := database.NewManager(pool, 5*time.Second)
	driver := uuid.New()
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, "DELETE FROM trips WHERE driver_id=$1", driver); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	})
	tr := domain.Trip{ID: uuid.New(), UserID: uuid.New(), DriverID: driver, Status: domain.StatusActive, StartedAt: time.Now().UTC()}
	err = tx.Do(ctx, func(ctx context.Context) error {
		if err := repo.Create(ctx, tr); err != nil {
			return err
		}
		return repo.AddStatusHistory(ctx, tr.ID, nil, domain.Status("invalid"), nil)
	})
	if err == nil {
		t.Fatal("expected status history insert to fail")
	}
	if _, err = repo.Get(ctx, tr.ID); !errors.Is(err, domain.ErrTripNotFound) {
		t.Fatalf("trip survived rollback: %v", err)
	}
	creator := create.NewService(repo, tx)
	finisher := finish.NewService(repo, tx)
	got, err := creator.Create(ctx, domain.CreateInput{UserID: tr.UserID, DriverID: driver})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := creator.Create(ctx, domain.CreateInput{UserID: uuid.New(), DriverID: driver}); !errors.Is(err, domain.ErrDriverBusy) {
		t.Fatalf("duplicate: %v", err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := finisher.Finish(ctx, got.ID); results <- err }()
	}
	wg.Wait()
	close(results)
	success, conflict := 0, 0
	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, domain.ErrTripCompleted):
			conflict++
		default:
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("success=%d conflict=%d", success, conflict)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM trip_status_history WHERE trip_id=$1", got.ID).Scan(&count); err != nil || count != 2 {
		t.Fatalf("history=%d err=%v", count, err)
	}
}
