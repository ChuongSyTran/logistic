package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"github.com/logistic/pkg/uuidx"
)

type NotificationPreference struct {
	ent.Schema
}

func (NotificationPreference) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuidx.New).Unique(),
		field.UUID("user_id", uuid.UUID{}).Unique(),
		field.Bool("in_app_enabled").Default(true),
		field.Bool("push_enabled").Default(true),
		field.Bool("email_enabled").Default(false),
		field.Bool("sms_enabled").Default(false),
		field.Bool("match_events_enabled").Default(true),
		field.Bool("promotion_enabled").Default(true),
		field.String("quiet_hours_start").Optional(),
		field.String("quiet_hours_end").Optional(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (NotificationPreference) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id").Unique(),
	}
}