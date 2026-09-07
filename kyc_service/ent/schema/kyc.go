package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"github.com/logistic/pkg/uuidx"
)

type Kyc struct {
	ent.Schema
}

func (Kyc) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuidx.New).Unique(),
		field.UUID("user_id", uuid.UUID{}).Unique(),

		field.String("id_card_number").Optional().Nillable().Unique(),
		field.String("license_number").Optional().Nillable().Unique(),

		field.String("id_card_front_url").Optional(),
		field.String("id_card_back_url").Optional(),
		field.String("license_front_url").Optional(),
		field.String("license_back_url").Optional(),

		field.Enum("status").Values("pending", "approved", "rejected").Default("pending"),
		field.String("note").Optional(),
		field.UUID("reviewed_by", uuid.UUID{}).Optional().Nillable(),
		field.Time("reviewed_at").Optional().Nillable(),

		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Kyc) Edges() []ent.Edge {
	return nil
}

func (Kyc) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
		index.Fields("user_id").Unique(),
		index.Fields("created_at"),
	}
}
