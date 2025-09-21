-- +goose Up
-- +goose StatementBegin
create table tunnels (
    id UUID PRIMARY KEY,
    subdomain VARCHAR(63) UNIQUE NOT NULL,
    local_host VARCHAR(255) NOT NULL,
    local_port INT NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

create index idx_tunnels_subdomain on tunnels (subdomain);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tunnels;
-- +goose StatementEnd
