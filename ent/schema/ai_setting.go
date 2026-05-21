package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type AISetting struct {
	ent.Schema
}

func (AISetting) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.String("track_extract_url").Optional(),
		field.String("api_key").Optional(),
		field.Int("enabled").Default(0),
		field.String("difficulty").Default("balanced"),
		field.Int("aggression").Default(50),
		field.Int("error_rate").Default(30),
		field.Int("consistency").Default(50),
	}
}
