package domain

import "time"

type TunnelID string
type UserID string

type Endpoint struct {
	Subdomain string
	Domain    string
	Port      int
}

type LocalTarget struct {
	Host string
	Port int
}

type TunnelStatus string

const (
	StatusActive   TunnelStatus = "active"
	StatusInactive TunnelStatus = "inactive"
)

type Tunnel struct {
	ID          TunnelID
	UserID      UserID
	Endpoints   []Endpoint
	LocalTarget LocalTarget
	Status      TunnelStatus
	CreatedAt   time.Time
}

func (t *Tunnel) Activate() error {
	t.Status = StatusActive
	return nil
}

func (t *Tunnel) Deactivate() error {
	t.Status = StatusInactive
	return nil
}
