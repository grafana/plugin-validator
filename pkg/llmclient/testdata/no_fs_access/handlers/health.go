package handlers

import (
	"encoding/json"
	"net/http"
	"runtime"
)

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	GoVer   string `json:"go_version"`
}

func HealthHandler(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := HealthResponse{
			Status:  "ok",
			Version: version,
			GoVer:   runtime.Version(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
