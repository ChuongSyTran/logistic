package uuidx_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/logistic/pkg/uuidx"
)

func TestFromBytesRejectsStringifiedUUID(t *testing.T) {
	raw := []byte(uuid.Must(uuid.NewV7()).String())
	if len(raw) != 36 {
		t.Fatalf("tiền đề sai: chuỗi UUID phải dài 36, được %d", len(raw))
	}
	if _, err := uuidx.FromBytes(raw); err != uuidx.ErrInvalidLength {
		t.Fatalf("mong đợi ErrInvalidLength, được %v", err)
	}
}

func TestRoundTrip(t *testing.T) {
	original := uuidx.New()
	s := uuidx.String(uuidx.ToBytes(original))
	if s != original.String() {
		t.Fatalf("round trip lệch: %s != %s", s, original.String())
	}

	b, err := uuidx.Parse(s)
	if err != nil {
		t.Fatalf("Parse lỗi: %v", err)
	}
	back, err := uuidx.FromBytes(b)
	if err != nil {
		t.Fatalf("FromBytes lỗi: %v", err)
	}
	if back != original {
		t.Fatalf("giá trị lệch sau round trip: %v != %v", back, original)
	}
}

func TestV7IsMonotonic(t *testing.T) {
	prev := uuidx.New()
	for i := 0; i < 1000; i++ {
		next := uuidx.New()
		if next.String() <= prev.String() {
			t.Fatalf("id thứ %d không tăng: %s <= %s", i, next, prev)
		}
		prev = next
	}
}

func TestV7Version(t *testing.T) {
	if v := uuidx.New().Version(); v != 7 {
		t.Fatalf("mong đợi UUID v7, được v%d", v)
	}
}

func TestEmptyAndNil(t *testing.T) {
	if s := uuidx.String(nil); s != "" {
		t.Fatalf("nil phải cho chuỗi rỗng, được %q", s)
	}
	b, err := uuidx.Parse("")
	if err != nil || b != nil {
		t.Fatalf("chuỗi rỗng phải cho (nil, nil), được (%v, %v)", b, err)
	}
	if _, err := uuidx.ParseRequired(""); err != uuidx.ErrEmpty {
		t.Fatalf("mong đợi ErrEmpty, được %v", err)
	}
	if _, err := uuidx.Parse("không-phải-uuid"); err != uuidx.ErrInvalidString {
		t.Fatalf("mong đợi ErrInvalidString, được %v", err)
	}
}

func TestIsZero(t *testing.T) {
	if !uuidx.IsZero(make([]byte, 16)) {
		t.Fatal("16 byte 0 phải là zero")
	}
	if uuidx.IsZero(uuidx.NewBytes()) {
		t.Fatal("id mới sinh không được là zero")
	}
}

func BenchmarkString(b *testing.B) {
	raw := uuidx.NewBytes()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = uuidx.String(raw)
	}
}

func BenchmarkParse(b *testing.B) {
	s := uuidx.New().String()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = uuidx.Parse(s)
	}
}