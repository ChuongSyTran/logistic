package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"github.com/logistic/pkg/uuidx"
)

type UserDevice struct {
	ent.Schema
}

func (UserDevice) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuidx.New).Unique(),
		field.UUID("user_id", uuid.UUID{}),
		field.String("device_token").Unique(),
		field.Enum("platform").Values("android", "ios", "web").Default("android"),
		field.String("device_name").Optional(),
		field.Bool("is_active").Default(true),
		field.Time("last_seen_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (UserDevice) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("devices").Field("user_id").Unique().Required(),
	}
}

func (UserDevice) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "is_active"),
	}
}