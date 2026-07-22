package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Job struct{ ent.Schema }

func (Job) Fields() []ent.Field {
	return []ent.Field{
		field.String("job_key").MaxLen(80),
		field.String("type").MaxLen(80),
		field.String("status").Default("queued"),
		field.Int("progress").Default(0),
		field.String("message").Optional().MaxLen(2000),
		field.JSON("payload", map[string]any{}).Optional(),
		field.Time("started_at").Optional().Nillable(),
		field.Time("finished_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Job) Indexes() []ent.Index { return []ent.Index{index.Fields("job_key").Unique()} }
