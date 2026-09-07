package repo

import (
	"entgo.io/ent/dialect/sql"
)

const (
	earthRadiusKm = 6371
	// Lọc thô ở tầng SQL: đủ rộng để giữ lại ứng viên nằm dọc tuyến.
	// Việc loại tinh do bộ chấm điểm ở tầng biz đảm nhiệm.
	coarseFilterRadiusKm = 150.0
)

func withinRadiusKm(latColumn, lngColumn string, lat, lng, radiusKm float64) func(*sql.Selector) {
	return func(s *sql.Selector) {
		s.Where(sql.P(func(b *sql.Builder) {
			b.WriteString("6371 * acos(least(1.0, cos(radians(")
			b.Arg(lat)
			b.WriteString(")) * cos(radians(").Ident(latColumn)
			b.WriteString(")) * cos(radians(").Ident(lngColumn)
			b.WriteString(") - radians(")
			b.Arg(lng)
			b.WriteString(")) + sin(radians(")
			b.Arg(lat)
			b.WriteString(")) * sin(radians(").Ident(latColumn)
			b.WriteString(")))) <= ")
			b.Arg(radiusKm)
		}))
	}
}
