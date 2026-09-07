package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/logistic/pkg/ent/mixin"
)

type ProcessedMessage struct {
	ent.Schema
}

func (ProcessedMessage) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
	}
}

func (ProcessedMessage) Edges() []ent.Edge {
	return nil
}

func (ProcessedMessage) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AuditMixin{},
	}
}