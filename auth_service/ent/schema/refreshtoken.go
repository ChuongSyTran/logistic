package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type RefreshToken struct {
	ent.Schema
}

func (RefreshToken) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Unique().Immutable(),

		field.UUID("user_id", uuid.UUID{}).Immutable(),
		field.Time("expires_at").Immutable(),

		field.Time("revoked_at").Optional().Nillable(),

		field.Time("used_at").Optional().Nillable(),

		field.String("user_agent").Optional(),
		field.String("ip").Optional(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (RefreshToken) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),

		index.Fields("expires_at"),
	}
}