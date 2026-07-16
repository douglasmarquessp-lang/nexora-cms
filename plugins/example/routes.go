package example

import (
	"context"
	"log/slog"
	"net/http"
)

func HelloHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(`{"message": "Hello from Example Plugin!"}`)); err != nil {
		slog.Error("failed to write response", "error", err)
	}
}
