// Copyright 2023 The Infratographer Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package schema

import (
	"strconv"

	"entgo.io/contrib/entgql"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/vektah/gqlparser/v2/ast"
	"go.infratographer.com/x/entx"
	"go.infratographer.com/x/gidx"
)

// Tenant holds the schema definition for the Tenant entity.
type Tenant struct {
	ent.Schema
}

// Mixin of the Tenant
func (Tenant) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entx.NewTimestampMixin(),
	}
}

// Fields of the Tenant.
func (Tenant) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Comment("ID for the tenant.").
			GoType(gidx.PrefixedID("")).
			DefaultFunc(func() gidx.PrefixedID { return gidx.MustNewID(TenantPrefix) }).
			Unique().
			Immutable(),
		field.String("name").
			Comment("The name of a tenant.").
			Annotations(
				entgql.OrderField("NAME"),
				entgql.Skip(entgql.SkipWhereInput),
			),
		field.String("description").
			Comment("An optional description of the tenant.").
			Optional().
			Annotations(
				entgql.Skip(entgql.SkipWhereInput),
			),
		field.String("parent_tenant_id").
			Comment("The ID of the parent tenant for the tenant.").
			Optional().
			Immutable().
			GoType(gidx.PrefixedID("")).
			Annotations(
				entgql.Type("ID"),
				entgql.Skip(entgql.SkipWhereInput, entgql.SkipMutationUpdateInput, entgql.SkipType),
				entx.EventsHookAdditionalSubject(),
			),
	}
}

// Indexes of the Tenant
func (Tenant) Indexes() []ent.Index {
	return []ent.Index{}
}

// Edges of the Tenant
func (Tenant) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("children", Tenant.Type).
			Annotations(
				entgql.RelayConnection(),
				entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput),
			).
			From("parent").
			Field("parent_tenant_id").
			Immutable().
			Unique(),
	}
}

// Annotations for the Tenant
func (Tenant) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entx.GraphKeyDirective("id"),
		prefixIDDirective(TenantPrefix),
		rolesDirective(true, true),
		entx.EventsHookSubjectName("tenant"),
		entgql.RelayConnection(),
		schema.Comment("Representation of a tenant."),
		entgql.Implements("ResourceOwner"),
		entgql.Implements("MetadataNode"),
		entgql.Mutations(
			entgql.MutationCreate().Description("Input information to create a tenant."),
			entgql.MutationUpdate().Description("Input information to update a tenant."),
		),
	}
}

func prefixIDDirective(prefix string) entgql.Annotation {
	var args []*ast.Argument
	if prefix != "" {
		args = append(args, &ast.Argument{
			Name: "prefix",
			Value: &ast.Value{
				Raw:  prefix,
				Kind: ast.StringValue,
			},
		})
	}

	return entgql.Directives(entgql.NewDirective("prefixedID", args...))
}

func rolesDirective(hasRoles bool, hasParentRoles bool) entgql.Annotation {
	args := []*ast.Argument{
		{
			Name: "hasRoles",
			Value: &ast.Value{
				Raw:  strconv.FormatBool(hasRoles),
				Kind: ast.StringValue,
			},
		},
		{
			Name: "hasParentRoles",
			Value: &ast.Value{
				Raw:  strconv.FormatBool(hasParentRoles),
				Kind: ast.StringValue,
			},
		},
	}

	return entgql.Directives(entgql.NewDirective("infratographerRoles", args...))
}
