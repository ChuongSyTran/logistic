package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"github.com/logistic/pkg/uuidx"
)

type ProcessedEvent struct {
	ent.Schema
}

func (ProcessedEvent) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuidx.New).Unique(),
		field.String("event_id").Unique(),
		field.String("routing_key"),
		field.String("source").Optional(),
		field.Time("processed_at").Default(time.Now).Immutable(),
	}
}

func (ProcessedEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("event_id").Unique(),
		index.Fields("processed_at"),
	}
}