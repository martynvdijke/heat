package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type PlayerUpgrade struct {
	ent.Schema
}

func (PlayerUpgrade) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.Int("racer_id"),
		field.Int("upgrade_id"),
		field.Int("season_id").Default(0),
		field.Int("equipped").Default(1),
		field.Int("round_bought").Default(0),
	}
}
