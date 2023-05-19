-- +goose Up
-- +goose StatementBegin

CREATE TABLE tenants (
  id VARCHAR(29) PRIMARY KEY NOT NULL,
  name TEXT NOT NULL,
  parent_tenant_id VARCHAR(29) NULL REFERENCES tenants(id),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  deleted_at TIMESTAMPTZ NULL
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE tenants;

-- +goose StatementEnd
