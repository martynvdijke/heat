package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type RacerStats struct {
	ent.Schema
}

func (RacerStats) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.Int("racer_id").Unique(),
		field.Int("races").Default(0),
		field.Int("wins").Default(0),
		field.Int("gold").Default(0),
		field.Int("silver").Default(0),
		field.Int("bronze").Default(0),
		field.Int("fastest_laps").Default(0),
		field.Int("points").Default(0),
		field.Int("dnf").Default(0),
		field.Int("dns").Default(0),
	}
}
