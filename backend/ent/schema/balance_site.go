package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type BalanceSite struct {
	ent.Schema
}

func (BalanceSite) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "balance_sites"},
	}
}

func (BalanceSite) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (BalanceSite) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("platform").Values("newapi", "sub2api"),
		field.String("name").NotEmpty().MaxLen(100),
		field.String("base_url").NotEmpty().MaxLen(500),
		field.Enum("auth_mode").Values("password", "access_token", "cookie").Default("password"),
		field.String("username").Default("").MaxLen(255),
		field.String("email").Default("").MaxLen(255),
		field.String("password_encrypted").
			Default("").
			Sensitive().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.String("access_token_encrypted").
			Default("").
			Sensitive().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Bool("enabled").Default(true),
		field.Int("refresh_interval_minutes").Default(180).Positive(),
		field.Time("last_refresh_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.String("last_refresh_status").Default("").MaxLen(20),
		field.String("last_refresh_error").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
	}
}

func (BalanceSite) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("snapshots", BalanceKeySnapshot.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("bindings", AccountBalanceBinding.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (BalanceSite) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("enabled", "last_refresh_at"),
		index.Fields("platform"),
	}
}
