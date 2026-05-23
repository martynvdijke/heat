package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

type RaceInfo struct {
	ent.Schema
}

func (RaceInfo) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "race_info"},
	}
}

func (RaceInfo) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.String("country").Default(""),
		field.String("track").Default(""),
		field.String("track_id").Default("monza"),
		field.Int("laps").Default(0),
	}
}
