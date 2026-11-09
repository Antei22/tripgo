package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type DBTX interface {
	Exec(
		ctx context.Context,
		sql string,
		arguments ...any,
	) (pgconn.CommandTag, error)

	Query(
		ctx context.Context,
		sql string,
		args ...any,
	) (pgx.Rows, error)

	QueryRow(
		ctx context.Context,
		sql string,
		args ...any,
	) pgx.Row
}

type Manager struct {
	pool            *pgxpool.Pool
	rollbackTimeout time.Duration
}

type transactionKey struct{}

func NewManager(
	pool *pgxpool.Pool,
	rollbackTimeout time.Duration,
) *Manager {
	return &Manager{
		pool:            pool,
		rollbackTimeout: rollbackTimeout,
	}
}

func Executor(ctx context.Context, pool *pgxpool.Pool) DBTX {
	if tx, ok := ctx.Value(transactionKey{}).(pgx.Tx); ok {
		return tx
	}

	return pool
}

func (m *Manager) Do(
	ctx context.Context,
	fn func(ctx context.Context) error,
) (resultErr error) {
	if _, ok := ctx.Value(transactionKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}

	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted,
	})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	txCtx := context.WithValue(ctx, transactionKey{}, tx)

	defer func() {
		if recovered := recover(); recovered != nil {
			_ = m.rollback(ctx, tx)
			panic(recovered)
		}
	}()

	if err := fn(txCtx); err != nil {
		rollbackErr := m.rollback(ctx, tx)
		if rollbackErr != nil {
			return errors.Join(
				err,
				fmt.Errorf("rollback transaction: %w", rollbackErr),
			)
		}

		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (m *Manager) rollback(
	ctx context.Context,
	tx pgx.Tx,
) error {
	rollbackCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		m.rollbackTimeout,
	)
	defer cancel()

	err := tx.Rollback(rollbackCtx)
	if errors.Is(err, pgx.ErrTxClosed) {
		return nil
	}

	return err
}
