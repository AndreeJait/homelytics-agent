package entity

import "time"

// LoginRequest is the payload for merchant login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthSession is returned by a successful login and stored locally.
type AuthSession struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}
