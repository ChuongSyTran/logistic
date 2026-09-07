package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"github.com/logistic/pkg/uuidx"
)

type NotificationTemplate struct {
	ent.Schema
}

func (NotificationTemplate) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuidx.New).Unique(),
		field.String("code"),
		field.String("name"),
		field.Enum("channel").Values("in_app", "push", "email", "sms").Default("in_app"),
		field.String("locale").Default("vi"),
		field.String("title_template"),
		field.Text("body_template"),
		field.Bool("is_active").Default(true),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (NotificationTemplate) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("code", "channel", "locale").Unique(),
		index.Fields("is_active"),
	}
}