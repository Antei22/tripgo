package get_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domain "github.com/antei22/tripgo/internal/domain/trip"
	usecase "github.com/antei22/tripgo/internal/usecase/trip/get"
)

func TestGet(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	id := uuid.New()
	repo.EXPECT().Get(gomock.Any(), id).Return(domain.Trip{}, domain.ErrTripNotFound)
	_, err := usecase.NewService(repo).Get(context.Background(), id)
	if !errors.Is(err, domain.ErrTripNotFound) {
		t.Fatal(err)
	}
}
