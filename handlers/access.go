package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type AddAccessRequest struct {
	GroupBkeys []int `json:"groupBkeys"`
}

func (h *Handler) ListUserAccess(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	accessRepo := h.accessRepo
	h.mu.RUnlock()

	if accessRepo == nil {
		http.Error(w, "Database not connected", http.StatusServiceUnavailable)
		return
	}

	idStr := r.PathValue("id")
	userID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	accessList, err := accessRepo.ListByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accessList)
}

func (h *Handler) AddUserAccess(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	accessRepo := h.accessRepo
	h.mu.RUnlock()

	if accessRepo == nil {
		http.Error(w, "Database not connected", http.StatusServiceUnavailable)
		return
	}

	idStr := r.PathValue("id")
	userID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var req AddAccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.GroupBkeys) == 0 {
		http.Error(w, "At least one group is required", http.StatusBadRequest)
		return
	}

	// Filter out groups the user already has access to
	var newGroupBkeys []int
	for _, groupBkey := range req.GroupBkeys {
		exists, err := accessRepo.Exists(r.Context(), userID, groupBkey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !exists {
			newGroupBkeys = append(newGroupBkeys, groupBkey)
		}
	}

	if len(newGroupBkeys) > 0 {
		if err := accessRepo.AddGroups(r.Context(), userID, newGroupBkeys); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RemoveAccess(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	accessRepo := h.accessRepo
	h.mu.RUnlock()

	if accessRepo == nil {
		http.Error(w, "Database not connected", http.StatusServiceUnavailable)
		return
	}

	idStr := r.PathValue("id")
	accessID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid access ID", http.StatusBadRequest)
		return
	}

	if err := accessRepo.Remove(r.Context(), accessID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
