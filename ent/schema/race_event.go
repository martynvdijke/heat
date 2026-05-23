package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type RaceEvent struct {
	ent.Schema
}

func (RaceEvent) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.Int("race_id").Default(0),
		field.Int("lap").Default(1),
		field.String("event_type"),
		field.Int("racer_id"),
		field.Int("racer_id2").Default(0),
		field.String("note").Default(""),
		field.String("timestamp").Default(""),
	}
}
