// Copyright 2026, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"log"
	"sync"
)

type SSEEvent struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type SSEHub struct {
	mu      sync.RWMutex
	clients map[chan SSEEvent]bool
}

var globalHub = &SSEHub{
	clients: make(map[chan SSEEvent]bool),
}

func (h *SSEHub) subscribe() chan SSEEvent {
	ch := make(chan SSEEvent, 32)
	h.mu.Lock()
	h.clients[ch] = true
	h.mu.Unlock()
	return ch
}

func (h *SSEHub) unsubscribe(ch chan SSEEvent) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
	close(ch)
}

func (h *SSEHub) publish(evtType string, payload any) {
	evt := SSEEvent{Type: evtType, Payload: payload}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- evt:
		default:
			log.Printf("sse: client channel full, dropping event %s\n", evtType)
		}
	}
}

func publishEvent(evtType string, payload any) {
	globalHub.publish(evtType, payload)
}

func marshalSSEEvent(evt SSEEvent) ([]byte, error) {
	return json.Marshal(evt)
}
