package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type GearShift struct {
	ent.Schema
}

func (GearShift) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.Int("racer_id"),
		field.Int("race_id"),
		field.Int("lap").Default(1),
		field.Int("gear").Default(1),
		field.Int("stress").Default(0),
	}
}
