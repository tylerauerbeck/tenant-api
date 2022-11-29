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

CREATE TABLE oidc_issuers (
  id UUID PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
  name STRING NOT NULL,
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  uri STRING UNIQUE NOT NULL,
  audience STRING NOT NULL,
  client_id STRING NOT NULL,
  subject_claim STRING NOT NULL DEFAULT 'sub',
  email_claim STRING NOT NULL DEFAULT 'email',
  name_claim STRING NOT NULL DEFAULT 'name',
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  deleted_at TIMESTAMPTZ NULL
);

CREATE TABLE users (
  id UUID PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
  name STRING NULL,
  email string NULL,
  oidc_issuer_id UUID NOT NULL REFERENCES oidc_issuers(id),
  oidc_subject STRING NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  deleted_at TIMESTAMPTZ NULL,
  UNIQUE (oidc_issuer_id, oidc_subject)
);

CREATE TABLE service_accounts (
  id UUID PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
  name STRING NULL,
  -- TODO: not sure we need this....figure out before submitting PR
  tenant_id UUID NULL REFERENCES tenants(id),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  deleted_at TIMESTAMPTZ NULL
);

CREATE TABLE tokens (
  id UUID PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
  resource_urn STRING NOT NULL,
  token STRING NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  last_used_at TIMESTAMPTZ NULL,
  deleted_at TIMESTAMPTZ NULL
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE tokens;
DROP TABLE service_accounts;
DROP TABLE users;
DROP TABLE oidc_issuers;
DROP TABLE tenants;

-- +goose StatementEnd
