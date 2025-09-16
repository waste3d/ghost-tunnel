package domain

import "context"

type TunnelRepository interface {
	Save(ctx context.Context, tunnel *Tunnel) error
	FindByID(ctx context.Context, id TunnelID) (*Tunnel, error)
	FindBySubdomain(ctx context.Context, subdomain string) (*Tunnel, error)
	Delete(ctx context.Context, subdomain string) error
}
