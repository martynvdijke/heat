package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type DriverShare struct {
	ent.Schema
}

func (DriverShare) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.Int("racer_id").Unique(),
		field.String("token").Unique(),
		field.String("created_at").Default(""),
	}
}
