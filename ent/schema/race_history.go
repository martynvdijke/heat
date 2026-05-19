package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type RaceHistory struct {
	ent.Schema
}

func (RaceHistory) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.String("name"),
		field.String("race_date"),
		field.String("country"),
		field.String("track"),
		field.String("track_id"),
		field.Int("total_laps"),
		field.String("race_type").Default("season"),
	}
}
