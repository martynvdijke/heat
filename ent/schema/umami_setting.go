package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type UmamiSetting struct {
	ent.Schema
}

func (UmamiSetting) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.String("url").Optional(),
		field.String("website_id").Optional(),
		field.Int("enabled").Default(0),
	}
}
