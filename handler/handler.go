package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// Handler for http requests
type Handler struct{}

type ClearRequest struct {
	IP string
}

// NewHandler http handler
func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Run(w http.ResponseWriter, r *http.Request) {
	ips := r.Header.Get("X-FORWARDED-FOR")
	err := json.NewEncoder(w).Encode(map[string]string{"message": "Run response", "X-FORWARDED-FOR": ips})
	if err != nil {
		log.Fatal(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
}

func (h *Handler) Reset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Header().Set("Content-Type", "application/json")
		return
	}

	var clearReq ClearRequest
	if err := json.NewDecoder(r.Body).Decode(&clearReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if clearReq.IP == "" {
		http.Error(w, "Empty param IP", http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, "Clear request: %+v", clearReq)

	ips := r.Header.Get("X-FORWARDED-FOR")
	err := json.NewEncoder(w).Encode(map[string]string{"message": "Reset response", "X-FORWARDED-FOR": ips})
	if err != nil {
		log.Fatal(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
}
