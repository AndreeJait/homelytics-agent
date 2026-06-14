package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/AndreeJait/go-utility/v2/logw"
	"github.com/AndreeJait/homelytics-agent/config"
	"github.com/AndreeJait/homelytics-agent/domain/entity"
	portOutbound "github.com/AndreeJait/homelytics-agent/port/outbound"
	"github.com/AndreeJait/homelytics-agent/usecase/command"
)

// TSNetCommandListener serves an HTTP endpoint on the tailnet for immediate push commands.
type TSNetCommandListener struct {
	cfg      *config.AppConfig
	vpn      portOutbound.VPN
	executor *command.Executor
}

// NewTSNetCommandListener creates a tailnet command listener.
func NewTSNetCommandListener(cfg *config.AppConfig, vpn portOutbound.VPN, executor *command.Executor) *TSNetCommandListener {
	return &TSNetCommandListener{cfg: cfg, vpn: vpn, executor: executor}
}

// Start begins the listener in a goroutine. It exits when ctx is cancelled.
func (l *TSNetCommandListener) Start(ctx context.Context) {
	if !l.cfg.TSNet.EnableCommandListener {
		return
	}

	// Start the listener in a goroutine and retry until tsnet is available.
	go l.serve(ctx)
}

func (l *TSNetCommandListener) serve(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := l.tryServe(ctx); err != nil {
				logw.CtxWarningf(ctx, "tsnet listener: %v", err)
				continue
			}
			return
		}
	}
}

func (l *TSNetCommandListener) tryServe(ctx context.Context) error {
	listener, err := l.vpn.Listen("tcp", l.cfg.TSNet.CommandListenerAddr)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/commands", l.handleCommand)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	server := &http.Server{Handler: mux}

	go func() {
		logw.CtxInfof(ctx, "tsnet listener: serving on %s", l.cfg.TSNet.CommandListenerAddr)
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logw.CtxWarningf(ctx, "tsnet listener: serve error: %v", err)
		}
	}()

	go func() {
		<-ctx.Done()
		_ = server.Close()
		_ = listener.Close()
	}()

	return nil
}

func (l *TSNetCommandListener) handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var cmd entity.Command
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	result, err := l.executor.Execute(r.Context(), cmd)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	if result != nil {
		_ = json.NewEncoder(w).Encode(result)
	} else {
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}
