package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	manifest "github.com/Muxcore-Media/ai-tickets"
	"github.com/Muxcore-Media/core/pkg/contracts"
	modulesdk "github.com/Muxcore-Media/core/sdk/go/module"
	"google.golang.org/grpc"
)

type Module struct {
	id, grpcAddr, httpAddr string
	mu                     sync.RWMutex
	seq                    int
	tickets                map[string]Ticket
	grpcSrv                *grpc.Server
	lis                    net.Listener
	httpSrv                *http.Server
}

func NewModule() *Module { return New(Config{}) }

type Config struct{ ID, GRPCAddr, HTTPAddr string }

func New(cfg Config) *Module {
	if cfg.ID == "" {
		cfg.ID = "ai-tickets"
	}
	if cfg.GRPCAddr == "" {
		cfg.GRPCAddr = "127.0.0.1:9766"
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = "127.0.0.1:9767"
	}
	if v := os.Getenv("AI_TICKETS_GRPC_ADDR"); v != "" {
		cfg.GRPCAddr = v
	}
	if v := os.Getenv("MUXCORE_HTTP_ADDR"); v != "" {
		cfg.HTTPAddr = v
	}
	return &Module{id: cfg.ID, grpcAddr: cfg.GRPCAddr, httpAddr: cfg.HTTPAddr, tickets: map[string]Ticket{}}
}

func (m *Module) Info() contracts.ModuleInfo {
	return contracts.ModuleInfo{
		ID: m.id, Name: "AI Tickets", Version: modulesdk.ManifestVersion(manifest.ManifestJSON),
		Roles: []string{"ai"}, Description: "Automatic household tickets that an AI can classify and resolve",
		Author: "Muxcore-Media", Capabilities: []string{"ai.tickets", "settings"},
		MinCoreVersion: MinCoreVersion, HTTPAddr: m.grpcAddr,
	}
}

func (m *Module) Init(context.Context) error { return nil }
func (m *Module) Start(ctx context.Context) error {
	return startHTTPGRPC(ctx, m.id, &m.grpcAddr, &m.httpAddr, &m.lis, &m.grpcSrv, &m.httpSrv, m, m.routes)
}
func (m *Module) Stop(ctx context.Context) error { return stopServers(ctx, m.grpcSrv, m.httpSrv) }
func (m *Module) Health(context.Context) error   { return nil }
func (m *Module) GRPCListenAddr() string         { return m.grpcAddr }
func (m *Module) HTTPListenAddr() string         { return m.httpAddr }
func (m *Module) Settings() []contracts.SettingDef {
	return []contracts.SettingDef{{Key: "auto_resolve", Label: "Auto-resolve on open", Type: contracts.SettingTypeBool, Value: "true", Default: "true", Group: "AI"}}
}
func (m *Module) UpdateSetting(key, value string) error {
	if key != "auto_resolve" {
		return fmt.Errorf("unknown setting %q", key)
	}
	return nil
}

func (m *Module) open(in Ticket) Ticket {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq++
	if in.ID == "" {
		in.ID = fmt.Sprintf("tkt-%d", m.seq)
	}
	in.Status = StatusOpen
	in.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	in.Intent, in.Language, in.Action = classifyTicket(in.Subject, in.Body)
	resolved, err := resolveTicket(in)
	if err != nil {
		in.Status = StatusFailed
		in.Error = err.Error()
		m.tickets[in.ID] = in
		return in
	}
	m.tickets[resolved.ID] = resolved
	return resolved
}

func (m *Module) routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/tickets", func(w http.ResponseWriter, r *http.Request) {
		var in Ticket
		if !readJSON(w, r, &in) {
			return
		}
		if stringsTrim(in.Subject) == "" && stringsTrim(in.Body) == "" {
			http.Error(w, `{"error":"subject or body required"}`, http.StatusBadRequest)
			return
		}
		writeJSON(w, m.open(in))
	})
	mux.HandleFunc("GET /v1/tickets", func(w http.ResponseWriter, _ *http.Request) {
		m.mu.RLock()
		defer m.mu.RUnlock()
		out := make([]Ticket, 0, len(m.tickets))
		for _, t := range m.tickets {
			out = append(out, t)
		}
		writeJSON(w, map[string]any{"tickets": out})
	})
	mux.HandleFunc("POST /v1/tickets/{id}/resolve", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		m.mu.Lock()
		t, ok := m.tickets[id]
		m.mu.Unlock()
		if !ok {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		got, err := resolveTicket(t)
		if err != nil {
			http.Error(w, jsonErr(err), http.StatusConflict)
			return
		}
		m.mu.Lock()
		m.tickets[id] = got
		m.mu.Unlock()
		writeJSON(w, got)
	})
}

func (m *Module) handleMesh(_ context.Context, method string, payload []byte) ([]byte, error) {
	switch method {
	case "OpenTicket":
		var in Ticket
		if err := json.Unmarshal(payload, &in); err != nil {
			return nil, err
		}
		return json.Marshal(m.open(in))
	case "ListTickets":
		m.mu.RLock()
		defer m.mu.RUnlock()
		out := make([]Ticket, 0, len(m.tickets))
		for _, t := range m.tickets {
			out = append(out, t)
		}
		return json.Marshal(map[string]any{"tickets": out})
	default:
		return nil, fmt.Errorf("unknown method %s", method)
	}
}
