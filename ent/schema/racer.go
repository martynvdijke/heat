package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type Racer struct {
	ent.Schema
}

func (Racer) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.String("name"),
		field.String("profile_picture").Default(""),
		field.String("car_color").Default(""),
		field.String("car_name").Default(""),
		field.Int("points").Default(0),
		field.Int("rank").Default(0),
		field.Int("position").Default(0),
		field.Int("team_id").Optional().Default(0),
	}
}
