package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type AuditLog struct{ ent.Schema }

func (AuditLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").Optional(),
		field.String("action").MaxLen(120),
		field.String("target_type").Optional().MaxLen(80),
		field.String("target_id").Optional().MaxLen(80),
		field.String("ip").Optional().MaxLen(64),
		field.JSON("details", map[string]any{}).Optional(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}
