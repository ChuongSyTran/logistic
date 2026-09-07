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

type DriverProfile struct {
	ent.Schema
}

func (DriverProfile) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuidx.New).Unique(),

		field.UUID("user_id", uuid.UUID{}),

		field.String("license_number").Optional().Nillable().Unique(),
		field.String("id_card").Optional().Nillable().Unique(),

		field.Float("rating").Default(5.0),
		field.Int("total_trips").Default(0),
		field.Enum("kyc_status").Values("pending", "approved", "rejected").Default("pending"),
		field.String("kyc_note").Optional(),
		field.UUID("kyc_reviewed_by", uuid.UUID{}).Optional().Nillable(),
		field.Time("kyc_reviewed_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (DriverProfile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("driver_profile").Field("user_id").Unique().Required(),
	}
}

func (DriverProfile) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("kyc_status"),
		index.Fields("user_id").Unique(),
	}
}