-- +goose Up
-- +goose StatementBegin

CREATE TABLE tenants (
  id UUID PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
  name STRING NOT NULL,
  parent_tenant_id UUID NULL REFERENCES tenants(id),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  deleted_at TIMESTAMPTZ NULL
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE tenants;

-- +goose StatementEnd
