package ipc

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/AndreeJait/go-utility/v2/logw"
	"github.com/AndreeJait/go-utility/v2/statusw"
	"github.com/AndreeJait/homelytics-agent/domain/entity"
	authUC "github.com/AndreeJait/homelytics-agent/port/inbound/auth"
	runtimeUC "github.com/AndreeJait/homelytics-agent/port/inbound/runtime"
	statusUC "github.com/AndreeJait/homelytics-agent/port/inbound/status"
	tsnetUC "github.com/AndreeJait/homelytics-agent/port/inbound/tsnet"
)

// Server listens on a Unix domain socket and routes JSON command requests.
type Server struct {
	listener     net.Listener
	authUC       authUC.UseCase
	tsnetAuthUC  tsnetUC.UseCase
	runtimeUC    runtimeUC.UseCase
	statusUC     statusUC.UseCase
}

// NewServer creates an IPC server bound to the given socket path.
func NewServer(socketPath string, authUC authUC.UseCase, tsnetAuthUC tsnetUC.UseCase, runtimeUC runtimeUC.UseCase, statusUC statusUC.UseCase) (*Server, error) {
	if err := os.RemoveAll(socketPath); err != nil {
		return nil, fmt.Errorf("ipc: remove stale socket: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(socketPath), 0o750); err != nil {
		return nil, fmt.Errorf("ipc: create socket directory: %w", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("ipc: listen on %s: %w", socketPath, err)
	}

	if err := os.Chmod(socketPath, 0o666); err != nil {
		_ = listener.Close()
		return nil, fmt.Errorf("ipc: chmod socket: %w", err)
	}

	return &Server{
		listener:    listener,
		authUC:      authUC,
		tsnetAuthUC: tsnetAuthUC,
		runtimeUC:   runtimeUC,
		statusUC:    statusUC,
	}, nil
}

// Serve accepts connections until the listener is closed.
func (s *Server) Serve(ctx context.Context) error {
	logw.CtxInfof(ctx, "ipc: listening on %s", s.listener.Addr().String())

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				logw.CtxErrorf(ctx, "ipc: accept failed: %v", err)
				continue
			}
		}

		go s.handleConnection(ctx, conn)
	}
}

// Close stops the IPC server.
func (s *Server) Close() error {
	return s.listener.Close()
}

func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		_ = s.encodeError(conn, "", err)
		return
	}

	var req entity.CommandRequest
	if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
		_ = s.encodeError(conn, "", fmt.Errorf("ipc: invalid request: %w", err))
		return
	}

	resp, err := s.route(ctx, req)
	if err != nil {
		_ = s.encodeError(conn, req.ID, err)
		return
	}

	data, _ := json.Marshal(resp)
	_, _ = conn.Write(data)
	_, _ = conn.Write([]byte("\n"))
}

func (s *Server) route(ctx context.Context, req entity.CommandRequest) (*entity.CommandResponse, error) {
	switch req.Method {
	case "login":
		return s.handleLogin(ctx, req)
	case "tsnet.auth":
		return s.handleTSNetAuth(ctx, req)
	case "runtime.status":
		return s.handleRuntimeStatus(ctx, req)
	case "status":
		return s.handleStatus(ctx, req)
	default:
		return nil, statusw.InvalidReqParam.WithCustomMessage("unknown method: " + req.Method)
	}
}

func (s *Server) handleLogin(ctx context.Context, req entity.CommandRequest) (*entity.CommandResponse, error) {
	var payload entity.LoginRequest
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		return nil, statusw.InvalidReqParam.WithCustomMessage("invalid login payload")
	}

	session, err := s.authUC.Login(ctx, payload)
	if err != nil {
		return nil, err
	}

	return s.encodeData(req.ID, session)
}

func (s *Server) handleTSNetAuth(ctx context.Context, req entity.CommandRequest) (*entity.CommandResponse, error) {
	key, err := s.tsnetAuthUC.GetAuthKey(ctx)
	if err != nil {
		return nil, err
	}

	return s.encodeData(req.ID, key)
}

func (s *Server) handleRuntimeStatus(ctx context.Context, req entity.CommandRequest) (*entity.CommandResponse, error) {
	status, err := s.runtimeUC.Status(ctx)
	if err != nil {
		return nil, err
	}

	return s.encodeData(req.ID, status)
}

func (s *Server) handleStatus(ctx context.Context, req entity.CommandRequest) (*entity.CommandResponse, error) {
	status, err := s.statusUC.Get(ctx)
	if err != nil {
		return nil, err
	}

	return s.encodeData(req.ID, status)
}

func (s *Server) encodeData(id string, data any) (*entity.CommandResponse, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return &entity.CommandResponse{ID: id, OK: true, Data: raw}, nil
}

func (s *Server) encodeError(conn net.Conn, id string, err error) error {
	code, message := "INTERNAL_ERROR", err.Error()
	if se, ok := err.(*statusw.Error); ok {
		code = se.CustomCode
		message = se.Message
	}

	resp := entity.CommandResponse{
		ID:  id,
		OK:  false,
		Error: &entity.ErrorDetail{Code: code, Message: message},
	}

	data, _ := json.Marshal(resp)
	_, _ = conn.Write(data)
	_, _ = conn.Write([]byte("\n"))
	return nil
}
