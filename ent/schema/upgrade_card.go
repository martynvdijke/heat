package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type UpgradeCard struct {
	ent.Schema
}

func (UpgradeCard) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.String("name"),
		field.String("description"),
		field.String("card_type").Default("upgrade"),
		field.Int("cost").Default(0),
		field.String("effects").Default("{}"),
	}
}
