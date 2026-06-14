package entity

// TSNetAuthKey is a Tailscale auth key provisioned by the control plane.
type TSNetAuthKey struct {
	AuthKey string `json:"auth_key"`
}
