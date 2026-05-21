package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type RoundSnapshotScore struct {
	ent.Schema
}

func (RoundSnapshotScore) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.Int("snapshot_id"),
		field.Int("racer_id"),
		field.String("racer_name"),
		field.Int("points").Default(0),
		field.Int("position").Default(0),
	}
}
