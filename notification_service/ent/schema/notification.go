package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"github.com/logistic/pkg/uuidx"
)

type Notification struct {
	ent.Schema
}

func (Notification) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuidx.New).Unique(),
		field.UUID("user_id", uuid.UUID{}),
		field.Enum("recipient_role").Values("driver", "shipper", "admin").Default("driver"),

		field.String("type"),
		field.Enum("channel").Values("in_app", "push", "email", "sms").Default("in_app"),

		field.String("title"),
		field.Text("body"),

		field.Text("data").Optional(),

		field.String("ref_type").Optional(),
		field.String("ref_id").Optional(),

		field.Bool("is_read").Default(false),
		field.Enum("status").Values("pending", "sent", "failed", "read").Default("pending"),
		field.String("error_message").Optional(),
		field.Time("read_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (Notification) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "created_at"),

		index.Fields("user_id", "is_read"),
		index.Fields("type"),
		index.Fields("status"),
	}
}