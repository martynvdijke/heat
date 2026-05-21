package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type RacerLegendAbility struct {
	ent.Schema
}

func (RacerLegendAbility) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.Int("racer_id"),
		field.Int("ability_id"),
		field.Int("active").Default(1),
	}
}
