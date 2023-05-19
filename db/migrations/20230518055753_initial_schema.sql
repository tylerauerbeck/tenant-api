-- +goose Up
-- create "tenants" table
CREATE TABLE "tenants" (
  "id" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "parent_tenant_id" character varying NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "tenants_tenants_children" FOREIGN KEY ("parent_tenant_id") REFERENCES "tenants" ("id") ON UPDATE NO ACTION ON DELETE
  SET NULL
);
-- create index "tenant_created_at" to table: "tenants"
CREATE INDEX "tenant_created_at" ON "tenants" ("created_at");
-- create index "tenant_updated_at" to table: "tenants"
CREATE INDEX "tenant_updated_at" ON "tenants" ("updated_at");
-- +goose Down
-- reverse: create index "tenant_updated_at" to table: "tenants"
DROP INDEX "tenant_updated_at";
-- reverse: create index "tenant_created_at" to table: "tenants"
DROP INDEX "tenant_created_at";
-- reverse: create "tenants" table
DROP TABLE "tenants";
