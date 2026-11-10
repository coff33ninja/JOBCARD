// Package api exposes stock inventory operations over HTTP.
package api

import (
	"net/http"

	"jobcard/internal/core"
)

func (s *Server) listStockItems(w http.ResponseWriter, _ *http.Request) {
	items, err := s.Svc.ListStockItems()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) createStockIntake(w http.ResponseWriter, r *http.Request) {
	var in core.StockIntakeInput
	if err := decode(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	items, err := s.Svc.CreateStockIntake(in)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, items)
}

func (s *Server) assignStock(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ClientID string `json:"clientId"`
		Quantity int    `json:"quantity"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	result, err := s.Svc.AllocateStock(r.PathValue("id"), in.ClientID, in.Quantity)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}
