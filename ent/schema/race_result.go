package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type RaceResult struct {
	ent.Schema
}

func (RaceResult) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.Int("race_id"),
		field.Int("racer_id"),
		field.String("racer_name"),
		field.Int("position"),
		field.Int("points"),
		field.Int("fastest_lap").Default(0),
		field.Int("finished").Default(1),
	}
}
