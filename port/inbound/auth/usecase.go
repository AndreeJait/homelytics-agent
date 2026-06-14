package auth

import (
	"context"

	"github.com/AndreeJait/homelytics-agent/domain/entity"
)

// UseCase orchestrates merchant login against the control plane.
type UseCase interface {
	Login(ctx context.Context, req entity.LoginRequest) (*entity.AuthSession, error)
}
