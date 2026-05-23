package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type Sector struct {
	ent.Schema
}

func (Sector) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.String("name"),
		field.String("track_id"),
		field.Int("order").Default(0),
	}
}
