package httpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domain "github.com/antei22/tripgo/internal/domain/trip"
	"github.com/antei22/tripgo/internal/httpserver"
)

func TestHandlers(t *testing.T) {
	const tripsPath = "/api/v1/trips"
	const invalidRequest = "invalid_request"
	id := uuid.New()
	driver := uuid.New()
	user := uuid.New()
	valid := `{"user_id":"` + user.String() + `","driver_id":"` + driver.String() + `","start_point":{"latitude":1,"longitude":1},"end_point":{"latitude":2,"longitude":2},"price":10}`
	tests := []struct {
		name, method, path, body string
		status                   int
		code                     string
		setup                    func(*MockcreateService, *MockgetService, *MockfinishService)
	}{
		{"invalid", http.MethodPost, tripsPath, "{", 400, invalidRequest, func(*MockcreateService, *MockgetService, *MockfinishService) {}},
		{"missing fields", http.MethodPost, tripsPath, `{}`, 400, invalidRequest, func(*MockcreateService, *MockgetService, *MockfinishService) {}},
		{"unknown field", http.MethodPost, tripsPath, strings.Replace(valid, `"price":10`, `"price":10,"unknown":true`, 1), 400, invalidRequest, func(*MockcreateService, *MockgetService, *MockfinishService) {}},
		{"panic", http.MethodGet, "/api/v1/trips/" + id.String(), "", 500, "internal_error", func(c *MockcreateService, g *MockgetService, f *MockfinishService) {
			g.EXPECT().Get(gomock.Any(), id).DoAndReturn(func(context.Context, uuid.UUID) (domain.Trip, error) { panic("test panic") })
		}},
		{"create", http.MethodPost, tripsPath, valid, 201, "", func(c *MockcreateService, g *MockgetService, f *MockfinishService) {
			c.EXPECT().Create(gomock.Any(), gomock.Any()).Return(domain.Trip{ID: id, Status: domain.StatusActive}, nil)
		}},
		{"busy", http.MethodPost, tripsPath, valid, 409, "driver_busy", func(c *MockcreateService, g *MockgetService, f *MockfinishService) {
			c.EXPECT().Create(gomock.Any(), gomock.Any()).Return(domain.Trip{}, domain.ErrDriverBusy)
		}},
		{"missing", http.MethodGet, "/api/v1/trips/" + id.String(), "", 404, "trip_not_found", func(c *MockcreateService, g *MockgetService, f *MockfinishService) {
			g.EXPECT().Get(gomock.Any(), id).Return(domain.Trip{}, domain.ErrTripNotFound)
		}},
		{"finish", http.MethodPost, "/api/v1/trips/" + id.String() + "/finish", "", 200, "", func(c *MockcreateService, g *MockgetService, f *MockfinishService) {
			f.EXPECT().Finish(gomock.Any(), id).Return(domain.Trip{ID: id, Status: domain.StatusCompleted}, nil)
		}},
		{"completed", http.MethodPost, "/api/v1/trips/" + id.String() + "/finish", "", 409, "trip_completed", func(c *MockcreateService, g *MockgetService, f *MockfinishService) {
			f.EXPECT().Finish(gomock.Any(), id).Return(domain.Trip{}, domain.ErrTripCompleted)
		}},
		{"internal", http.MethodGet, "/api/v1/trips/" + id.String(), "", 500, "internal_error", func(c *MockcreateService, g *MockgetService, f *MockfinishService) {
			g.EXPECT().Get(gomock.Any(), id).Return(domain.Trip{}, errors.New("secret sql detail"))
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			creator := NewMockcreateService(ctrl)
			getter := NewMockgetService(ctrl)
			finisher := NewMockfinishService(ctrl)
			ping := NewMockpinger(ctrl)
			tt.setup(creator, getter, finisher)
			h := httpserver.NewRouter(httpserver.NewHandler(creator, getter, finisher, ping, time.Second, slog.New(slog.NewJSONHandler(io.Discard, nil))))
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != tt.status {
				t.Fatalf("status %d, body %s", rr.Code, rr.Body.String())
			}
			if strings.Contains(rr.Body.String(), "secret sql detail") {
				t.Fatal("internal detail leaked")
			}
			if tt.code != "" {
				var body struct {
					Code string `json:"code"`
				}
				if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil || body.Code != tt.code {
					t.Fatalf("body %s err %v", rr.Body, err)
				}
				if rr.Header().Get("Content-Type") != "application/problem+json" {
					t.Fatal(rr.Header())
				}
			}
		})
	}
}
