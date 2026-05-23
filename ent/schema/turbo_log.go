package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type TurboLog struct {
	ent.Schema
}

func (TurboLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.Int("racer_id"),
		field.Int("race_id").Default(0),
		field.Int("lap").Default(1),
		field.Int("times_used").Default(1),
	}
}
