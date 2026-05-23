package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type AdminUser struct {
	ent.Schema
}

func (AdminUser) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.String("username").Unique(),
		field.String("password"),
	}
}
