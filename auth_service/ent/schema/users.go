package schema

import (
	softdelete "auth_service/ent/softdelete"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"github.com/logistic/pkg/uuidx"
)

type Users struct {
	ent.Schema
}

func (Users) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuidx.New).Unique().Immutable(),

		field.String("email").Unique(),
		field.String("full_name").Optional().Nillable().StructTag(`json:"fullName"`),
		field.String("avatar").Optional().Nillable().StructTag(`json:"avatar"`),
		field.String("password").Optional().Nillable().Sensitive(),
		field.String("totp_secret").Optional().Nillable().Sensitive(),
		field.String("google_id").Optional().Nillable().Sensitive(),

		field.Enum("role").Values("driver", "shipper", "admin").Default("shipper"),

		field.Time("created_at").Default(time.Now).Immutable().StructTag(`json:"createdAt"`),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).StructTag(`json:"updatedAt"`),
	}
}

func (Users) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("google_id"),
	}
}

func (Users) Mixin() []ent.Mixin {
	return []ent.Mixin{
		softdelete.SoftDeleteMixin{},
	}
}