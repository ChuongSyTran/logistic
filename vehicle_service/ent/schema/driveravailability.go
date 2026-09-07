package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"github.com/logistic/pkg/uuidx"
)

type DriverAvailability struct {
	ent.Schema
}

func (DriverAvailability) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuidx.New).Unique(),
		field.UUID("driver_id", uuid.UUID{}).Unique(),
		field.UUID("vehicle_id", uuid.UUID{}),
		field.Bool("is_online").Default(false),
		field.Float("available_weight_kg").Default(0),
		field.Float("available_volume_cbm").Default(0),
		field.Float("current_lat").Default(0),
		field.Float("current_lng").Default(0),
		field.String("zone_id").Optional(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (DriverAvailability) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("vehicle", Vehicle.Type).Ref("availability").Field("vehicle_id").Unique().Required(),
	}
}

func (DriverAvailability) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("is_online", "zone_id"),
		index.Fields("vehicle_id"),
	}
}