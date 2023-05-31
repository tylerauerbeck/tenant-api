//go:build ignore

package main

import (
	"log"

	"entgo.io/contrib/entgql"
	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"go.infratographer.com/x/entx"
	"go.infratographer.com/x/events"
)

func main() {
	xExt, err := entx.NewExtension(
		entx.WithFederation(),
		entx.WithJSONScalar(),
		entx.WithEventHooks(),
	)
	if err != nil {
		log.Fatalf("creating entx extension: %v", err)
	}

	gqlExt, err := entgql.NewExtension(
		// Tell Ent to generate a GraphQL schema for
		// the Ent schema in a file named ent.graphql.
		entgql.WithSchemaGenerator(),
		entgql.WithSchemaPath("schema/ent.graphql"),
		entgql.WithConfigPath("gqlgen.yml"),
		entgql.WithWhereInputs(true),
		entgql.WithSchemaHook(xExt.GQLSchemaHooks()...),
	)
	if err != nil {
		log.Fatalf("creating entgql extension: %v", err)
	}

	opts := []entc.Option{
		entc.Extensions(
			xExt,
			gqlExt,
		),
		entc.TemplateDir("./internal/ent/templates"),
		entc.FeatureNames("intercept"),
		entc.Dependency(
			entc.DependencyType(&events.Publisher{}),
		),
	}

	if err := entc.Generate("./internal/ent/schema", &gen.Config{
		Target:   "./internal/ent/generated",
		Package:  "go.infratographer.com/tenant-api/internal/ent/generated",
		Header:   entx.CopyrightHeader,
		Features: []gen.Feature{gen.FeatureVersionedMigration},
	}, opts...); err != nil {
		log.Fatalf("running ent codegen: %v", err)
	}
}
