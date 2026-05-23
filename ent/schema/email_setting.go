package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type EmailSetting struct {
	ent.Schema
}

func (EmailSetting) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.String("smtp_host").Optional(),
		field.Int("smtp_port").Default(587),
		field.String("username").Optional(),
		field.String("password").Optional(),
		field.String("from_addr").Optional(),
		field.Int("enabled").Default(0),
	}
}
