package entity

import "time"

// LoginRequest is the payload for merchant login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthSession is returned by a successful login and stored locally.
type AuthSession struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
	MerchantID   string    `json:"merchant_id"`
}

// RefreshTokenRequest is sent to refresh an access token.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}
