package schema

import (
	"github.com/logistic/pkg/ent/mixin"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
	"github.com/logistic/pkg/uuidx"
)

type Wallet struct {
	ent.Schema
}

func (Wallet) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuidx.New).Unique(),
		field.UUID("user_id", uuid.UUID{}),
		field.Uint8("user_type"),
		field.Int64("balance"),
		field.String("currency").Default("VND"),
		field.Uint8("status"),
	}
}

func (Wallet) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("transactions", Transaction.Type),
	}
}

func (Wallet) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AuditMixin{},
		mixin.SoftDeleteMixin{},
	}
}