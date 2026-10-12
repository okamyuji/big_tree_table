package handler

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"bigtable-backend/model"
)

type failingService struct{ err error }

func (f failingService) GetOrders(model.QueryParams) (model.OrdersResponse, error) {
	return model.OrdersResponse{}, f.err
}

func (f failingService) GetOrderTree(model.QueryParams) (model.OrderTreeResponse, error) {
	return model.OrderTreeResponse{}, f.err
}

func TestHandlers_LogServiceErrorAndHideItFromClient(t *testing.T) {
	cause := errors.New("dial tcp 10.0.0.1:3306: connection refused")
	tests := []struct {
		name    string
		handler http.HandlerFunc
		path    string
	}{
		{"orders", OrdersHandler(failingService{cause}), "/api/orders"},
		{"order-tree", OrderTreeHandler(failingService{cause}), "/api/order-tree"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			log.SetOutput(&logs)
			t.Cleanup(func() { log.SetOutput(os.Stderr) })

			rec := httptest.NewRecorder()
			tt.handler(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500", rec.Code)
			}
			if strings.Contains(rec.Body.String(), cause.Error()) {
				t.Errorf("response body leaks the cause: %q", rec.Body.String())
			}
			if !strings.Contains(logs.String(), cause.Error()) {
				t.Errorf("log = %q, want it to contain %q", logs.String(), cause.Error())
			}
		})
	}
}
