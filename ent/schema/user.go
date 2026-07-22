package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type User struct{ ent.Schema }

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("username").MinLen(3).MaxLen(64),
		field.String("password_hash").Sensitive(),
		field.String("role").Default("admin"),
		field.Bool("enabled").Default(true),
		field.Bool("must_change_password").Default(true),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (User) Indexes() []ent.Index { return []ent.Index{index.Fields("username").Unique()} }
