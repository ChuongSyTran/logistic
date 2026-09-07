package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
	"github.com/logistic/pkg/ent/mixin"
	"github.com/logistic/pkg/uuidx"
)

type Transaction struct {
	ent.Schema
}

func (Transaction) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuidx.New).Unique(),
		field.UUID("wallet_id", uuid.UUID{}),
		field.Int64("amount"),
		field.Uint8("transaction_type"),
		field.String("reference_id"),
		field.Uint8("status"),
		field.String("description"),
	}
}

func (Transaction) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("wallet", Wallet.Type).Ref("transactions").Field("wallet_id").Unique().Required(),
	}
}

func (Transaction) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AuditMixin{},
		mixin.SoftDeleteMixin{},
	}
}