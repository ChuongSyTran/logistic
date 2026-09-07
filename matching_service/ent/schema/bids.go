package schema

import (
	"github.com/logistic/pkg/ent/mixin"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
	"github.com/logistic/pkg/uuidx"
)

type Bids struct {
	ent.Schema
}

func (Bids) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuidx.New),
		field.UUID("shipper_id", uuid.UUID{}),
		field.String("shipper_phone"),
		field.String("shipper_mail"),
		field.UUID("consignee_id", uuid.UUID{}),
		field.String("consignee_phone"),
		field.String("consignee_mail"),
		field.Float("volume_m3"),
		field.Float("weight_kg"),
		field.Float("max_price").Optional().Nillable(),
		field.Float("cargo_value"),
		field.Float("required_deposit"),
		field.Float("desired_deposit"),
		field.String("zone_id"),
		field.Float("origin_lat"),
		field.Float("origin_lng"),
		field.Float("destination_lat"),
		field.Float("destination_lng"),
		// Giá tài xế đã báo và chuyến tương ứng, ghi lại lúc đơn vào NEGOTIATING.
		// Không có hai cột này thì lúc chốt không còn nguồn nào biết đã thoả thuận
		// bao nhiêu, và ai là người báo.
		field.Float("offered_price").Optional(),
		field.UUID("offered_ask_id", uuid.UUID{}).Optional(),
		field.Int8("status"),
		field.Time("expires_at").Optional(),
	}
}

func (Bids) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("matches", Match.Type),
		edge.To("bids_requirements", Bids_Requirements.Type),
	}
}

func (Bids) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AuditMixin{},
		mixin.SoftDeleteMixin{},
	}
}
