package uuidx

import (
	"errors"

	"github.com/google/uuid"
)

const Size = 16

var (
	ErrInvalidLength = errors.New("uuidx: id phải dài đúng 16 byte")

	ErrInvalidString = errors.New("uuidx: chuỗi không phải UUID hợp lệ")

	ErrEmpty = errors.New("uuidx: id rỗng")
)

func New() uuid.UUID {
	return uuid.Must(uuid.NewV7())
}

func NewBytes() []byte {
	id := New()
	return id[:]
}

func ToBytes(id uuid.UUID) []byte {
	return id[:]
}

func FromBytes(b []byte) (uuid.UUID, error) {
	if len(b) == 0 {
		return uuid.Nil, nil
	}
	if len(b) != Size {
		return uuid.Nil, ErrInvalidLength
	}
	var id uuid.UUID
	copy(id[:], b)
	return id, nil
}

func String(b []byte) string {
	id, err := FromBytes(b)
	if err != nil || id == uuid.Nil {
		return ""
	}
	return id.String()
}

func Parse(s string) ([]byte, error) {
	if s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, ErrInvalidString
	}
	return id[:], nil
}

func ParseRequired(s string) ([]byte, error) {
	if s == "" {
		return nil, ErrEmpty
	}
	return Parse(s)
}

func ParseUUID(s string) (uuid.UUID, error) {
	if s == "" {
		return uuid.Nil, ErrEmpty
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, ErrInvalidString
	}
	return id, nil
}

func Strings(list [][]byte) []string {
	if len(list) == 0 {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, b := range list {
		out = append(out, String(b))
	}
	return out
}

func IsZero(b []byte) bool {
	if len(b) != Size {
		return true
	}
	for _, c := range b {
		if c != 0 {
			return false
		}
	}
	return true
}