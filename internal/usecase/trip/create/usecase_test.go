package create_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domain "github.com/antei22/tripgo/internal/domain/trip"
	usecase "github.com/antei22/tripgo/internal/usecase/trip/create"
)

func TestCreate(t *testing.T) {
	for _, tt := range []struct {
		name    string
		repoErr error
	}{{"success", nil}, {"driver busy", domain.ErrDriverBusy}} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMocktripRepository(ctrl)
			tx := NewMocktransactionManager(ctrl)
			tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
			repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(tt.repoErr)
			if tt.repoErr == nil {
				repo.EXPECT().AddStatusHistory(gomock.Any(), gomock.Any(), nil, domain.StatusActive, nil).Return(nil)
			}
			input := domain.CreateInput{UserID: uuid.New(), DriverID: uuid.New()}
			got, err := usecase.NewService(repo, tx).Create(context.Background(), input)
			if !errors.Is(err, tt.repoErr) {
				t.Fatalf("error = %v", err)
			}
			if err == nil && (got.ID == uuid.Nil || got.Status != domain.StatusActive || got.DriverID != input.DriverID) {
				t.Fatalf("trip = %+v", got)
			}
		})
	}
}
