package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type LegendAbility struct {
	ent.Schema
}

func (LegendAbility) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.String("name"),
		field.String("description"),
		field.String("ability_type"),
		field.String("racer_name"),
		field.Int("extension_id").Default(0),
	}
}
