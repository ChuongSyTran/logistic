package schema

import (
	"github.com/logistic/pkg/ent/mixin"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
	"github.com/logistic/pkg/uuidx"
)

type Asks struct {
	ent.Schema
}

func (Asks) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuidx.New),
		field.UUID("driver_id", uuid.UUID{}),
		field.String("driver_phone"),
		field.String("driver_mail"),
		field.UUID("vehicle_id", uuid.UUID{}),
		field.Int8("vehicle_type"),
		field.Float("capacity_weight_kg").Comment("Maxinum weight by kilogram"),
		field.Float("capacity_volume_cbm").Comment("Maxinum weight by CBM"),
		field.Float("available_volume_m3"),
		field.Float("available_weight_kg"),
		field.Float("min_price").Optional().Nillable(),
		field.Float("desired_deposit"),
		field.String("zone_id"),
		field.Float("origin_lat"),
		field.Float("origin_lng"),
		field.Float("destination_lat"),
		field.Float("destination_lng"),
		field.Bytes("route_id").Optional(),
		field.Int8("status"),
		field.Time("expires_at").Optional(),
	}
}

func (Asks) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("matches", Match.Type),
	}
}

func (Asks) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AuditMixin{},
		mixin.SoftDeleteMixin{},
	}
}
