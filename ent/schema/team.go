package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type Team struct {
	ent.Schema
}

func (Team) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.String("name").Unique(),
		field.String("color").Default("#d40000"),
		field.String("created_at").Default(""),
	}
}
