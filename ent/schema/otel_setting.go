package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type OTelSetting struct {
	ent.Schema
}

func (OTelSetting) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.String("endpoint").Optional(),
		field.Int("traces_enabled").Default(0),
		field.Int("metrics_enabled").Default(0),
		field.Int("logs_enabled").Default(0),
	}
}
