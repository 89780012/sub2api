package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type AccountBalanceBinding struct {
	ent.Schema
}

func (AccountBalanceBinding) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "account_balance_bindings"},
	}
}

func (AccountBalanceBinding) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (AccountBalanceBinding) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("account_id").Unique(),
		field.Int64("site_id"),
		field.String("external_key_id").Optional().Nillable().MaxLen(255),
		field.String("key_last4").Default("").MaxLen(16),
		field.Enum("match_mode").Values("auto", "manual").Default("manual"),
	}
}

func (AccountBalanceBinding) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("account", Account.Type).
			Field("account_id").
			Required().
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("site", BalanceSite.Type).
			Ref("bindings").
			Field("site_id").
			Required().
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (AccountBalanceBinding) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("site_id", "key_last4"),
	}
}
