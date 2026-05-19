package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type Season struct {
	ent.Schema
}

func (Season) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.String("name"),
		field.String("start_date"),
		field.String("end_date").Optional(),
		field.String("status").Default("active"),
		field.String("created_at").Default(""),
	}
}
