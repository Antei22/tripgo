package finish_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domain "github.com/antei22/tripgo/internal/domain/trip"
	usecase "github.com/antei22/tripgo/internal/usecase/trip/finish"
)

func TestFinish(t *testing.T) {
	for _, tt := range []struct {
		name    string
		status  domain.Status
		wantErr error
	}{{"success", domain.StatusActive, nil}, {"completed", domain.StatusCompleted, domain.ErrTripCompleted}} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMocktripRepository(ctrl)
			tx := NewMocktransactionManager(ctrl)
			id := uuid.New()
			tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
			repo.EXPECT().GetForUpdate(gomock.Any(), id).Return(domain.Trip{ID: id, Status: tt.status}, nil)
			if tt.wantErr == nil {
				repo.EXPECT().SaveFinished(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, tr domain.Trip) error {
					if tr.Status != domain.StatusCompleted || tr.FinishedAt == nil || tr.FinishedAt.After(time.Now().Add(time.Second)) {
						t.Fatalf("trip = %+v", tr)
					}
					return nil
				})
				repo.EXPECT().AddStatusHistory(gomock.Any(), id, gomock.Any(), domain.StatusCompleted, nil).Return(nil)
			}
			got, err := usecase.NewService(repo, tx).Finish(context.Background(), id)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v", err)
			}
			if err == nil && got.Status != domain.StatusCompleted {
				t.Fatalf("trip = %+v", got)
			}
		})
	}
}
