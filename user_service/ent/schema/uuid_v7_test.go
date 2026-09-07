package schema_test

import (
	"testing"

	"user_service/ent/address"
	"user_service/ent/driverprofile"
	"user_service/ent/shipperprofile"
	"user_service/ent/user"
	"user_service/ent/userdevice"

	_ "user_service/ent"

	"github.com/google/uuid"
)

func TestPrimaryKeysAreUUIDv7(t *testing.T) {
	defaults := map[string]func() uuid.UUID{
		"user":            user.DefaultID,
		"address":         address.DefaultID,
		"driver_profile":  driverprofile.DefaultID,
		"shipper_profile": shipperprofile.DefaultID,
		"user_device":     userdevice.DefaultID,
	}

	for name, gen := range defaults {
		if gen == nil {
			t.Errorf("%s: không có hàm sinh id mặc định", name)
			continue
		}
		if v := gen().Version(); v != 7 {
			t.Errorf("%s: id là UUID v%d, phải là v7", name, v)
		}
	}
}

func TestGeneratedIDsAreOrdered(t *testing.T) {
	prev := user.DefaultID()
	for i := 0; i < 500; i++ {
		next := user.DefaultID()
		if next.String() <= prev.String() {
			t.Fatalf("id thứ %d không tăng: %s <= %s", i, next, prev)
		}
		prev = next
	}
}