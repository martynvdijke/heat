package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type RoundSnapshot struct {
	ent.Schema
}

func (RoundSnapshot) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.String("race_name"),
		field.String("race_date"),
		field.Int("round").Default(1),
		field.String("created_at").Default(""),
		field.Int("season_id").Optional().Default(1),
	}
}
