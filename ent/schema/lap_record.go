package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type LapRecord struct {
	ent.Schema
}

func (LapRecord) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.Int("race_id").Default(0),
		field.Int("racer_id"),
		field.Int("lap_number"),
		field.Int("position"),
		field.Int("gear_used").Default(1),
		field.Int("heat_generated").Default(0),
		field.Int("turbo_used").Default(0),
		field.String("timestamp").Default(""),
	}
}
