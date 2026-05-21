package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type NotificationSetting struct {
	ent.Schema
}

func (NotificationSetting) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.String("gotify_url").Optional(),
		field.String("gotify_token").Optional(),
		field.Int("notify_winner").Default(1),
		field.Int("notify_race_start").Default(0),
		field.Int("notify_podium").Default(0),
	}
}
