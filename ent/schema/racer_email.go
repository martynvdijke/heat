package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type RacerEmail struct {
	ent.Schema
}

func (RacerEmail) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.Int("racer_id").Unique().Optional(),
		field.String("email").Optional(),
	}
}
