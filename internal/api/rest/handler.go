package rest

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type Context struct {
	http.ResponseWriter
	*http.Request
}

func (c *Context) JSON(status int, data interface{}) {
	c.ResponseWriter.Header().Set("Content-Type", "application/json")
	c.ResponseWriter.WriteHeader(status)
	if err := json.NewEncoder(c.ResponseWriter).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

func (c *Context) Error(status int, code, message string, details ...interface{}) {
	body := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}

	if len(details) > 0 {
		if errMap, ok := body["error"].(map[string]interface{}); ok {
			errMap["details"] = details[0]
		}
	}

	c.JSON(status, body)
}

func (c *Context) Decode(v interface{}) error {
	return json.NewDecoder(c.Body).Decode(v)
}

func AdaptHandler(fn func(*Context)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := &Context{
			ResponseWriter: w,
			Request:        r,
		}
		fn(ctx)
	}
}
