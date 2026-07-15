package dto

type APIResponse struct {
	Data  interface{} `json:"data,omitempty"`
	Meta  *Meta       `json:"meta,omitempty"`
	Error *APIError   `json:"error,omitempty"`
}

type Meta struct {
	Page       int `json:"page,omitempty"`
	PerPage    int `json:"per_page,omitempty"`
	Total      int `json:"total,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

type APIError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

type PaginationParams struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
	Database  string `json:"database"`
}
