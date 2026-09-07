package schema

import (
	"github.com/logistic/pkg/ent/mixin"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
	"github.com/logistic/pkg/uuidx"
)

type Bids_Requirements struct {
	ent.Schema
}

func (Bids_Requirements) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuidx.New),
		field.UUID("bids_id", uuid.UUID{}),
		field.UUID("requirements_id", uuid.UUID{}),
	}
}

func (Bids_Requirements) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("bids", Bids.Type).Ref("bids_requirements").Field("bids_id").Unique().Required(),
		edge.From("requirements", Requirements.Type).Ref("bids_requirements").Field("requirements_id").Unique().Required(),
	}
}

func (Bids_Requirements) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AuditMixin{},
		mixin.SoftDeleteMixin{},
	}
}
