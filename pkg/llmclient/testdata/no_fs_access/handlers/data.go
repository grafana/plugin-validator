package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
)

type DataRequest struct {
	Query  string   `json:"query"`
	Fields []string `json:"fields"`
	Limit  int      `json:"limit"`
}

type DataResponse struct {
	Results []map[string]interface{} `json:"results"`
	Total   int                      `json:"total"`
}

func DataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 10
	}

	query := strings.TrimSpace(req.Query)
	if query == "" {
		http.Error(w, "query is required", http.StatusBadRequest)
		return
	}

	resp := DataResponse{
		Results: []map[string]interface{}{},
		Total:   0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
