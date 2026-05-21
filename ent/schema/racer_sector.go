package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type RacerSector struct {
	ent.Schema
}

func (RacerSector) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.Int("race_id").Default(0),
		field.Int("racer_id"),
		field.Int("sector_id"),
		field.Int("lap").Default(1),
		field.String("entry_time").Default(""),
		field.String("exit_time").Default(""),
	}
}
