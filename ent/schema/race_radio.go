package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

type RaceRadio struct {
	ent.Schema
}

func (RaceRadio) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "race_radio"},
	}
}

func (RaceRadio) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.Int("race_id").Default(0),
		field.Int("racer_id"),
		field.String("message"),
		field.String("timestamp").Default(""),
	}
}
