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

type BalanceKeySnapshot struct {
	ent.Schema
}

func (BalanceKeySnapshot) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "balance_key_snapshots"},
	}
}

func (BalanceKeySnapshot) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (BalanceKeySnapshot) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("site_id"),
		field.String("external_key_id").Default("").MaxLen(255),
		field.String("masked_key").Default("").MaxLen(255),
		field.String("key_last4").Default("").MaxLen(16),
		field.Int64("account_id").Optional().Nillable(),
		field.Enum("match_status").Values("unmatched", "matched", "ambiguous", "manual").Default("unmatched"),
		field.String("key_name").Default("").MaxLen(255),
		field.String("status").Default("").MaxLen(50),
		field.String("group_id").Default("").MaxLen(255),
		field.String("group_name").Default("").MaxLen(255),
		field.JSON("balance", map[string]any{}).
			Default(func() map[string]any { return map[string]any{} }).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.JSON("account_balance", map[string]any{}).
			Default(func() map[string]any { return map[string]any{} }).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.JSON("subscription_balance", map[string]any{}).
			Default(func() map[string]any { return map[string]any{} }).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.Float("rate_multiplier").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(10,4)"}),
		field.Time("fetched_at").
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (BalanceKeySnapshot) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("site", BalanceSite.Type).
			Ref("snapshots").
			Field("site_id").
			Required().
			Unique(),
		edge.To("account", Account.Type).
			Field("account_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.SetNull)),
	}
}

func (BalanceKeySnapshot) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("site_id", "external_key_id"),
		index.Fields("key_last4"),
		index.Fields("account_id", "fetched_at"),
	}
}
