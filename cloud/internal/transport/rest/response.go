// Package rest — response.go provides shared, type-safe helpers for REST
// responses: success/error envelopes, domain-error mapping, and list options.
package rest

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

type ListResponse[T any] struct {
	Items   []T `json:"items"`
	Total   int `json:"total"`
	Page    int `json:"page"`
	PerPage int `json:"perPage"`
}

func NewListResponse[T any](items []T, total int, opts domain.ListOptions) ListResponse[T] {
	if items == nil {
		items = []T{}
	}
	return ListResponse[T]{
		Items:   items,
		Total:   total,
		Page:    opts.Page,
		PerPage: opts.PerPage,
	}
}

func respondJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, ErrorResponse{Error: ErrorDetail{Code: code, Message: message}})
}

func respondDomainError(w http.ResponseWriter, err error) {
	var validationErr *domain.ValidationError
	switch {
	case errors.Is(err, domain.ErrNotFound):
		respondError(w, http.StatusNotFound, "not_found", "resource not found")
	case errors.As(err, &validationErr):
		respondJSON(w, http.StatusUnprocessableEntity, ErrorResponse{Error: ErrorDetail{
			Code:    "validation_failed",
			Message: validationErr.Message,
			Field:   validationErr.Field,
		}})
	default:
		slog.Error("unhandled internal error", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
	}
}

func respondBadRequest(w http.ResponseWriter, message string) {
	respondError(w, http.StatusBadRequest, "bad_request", message)
}

func parseListOptions(r *http.Request) domain.ListOptions {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(q.Get("perPage"))
	if perPage < 1 || perPage > 100 {
		perPage = 25
	}

	return domain.ListOptions{
		Page:    page,
		PerPage: perPage,
		OrderBy: q.Get("orderBy"),
		Search:  q.Get("search"),
	}
}
