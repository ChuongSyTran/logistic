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

type ShipperProfile struct {
	ent.Schema
}

func (ShipperProfile) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuidx.New).Unique(),
		field.UUID("user_id", uuid.UUID{}),
		field.String("company_name").Optional(),
		field.String("tax_code").Optional(),
		field.String("business_address").Optional(),
		field.Int("total_orders").Default(0),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (ShipperProfile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("shipper_profile").Field("user_id").Unique().Required(),
	}
}

func (ShipperProfile) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id").Unique(),
	}
}