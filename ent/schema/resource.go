package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Resource struct{ ent.Schema }

func (Resource) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").MinLen(1).MaxLen(16),
		field.String("name").MinLen(1).MaxLen(120),
		field.String("host").MaxLen(255),
		field.String("management_url").Optional().MaxLen(1024),
		field.String("access_token_ciphertext").Optional().Sensitive(),
		field.String("status").Default("unknown"),
		field.Int("node_count").Default(0),
		field.Int("client_count").Default(0),
		field.Int("socks_count").Default(0),
		field.Int("latency_ms").Default(0),
		field.Time("last_checked_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Resource) Indexes() []ent.Index { return []ent.Index{index.Fields("code").Unique()} }
