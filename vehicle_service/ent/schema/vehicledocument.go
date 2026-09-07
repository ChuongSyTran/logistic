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

type VehicleDocument struct {
	ent.Schema
}

func (VehicleDocument) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuidx.New).Unique(),
		field.UUID("vehicle_id", uuid.UUID{}),
		field.Enum("document_type").Values("registration", "inspection", "insurance", "license"),
		field.String("document_number").Optional(),
		field.String("file_url"),
		field.Time("issued_at").Optional().Nillable(),
		field.Time("expires_at").Optional().Nillable(),
		field.Enum("review_status").Values("pending", "approved", "rejected").Default("pending"),
		field.String("review_note").Optional(),
		field.UUID("reviewed_by", uuid.UUID{}).Optional().Nillable(),
		field.Time("reviewed_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (VehicleDocument) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("vehicle", Vehicle.Type).Ref("documents").Field("vehicle_id").Unique().Required(),
	}
}

func (VehicleDocument) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("vehicle_id", "document_type"),
		index.Fields("review_status"),
		index.Fields("expires_at"),
	}
}