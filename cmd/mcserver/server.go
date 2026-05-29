// Copyright 2026, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/wavetermdev/waveterm/pkg/mcstore"
)

func makeRouter(authKey string) *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/healthz", handleHealthz).Methods("GET")

	api := r.PathPrefix("/api").Subrouter()
	api.Use(makeAuthMiddleware(authKey))

	api.HandleFunc("/projects", handleGetProjects).Methods("GET")
	api.HandleFunc("/projects", handleCreateProject).Methods("POST")
	api.HandleFunc("/projects/{id}", handleGetProject).Methods("GET")
	api.HandleFunc("/projects/{id}", handleUpdateProject).Methods("PATCH")

	api.HandleFunc("/tasks", handleGetTasks).Methods("GET")
	api.HandleFunc("/tasks", handleCreateTask).Methods("POST")
	api.HandleFunc("/tasks/{id}", handleGetTask).Methods("GET")
	api.HandleFunc("/tasks/{id}", handleUpdateTask).Methods("PATCH")

	api.HandleFunc("/sessions", handleGetSessions).Methods("GET")
	api.HandleFunc("/sessions", handleCreateSession).Methods("POST")
	api.HandleFunc("/sessions/{id}", handleUpdateSession).Methods("PATCH")
	api.HandleFunc("/sessions/{id}/heartbeat", handleSessionHeartbeat).Methods("POST")
	api.HandleFunc("/sessions/{id}/complete", handleSessionComplete).Methods("POST")

	api.HandleFunc("/intents", handleGetIntents).Methods("GET")
	api.HandleFunc("/intents", handleCreateIntent).Methods("POST")
	api.HandleFunc("/intents/{id}", handleUpdateIntent).Methods("PATCH")
	api.HandleFunc("/intents/{id}/claim", handleClaimIntent).Methods("POST")
	api.HandleFunc("/intents/claim-next", handleClaimNextIntent).Methods("POST")

	api.HandleFunc("/activities", handleGetActivities).Methods("GET")

	api.HandleFunc("/events", handleSSEEvents).Methods("GET")

	return r
}

