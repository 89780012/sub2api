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

// AccountUpstreamMonitorRate stores the current upstream group/model rate
// snapshot for one local account.
type AccountUpstreamMonitorRate struct {
	ent.Schema
}

func (AccountUpstreamMonitorRate) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "account_upstream_monitor_rates"},
	}
}

func (AccountUpstreamMonitorRate) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (AccountUpstreamMonitorRate) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("account_id"),
		field.Enum("provider").
			Values("sub2api", "newapi").
			Default("sub2api"),
		field.String("rate_key").
			NotEmpty().
			MaxLen(256),
		field.String("display_name").
			NotEmpty().
			MaxLen(256),
		field.String("description").
			Optional().
			Nillable().
			MaxLen(512),
		field.Float("ratio").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("completion_ratio").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Time("first_seen_at"),
		field.Time("last_seen_at"),
	}
}

func (AccountUpstreamMonitorRate) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("account_id", "rate_key").
			Unique(),
		index.Fields("account_id", "last_seen_at"),
	}
}
