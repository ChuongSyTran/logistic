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

type VehicleLocation struct {
	ent.Schema
}

func (VehicleLocation) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuidx.New).Unique(),
		field.UUID("vehicle_id", uuid.UUID{}).Unique(),
		field.UUID("driver_id", uuid.UUID{}),
		field.Float("latitude"),
		field.Float("longitude"),
		field.Float("heading").Default(0),
		field.Float("speed_kph").Default(0),
		field.String("zone_id").Optional(),
		field.Time("recorded_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (VehicleLocation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("vehicle", Vehicle.Type).Ref("location").Field("vehicle_id").Unique().Required(),
	}
}

func (VehicleLocation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("zone_id"),
		index.Fields("driver_id"),
	}
}