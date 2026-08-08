package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type Track struct {
	ent.Schema
}

func (Track) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique(),
		field.String("name"),
		field.String("country"),
		field.String("geojson").Default(""),
		field.Int("length_km").Default(0),
		field.String("lap_record").Default(""),
		field.Int("use_map_image").Default(0),
		field.String("map_image_url").Default(""),
		field.Int("refresh_geojson").Default(1),
		field.Int("extension_id").Default(0),
	}
}
