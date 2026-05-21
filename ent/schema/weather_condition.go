package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type WeatherCondition struct {
	ent.Schema
}

func (WeatherCondition) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.Int("race_id").Default(0),
		field.String("condition").Default("dry"),
		field.Int("lap_start").Default(1),
		field.Int("lap_end").Default(999),
		field.Float("grip_modifier").Default(1.0),
	}
}
