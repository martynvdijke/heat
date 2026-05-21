package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type PlayerSession struct {
	ent.Schema
}

func (PlayerSession) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.Int("racer_id").Unique(),
		field.String("token").Unique(),
		field.String("device_name").Default(""),
		field.String("last_seen").Default(""),
		field.String("created_at").Default(""),
	}
}
