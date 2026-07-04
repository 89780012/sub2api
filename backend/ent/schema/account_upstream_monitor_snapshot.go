package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AccountUpstreamMonitorSnapshot stores the latest read-only upstream account
// monitoring snapshot for one local account.
type AccountUpstreamMonitorSnapshot struct {
	ent.Schema
}

func (AccountUpstreamMonitorSnapshot) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "account_upstream_monitor_snapshots"},
	}
}

func (AccountUpstreamMonitorSnapshot) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (AccountUpstreamMonitorSnapshot) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("account_id").
			Unique(),
		field.Enum("provider").
			Values("sub2api", "newapi", "unsupported", "unknown").
			Default("unknown"),
		field.Enum("status").
			Values("unknown", "unsupported", "success", "failed").
			Default("unknown"),
		field.String("site_url").
			Optional().
			Nillable().
			MaxLen(512),
		field.Float("balance").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.String("balance_unit").
			Optional().
			Nillable().
			MaxLen(32),
		field.Float("quota").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("quota_used").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("today_cost").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("total_cost").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Time("last_checked_at").
			Optional().
			Nillable(),
		field.Time("last_success_at").
			Optional().
			Nillable(),
		field.String("last_error").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.JSON("raw_meta", map[string]any{}).
			Default(func() map[string]any { return map[string]any{} }).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
	}
}

func (AccountUpstreamMonitorSnapshot) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status", "last_checked_at"),
		index.Fields("provider"),
	}
}
