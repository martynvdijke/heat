package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type WLEDSetting struct {
	ent.Schema
}

func (WLEDSetting) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.String("url").Optional(),
		field.Int("enabled").Default(0),
		field.String("presets").Optional(),
	}
}
