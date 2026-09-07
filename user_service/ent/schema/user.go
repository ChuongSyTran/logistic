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

type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuidx.New).Unique(),
		// Optional: hồ sơ dựng từ auth_service chưa có phone. Unique vẫn đúng vì
		// Postgres coi các NULL là khác nhau.
		field.String("phone").Unique().Optional(),
		field.String("email").Unique().Optional(),
		field.String("full_name").Optional(),
		field.String("avatar_url").Optional(),
		field.String("password_hash").Sensitive(),
		field.Enum("role").Values("driver", "shipper", "admin"),
		field.Enum("status").Values("active", "banned", "suspended").Default("active"),
		field.String("status_reason").Optional(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("driver_profile", DriverProfile.Type).Unique(),
		edge.To("shipper_profile", ShipperProfile.Type).Unique(),
		edge.To("addresses", Address.Type),
		edge.To("devices", UserDevice.Type),
	}
}

func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("role", "status"),
		index.Fields("created_at"),
	}
}