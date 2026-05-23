package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type Quote struct {
	ent.Schema
}

func (Quote) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.String("text"),
		field.String("author").Default("Commentator"),
		field.String("created_at").Default(""),
	}
}
