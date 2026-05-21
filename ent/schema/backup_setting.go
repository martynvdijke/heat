package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type BackupSetting struct {
	ent.Schema
}

func (BackupSetting) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.Int("enabled").Default(1),
		field.Int("interval_hrs").Default(24),
		field.Int("retention_count").Optional().Default(7),
	}
}
