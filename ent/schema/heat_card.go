package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type HeatCard struct {
	ent.Schema
}

func (HeatCard) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.Int("racer_id"),
		field.String("location").Default("hand"),
		field.String("card_type").Default("heat"),
		field.Int("lap_added").Default(0),
	}
}