func makeAuthMiddleware(authKey string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("Authorization")
			// SSE clients (EventSource) can't set custom headers — allow auth via query param
			if token == "" {
				if q := r.URL.Query().Get("auth"); q != "" {
					token = "Bearer " + q
				}
			}
			if token != "Bearer "+authKey {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// -- helpers --

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func readBody(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}

func queryParam(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}

func queryParamInt(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}

// -- handlers --

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleGetProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := mcstore.GetAllProjects(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": projects})
}

func handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var p mcstore.Project
	if err := readBody(r, &p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if p.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if err := mcstore.InsertProject(r.Context(), &p); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	publishEvent("project.created", p)
	writeJSON(w, http.StatusCreated, p)
}

func handleGetProject(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	p, err := mcstore.GetProject(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func handleUpdateProject(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var fields map[string]any
	if err := readBody(r, &fields); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := mcstore.UpdateProject(r.Context(), id, fields); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	p, _ := mcstore.GetProject(r.Context(), id)
	publishEvent("project.updated", p)
	writeJSON(w, http.StatusOK, p)
}

func handleGetTasks(w http.ResponseWriter, r *http.Request) {
	projectId := queryParam(r, "project_id")
	status := queryParam(r, "status")
	tasks, err := mcstore.GetTasksByProject(r.Context(), projectId, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": tasks})
}

func handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var t mcstore.Task
	if err := readBody(r, &t); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if t.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if t.ProjectId == "" {
		writeError(w, http.StatusBadRequest, "projectid is required")
		return
	}
	if t.Status == "" {
		t.Status = mcstore.TaskStatusTodo
	}
	if t.Priority == "" {
		t.Priority = mcstore.TaskPriorityMedium
	}
	if t.DependsOn == "" {
		t.DependsOn = "[]"
	}
	if err := mcstore.InsertTask(r.Context(), &t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	publishEvent("task.created", t)
	writeJSON(w, http.StatusCreated, t)
}

func handleGetTask(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	t, err := mcstore.GetTask(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var fields map[string]any
	if err := readBody(r, &fields); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := mcstore.UpdateTask(r.Context(), id, fields); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	t, _ := mcstore.GetTask(r.Context(), id)
	publishEvent("task.updated", t)
	writeJSON(w, http.StatusOK, t)
}

func handleGetSessions(w http.ResponseWriter, r *http.Request) {
	projectId := queryParam(r, "project_id")
	status := queryParam(r, "status")
	sessions, err := mcstore.GetSessionsByProject(r.Context(), projectId, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

func handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var s mcstore.AgentSession
	if err := readBody(r, &s); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if s.ProjectId == "" {
		writeError(w, http.StatusBadRequest, "projectid is required")
		return
	}
	if err := mcstore.InsertSession(r.Context(), &s); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	publishEvent("session.created", s)
	writeJSON(w, http.StatusCreated, s)
}

func handleUpdateSession(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var fields map[string]any
	if err := readBody(r, &fields); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := mcstore.UpdateSession(r.Context(), id, fields); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	publishEvent("session.updated", map[string]string{"id": id})
	writeJSON(w, http.StatusOK, map[string]string{"id": id})
}

func handleSessionHeartbeat(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := mcstore.UpdateSessionHeartbeat(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"id": id})
}

func handleGetIntents(w http.ResponseWriter, r *http.Request) {
	projectId := queryParam(r, "project_id")
	status := queryParam(r, "status")
	intents, err := mcstore.GetIntents(r.Context(), projectId, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"intents": intents})
}

func handleCreateIntent(w http.ResponseWriter, r *http.Request) {
	var intent mcstore.Intent
	if err := readBody(r, &intent); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if intent.Type == "" {
		writeError(w, http.StatusBadRequest, "type is required")
		return
	}
	if intent.ProjectId == "" {
		writeError(w, http.StatusBadRequest, "projectid is required")
		return
	}
	if intent.Payload == "" {
		intent.Payload = "{}"
	}
	if intent.Status == "" {
		intent.Status = mcstore.IntentStatusPending
	}
	if err := mcstore.InsertIntent(r.Context(), &intent); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	publishEvent("intent.created", intent)
	writeJSON(w, http.StatusCreated, intent)
}

func handleUpdateIntent(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var fields map[string]any
	if err := readBody(r, &fields); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := mcstore.UpdateIntent(r.Context(), id, fields); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	publishEvent("intent.updated", map[string]string{"id": id})
	writeJSON(w, http.StatusOK, map[string]string{"id": id})
}

func handleClaimIntent(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var body struct {
		ClaimedBy string `json:"claimedby"`
	}
	if err := readBody(r, &body); err != nil || body.ClaimedBy == "" {
		writeError(w, http.StatusBadRequest, "claimedby is required")
		return
	}
	ok, err := mcstore.ClaimIntent(r.Context(), id, body.ClaimedBy)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		writeError(w, http.StatusConflict, "intent already claimed or not found")
		return
	}
	publishEvent("intent.updated", map[string]string{"id": id, "status": "claimed"})
	writeJSON(w, http.StatusOK, map[string]string{"id": id, "status": "claimed"})
}

func handleClaimNextIntent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ClaimedBy string `json:"claimedby"`
	}
	if err := readBody(r, &body); err != nil || body.ClaimedBy == "" {
		writeError(w, http.StatusBadRequest, "claimedby is required")
		return
	}
	intent, err := mcstore.ClaimNextPendingIntent(r.Context(), body.ClaimedBy)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if intent == nil {
		writeJSON(w, http.StatusNoContent, nil)
		return
	}
	publishEvent("intent.updated", intent)
	writeJSON(w, http.StatusOK, intent)
}

func handleGetActivities(w http.ResponseWriter, r *http.Request) {
	projectId := queryParam(r, "project_id")
	taskId := queryParam(r, "task_id")
	limit := queryParamInt(r, "limit", 100)
	activities, err := mcstore.GetActivities(r.Context(), projectId, taskId, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"activities": activities})
}

func handleSSEEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	ch := globalHub.subscribe()
	defer globalHub.unsubscribe(ch)

	// Send initial ping
	fmt.Fprintf(w, "data: {\"type\":\"connected\"}\n\n")
	flusher.Flush()

	ctx := r.Context()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case evt, ok := <-ch:
			if !ok {
				return
			}
			data, err := marshalSSEEvent(evt)
			if err != nil {
				log.Printf("sse: marshal error: %v\n", err)
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// startStalenessWatcher marks sessions stale every 30s
func startStalenessWatcher() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			n, err := mcstore.MarkStaleSessionsOlderThan(ctx, 60)
			cancel()
			if err != nil {
				log.Printf("staleness watcher error: %v\n", err)
				continue
			}
			if n > 0 {
				log.Printf("marked %d sessions stale\n", n)
				publishEvent("session.stale", map[string]any{"count": n})
			}
		}
	}()
}
