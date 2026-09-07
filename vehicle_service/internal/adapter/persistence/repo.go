package persistence

import (
	"context"
	"log"
	"strings"
	"time"

	"vehicle_service/ent"
	"vehicle_service/ent/driveravailability"
	"vehicle_service/ent/vehicle"
	"vehicle_service/ent/vehicledocument"
	"vehicle_service/ent/vehiclelocation"
	"vehicle_service/internal/app"
	"vehicle_service/internal/entity"
	"vehicle_service/internal/mapper"

	"github.com/google/uuid"
	"github.com/logistic/pkg/cache"
)

const (
	ttlVehicle = 10 * time.Minute

	geoKeyOnline = "geo:online"
)

type vehicleRepoImpl struct {
	client *ent.Client
	cache  *cache.Client
	mapper mapper.AppMapper
}

var _ app.VehicleRepo = (*vehicleRepoImpl)(nil)

func NewVehicleRepo(client *ent.Client, redis *cache.Client, appMapper mapper.AppMapper) app.VehicleRepo {
	return &vehicleRepoImpl{client: client, cache: redis, mapper: appMapper}
}

func (r *vehicleRepoImpl) keyVehicle(id uuid.UUID) string { return r.cache.Key("vehicle", id.String()) }
func (r *vehicleRepoImpl) keyGeo() string                 { return r.cache.Key(geoKeyOnline) }

func (r *vehicleRepoImpl) invalidateVehicle(ctx context.Context, id uuid.UUID) {
	if r.cache == nil {
		return
	}
	if err := r.cache.Delete(ctx, r.keyVehicle(id)); err != nil {
		log.Printf("[repo] invalidate vehicle %s failed: %v", id, err)
	}
}

func (r *vehicleRepoImpl) CreateVehicle(ctx context.Context, param *entity.RegisterVehicleParam) (*entity.Vehicle, error) {
	builder := r.client.Vehicle.Create().
		SetDriverID(param.DriverID).
		SetLicensePlate(param.LicensePlate).
		SetVehicleType(vehicle.VehicleType(param.VehicleType)).
		SetCapacityWeightKg(param.CapacityWeightKg).
		SetCapacityVolumeCbm(param.CapacityVolumeCbm)

	if param.Brand != "" {
		builder = builder.SetBrand(param.Brand)
	}
	if param.Model != "" {
		builder = builder.SetModel(param.Model)
	}
	if param.ManufactureYear > 0 {
		builder = builder.SetManufactureYear(param.ManufactureYear)
	}

	dao, err := builder.Save(ctx)
	if err != nil {
		return nil, wrapError(err, entity.ErrVehicleNotFound)
	}

	e := r.mapper.EntVehicleToEntityVehicle(dao)
	return &e, nil
}

func (r *vehicleRepoImpl) GetVehicleByID(ctx context.Context, id uuid.UUID) (*entity.Vehicle, error) {
	key := r.keyVehicle(id)

	var cached entity.Vehicle
	if r.cache != nil {
		if err := r.cache.GetJSON(ctx, key, &cached); err == nil {
			return &cached, nil
		}
	}

	dao, err := r.client.Vehicle.Query().Where(vehicle.IDEQ(id)).Only(ctx)
	if err != nil {
		return nil, wrapError(err, entity.ErrVehicleNotFound)
	}

	e := r.mapper.EntVehicleToEntityVehicle(dao)
	if r.cache != nil {
		_ = r.cache.SetJSON(ctx, key, e, ttlVehicle)
	}
	return &e, nil
}

func (r *vehicleRepoImpl) GetVehiclesByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]entity.Vehicle, error) {
	result := make(map[uuid.UUID]entity.Vehicle, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	daos, err := r.client.Vehicle.Query().Where(vehicle.IDIn(ids...)).All(ctx)
	if err != nil {
		return nil, wrapError(err, entity.ErrVehicleNotFound)
	}

	for _, e := range r.mapper.EntVehicleListToEntityVehicleList(daos) {
		result[e.ID] = e
	}
	return result, nil
}

func (r *vehicleRepoImpl) ListVehicles(ctx context.Context, param *entity.ListVehiclesParam) ([]entity.Vehicle, int64, error) {
	_, pageSize, offset := entity.NormalizePaging(param.Page, param.PageSize)

	q := r.client.Vehicle.Query()
	if param.DriverID != uuid.Nil {
		q = q.Where(vehicle.DriverIDEQ(param.DriverID))
	}
	if param.Status != "" {
		q = q.Where(vehicle.StatusEQ(vehicle.Status(param.Status)))
	}
	if param.VehicleType != "" {
		q = q.Where(vehicle.VehicleTypeEQ(vehicle.VehicleType(param.VehicleType)))
	}

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, wrapError(err, entity.ErrVehicleNotFound)
	}

	daos, err := q.Order(ent.Desc(vehicle.FieldCreatedAt)).Offset(offset).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, wrapError(err, entity.ErrVehicleNotFound)
	}

	return r.mapper.EntVehicleListToEntityVehicleList(daos), int64(total), nil
}

func (r *vehicleRepoImpl) AdminListVehicles(ctx context.Context, param *entity.AdminListVehiclesParam) ([]entity.Vehicle, int64, error) {
	_, pageSize, offset := entity.NormalizePaging(param.Page, param.PageSize)

	q := r.client.Vehicle.Query()
	if param.Status != "" {
		q = q.Where(vehicle.StatusEQ(vehicle.Status(param.Status)))
	}
	if param.VerificationStatus != "" {
		q = q.Where(vehicle.VerificationStatusEQ(vehicle.VerificationStatus(param.VerificationStatus)))
	}
	if param.VehicleType != "" {
		q = q.Where(vehicle.VehicleTypeEQ(vehicle.VehicleType(param.VehicleType)))
	}
	if kw := strings.TrimSpace(param.Keyword); kw != "" {
		q = q.Where(vehicle.LicensePlateContainsFold(kw))
	}

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, wrapError(err, entity.ErrVehicleNotFound)
	}

	daos, err := q.Order(ent.Desc(vehicle.FieldCreatedAt)).Offset(offset).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, wrapError(err, entity.ErrVehicleNotFound)
	}

	return r.mapper.EntVehicleListToEntityVehicleList(daos), int64(total), nil
}

func (r *vehicleRepoImpl) UpdateVehicle(ctx context.Context, param *entity.UpdateVehicleParam) (*entity.Vehicle, error) {
	builder := r.client.Vehicle.UpdateOneID(param.ID)

	if param.Brand != "" {
		builder = builder.SetBrand(param.Brand)
	}
	if param.Model != "" {
		builder = builder.SetModel(param.Model)
	}
	if param.ManufactureYear > 0 {
		builder = builder.SetManufactureYear(param.ManufactureYear)
	}
	if param.VehicleType != "" {
		builder = builder.SetVehicleType(vehicle.VehicleType(param.VehicleType))
	}
	if param.CapacityWeightKg > 0 {
		builder = builder.SetCapacityWeightKg(param.CapacityWeightKg)
	}
	if param.CapacityVolumeCbm > 0 {
		builder = builder.SetCapacityVolumeCbm(param.CapacityVolumeCbm)
	}

	dao, err := builder.Save(ctx)
	if err != nil {
		return nil, wrapError(err, entity.ErrVehicleNotFound)
	}

	r.invalidateVehicle(ctx, param.ID)
	e := r.mapper.EntVehicleToEntityVehicle(dao)
	return &e, nil
}

func (r *vehicleRepoImpl) UpdateVehicleStatus(ctx context.Context, id uuid.UUID, status string) (*entity.Vehicle, error) {
	dao, err := r.client.Vehicle.UpdateOneID(id).
		SetStatus(vehicle.Status(status)).
		Save(ctx)
	if err != nil {
		return nil, wrapError(err, entity.ErrVehicleNotFound)
	}

	r.invalidateVehicle(ctx, id)

	if status != entity.VehicleStatusActive && r.cache != nil {
		if err := r.cache.GeoRemove(ctx, r.keyGeo(), id.String()); err != nil {
			log.Printf("[repo] gỡ xe %s khỏi geo index thất bại: %v", id, err)
		}
	}

	e := r.mapper.EntVehicleToEntityVehicle(dao)
	return &e, nil
}

func (r *vehicleRepoImpl) UpdateVerification(ctx context.Context, param *entity.VerifyVehicleParam, status string) (*entity.Vehicle, error) {
	builder := r.client.Vehicle.UpdateOneID(param.ID).
		SetVerificationStatus(vehicle.VerificationStatus(status)).
		SetVerificationNote(param.Note).
		SetVerifiedAt(time.Now())

	if param.ReviewerID != uuid.Nil {
		builder = builder.SetVerifiedBy(param.ReviewerID)
	}

	dao, err := builder.Save(ctx)
	if err != nil {
		return nil, wrapError(err, entity.ErrVehicleNotFound)
	}

	r.invalidateVehicle(ctx, param.ID)

	if status == entity.VerificationRejected && r.cache != nil {
		_ = r.cache.GeoRemove(ctx, r.keyGeo(), param.ID.String())
	}

	e := r.mapper.EntVehicleToEntityVehicle(dao)
	return &e, nil
}

func (r *vehicleRepoImpl) DeleteVehicle(ctx context.Context, id uuid.UUID) error {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return wrapError(err, nil)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.VehicleDocument.Delete().Where(vehicledocument.VehicleIDEQ(id)).Exec(ctx); err != nil {
		return wrapError(err, entity.ErrDocumentNotFound)
	}
	if _, err := tx.VehicleLocation.Delete().Where(vehiclelocation.VehicleIDEQ(id)).Exec(ctx); err != nil {
		return wrapError(err, entity.ErrLocationNotFound)
	}
	if _, err := tx.DriverAvailability.Delete().Where(driveravailability.VehicleIDEQ(id)).Exec(ctx); err != nil {
		return wrapError(err, entity.ErrAvailabilityNotFound)
	}
	if err := tx.Vehicle.DeleteOneID(id).Exec(ctx); err != nil {
		return wrapError(err, entity.ErrVehicleNotFound)
	}

	if err := tx.Commit(); err != nil {
		return wrapError(err, nil)
	}

	r.invalidateVehicle(ctx, id)
	if r.cache != nil {
		_ = r.cache.GeoRemove(ctx, r.keyGeo(), id.String())
	}
	return nil
}

func (r *vehicleRepoImpl) CountVehicles(ctx context.Context, status, verificationStatus string) (int64, error) {
	q := r.client.Vehicle.Query()
	if status != "" {
		q = q.Where(vehicle.StatusEQ(vehicle.Status(status)))
	}
	if verificationStatus != "" {
		q = q.Where(vehicle.VerificationStatusEQ(vehicle.VerificationStatus(verificationStatus)))
	}
	n, err := q.Count(ctx)
	if err != nil {
		return 0, wrapError(err, entity.ErrVehicleNotFound)
	}
	return int64(n), nil
}

func (r *vehicleRepoImpl) CreateDocument(ctx context.Context, param *entity.UploadDocumentParam) (*entity.VehicleDocument, error) {
	builder := r.client.VehicleDocument.Create().
		SetVehicleID(param.VehicleID).
		SetDocumentType(vehicledocument.DocumentType(param.DocumentType)).
		SetDocumentNumber(param.DocumentNumber).
		SetFileURL(param.FileURL)

	if !param.IssuedAt.IsZero() {
		builder = builder.SetIssuedAt(param.IssuedAt)
	}
	if !param.ExpiresAt.IsZero() {
		builder = builder.SetExpiresAt(param.ExpiresAt)
	}

	dao, err := builder.Save(ctx)
	if err != nil {
		return nil, wrapError(err, entity.ErrDocumentNotFound)
	}

	e := r.mapper.EntDocumentToEntityDocument(dao)
	return &e, nil
}

func (r *vehicleRepoImpl) GetDocument(ctx context.Context, id uuid.UUID) (*entity.VehicleDocument, error) {
	dao, err := r.client.VehicleDocument.Get(ctx, id)
	if err != nil {
		return nil, wrapError(err, entity.ErrDocumentNotFound)
	}
	e := r.mapper.EntDocumentToEntityDocument(dao)
	return &e, nil
}

func (r *vehicleRepoImpl) ListDocuments(ctx context.Context, param *entity.ListDocumentsParam) ([]entity.VehicleDocument, error) {
	q := r.client.VehicleDocument.Query().Where(vehicledocument.VehicleIDEQ(param.VehicleID))
	if param.ReviewStatus != "" {
		q = q.Where(vehicledocument.ReviewStatusEQ(vehicledocument.ReviewStatus(param.ReviewStatus)))
	}

	daos, err := q.Order(ent.Desc(vehicledocument.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, wrapError(err, entity.ErrDocumentNotFound)
	}
	return r.mapper.EntDocumentListToEntityList(daos), nil
}

func (r *vehicleRepoImpl) ListPendingDocuments(ctx context.Context, page, pageSize int) ([]entity.VehicleDocument, int64, error) {
	_, pageSize, offset := entity.NormalizePaging(page, pageSize)

	q := r.client.VehicleDocument.Query().
		Where(vehicledocument.ReviewStatusEQ(vehicledocument.ReviewStatusPending))

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, wrapError(err, entity.ErrDocumentNotFound)
	}

	daos, err := q.Order(ent.Asc(vehicledocument.FieldCreatedAt)).Offset(offset).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, wrapError(err, entity.ErrDocumentNotFound)
	}

	return r.mapper.EntDocumentListToEntityList(daos), int64(total), nil
}

func (r *vehicleRepoImpl) ReviewDocument(ctx context.Context, param *entity.ReviewDocumentParam, status string) (*entity.VehicleDocument, error) {
	builder := r.client.VehicleDocument.UpdateOneID(param.ID).
		SetReviewStatus(vehicledocument.ReviewStatus(status)).
		SetReviewNote(param.Note).
		SetReviewedAt(time.Now())

	if param.ReviewerID != uuid.Nil {
		builder = builder.SetReviewedBy(param.ReviewerID)
	}

	dao, err := builder.Save(ctx)
	if err != nil {
		return nil, wrapError(err, entity.ErrDocumentNotFound)
	}

	e := r.mapper.EntDocumentToEntityDocument(dao)
	return &e, nil
}

func (r *vehicleRepoImpl) DeleteDocument(ctx context.Context, id uuid.UUID) error {
	if err := r.client.VehicleDocument.DeleteOneID(id).Exec(ctx); err != nil {
		return wrapError(err, entity.ErrDocumentNotFound)
	}
	return nil
}

func (r *vehicleRepoImpl) CountPendingDocuments(ctx context.Context) (int64, error) {
	n, err := r.client.VehicleDocument.Query().
		Where(vehicledocument.ReviewStatusEQ(vehicledocument.ReviewStatusPending)).
		Count(ctx)
	if err != nil {
		return 0, wrapError(err, entity.ErrDocumentNotFound)
	}
	return int64(n), nil
}

func (r *vehicleRepoImpl) UpsertLocation(ctx context.Context, param *entity.ReportLocationParam, zoneID string) (*entity.VehicleLocation, error) {
	existing, err := r.client.VehicleLocation.Query().
		Where(vehiclelocation.VehicleIDEQ(param.VehicleID)).
		Only(ctx)

	var dao *ent.VehicleLocation
	switch {
	case err == nil:
		dao, err = existing.Update().
			SetDriverID(param.DriverID).
			SetLatitude(param.Latitude).
			SetLongitude(param.Longitude).
			SetHeading(param.Heading).
			SetSpeedKph(param.SpeedKph).
			SetZoneID(zoneID).
			Save(ctx)
		if err != nil {
			return nil, wrapError(err, entity.ErrLocationNotFound)
		}

	case ent.IsNotFound(err):
		dao, err = r.client.VehicleLocation.Create().
			SetVehicleID(param.VehicleID).
			SetDriverID(param.DriverID).
			SetLatitude(param.Latitude).
			SetLongitude(param.Longitude).
			SetHeading(param.Heading).
			SetSpeedKph(param.SpeedKph).
			SetZoneID(zoneID).
			Save(ctx)
		if err != nil {
			return nil, wrapError(err, entity.ErrLocationNotFound)
		}

	default:
		return nil, wrapError(err, entity.ErrLocationNotFound)
	}

	if _, err := r.client.DriverAvailability.Update().
		Where(driveravailability.VehicleIDEQ(param.VehicleID)).
		SetCurrentLat(param.Latitude).
		SetCurrentLng(param.Longitude).
		SetZoneID(zoneID).
		Save(ctx); err != nil {
		log.Printf("[repo] đồng bộ toạ độ sang availability thất bại (%s): %v", param.VehicleID, err)
	}

	r.syncGeoIndex(ctx, param.VehicleID, param.Latitude, param.Longitude)

	e := r.mapper.EntLocationToEntityLocation(dao)
	return &e, nil
}

func (r *vehicleRepoImpl) syncGeoIndex(ctx context.Context, vehicleID uuid.UUID, lat, lng float64) {
	if r.cache == nil {
		return
	}

	avail, err := r.client.DriverAvailability.Query().
		Where(driveravailability.VehicleIDEQ(vehicleID)).
		Only(ctx)
	if err != nil || !avail.IsOnline {
		return
	}

	if err := r.cache.GeoAdd(ctx, r.keyGeo(), cache.GeoMember{
		Name:      vehicleID.String(),
		Latitude:  lat,
		Longitude: lng,
	}); err != nil {
		log.Printf("[repo] cập nhật geo index cho xe %s thất bại: %v", vehicleID, err)
	}
}

func (r *vehicleRepoImpl) GetLocation(ctx context.Context, vehicleID uuid.UUID) (*entity.VehicleLocation, error) {
	dao, err := r.client.VehicleLocation.Query().
		Where(vehiclelocation.VehicleIDEQ(vehicleID)).
		Only(ctx)
	if err != nil {
		return nil, wrapError(err, entity.ErrLocationNotFound)
	}
	e := r.mapper.EntLocationToEntityLocation(dao)
	return &e, nil
}

func (r *vehicleRepoImpl) UpsertAvailability(ctx context.Context, param *entity.SetAvailabilityParam, zoneID string) (*entity.DriverAvailability, error) {
	existing, err := r.client.DriverAvailability.Query().
		Where(driveravailability.DriverIDEQ(param.DriverID)).
		Only(ctx)

	var dao *ent.DriverAvailability
	switch {
	case err == nil:
		dao, err = existing.Update().
			SetVehicleID(param.VehicleID).
			SetIsOnline(param.IsOnline).
			SetAvailableWeightKg(param.AvailableWeightKg).
			SetAvailableVolumeCbm(param.AvailableVolumeCbm).
			SetCurrentLat(param.CurrentLat).
			SetCurrentLng(param.CurrentLng).
			SetZoneID(zoneID).
			Save(ctx)
		if err != nil {
			return nil, wrapError(err, entity.ErrAvailabilityNotFound)
		}

	case ent.IsNotFound(err):
		dao, err = r.client.DriverAvailability.Create().
			SetDriverID(param.DriverID).
			SetVehicleID(param.VehicleID).
			SetIsOnline(param.IsOnline).
			SetAvailableWeightKg(param.AvailableWeightKg).
			SetAvailableVolumeCbm(param.AvailableVolumeCbm).
			SetCurrentLat(param.CurrentLat).
			SetCurrentLng(param.CurrentLng).
			SetZoneID(zoneID).
			Save(ctx)
		if err != nil {
			return nil, wrapError(err, entity.ErrAvailabilityNotFound)
		}

	default:
		return nil, wrapError(err, entity.ErrAvailabilityNotFound)
	}

	if r.cache != nil {
		if param.IsOnline && entity.IsValidCoordinate(param.CurrentLat, param.CurrentLng) {
			if err := r.cache.GeoAdd(ctx, r.keyGeo(), cache.GeoMember{
				Name:      param.VehicleID.String(),
				Latitude:  param.CurrentLat,
				Longitude: param.CurrentLng,
			}); err != nil {
				log.Printf("[repo] thêm xe %s vào geo index thất bại: %v", param.VehicleID, err)
			}
		} else {
			if err := r.cache.GeoRemove(ctx, r.keyGeo(), param.VehicleID.String()); err != nil {
				log.Printf("[repo] gỡ xe %s khỏi geo index thất bại: %v", param.VehicleID, err)
			}
		}
	}

	e := r.mapper.EntAvailabilityToEntity(dao)
	return &e, nil
}

func (r *vehicleRepoImpl) GetAvailability(ctx context.Context, driverID uuid.UUID) (*entity.DriverAvailability, error) {
	dao, err := r.client.DriverAvailability.Query().
		Where(driveravailability.DriverIDEQ(driverID)).
		Only(ctx)
	if err != nil {
		return nil, wrapError(err, entity.ErrAvailabilityNotFound)
	}
	e := r.mapper.EntAvailabilityToEntity(dao)
	return &e, nil
}

func (r *vehicleRepoImpl) GetAvailabilitiesByVehicleIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]entity.DriverAvailability, error) {
	result := make(map[uuid.UUID]entity.DriverAvailability, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	daos, err := r.client.DriverAvailability.Query().
		Where(driveravailability.VehicleIDIn(ids...)).
		All(ctx)
	if err != nil {
		return nil, wrapError(err, entity.ErrAvailabilityNotFound)
	}

	for _, e := range r.mapper.EntAvailabilityListToEntityList(daos) {
		result[e.VehicleID] = e
	}
	return result, nil
}

func (r *vehicleRepoImpl) CountOnlineDrivers(ctx context.Context) (int64, error) {
	n, err := r.client.DriverAvailability.Query().
		Where(driveravailability.IsOnlineEQ(true)).
		Count(ctx)
	if err != nil {
		return 0, wrapError(err, entity.ErrAvailabilityNotFound)
	}
	return int64(n), nil
}

func (r *vehicleRepoImpl) SearchNearby(ctx context.Context, param *entity.SearchNearbyParam) ([]entity.NearbyVehicle, error) {
	param.Normalize()

	if r.cache == nil {
		return r.searchNearbyFallback(ctx, param)
	}

	hits, err := r.cache.GeoSearch(ctx, r.keyGeo(), param.Latitude, param.Longitude, param.RadiusKm, param.Limit*3)
	if err != nil {
		log.Printf("[repo] GEOSEARCH lỗi (%v) — chuyển sang quét Postgres", err)
		return r.searchNearbyFallback(ctx, param)
	}
	if len(hits) == 0 {
		return []entity.NearbyVehicle{}, nil
	}

	ids := make([]uuid.UUID, 0, len(hits))
	distanceByID := make(map[uuid.UUID]float64, len(hits))
	for _, h := range hits {
		id, pErr := uuid.Parse(h.Name)
		if pErr != nil {
			_ = r.cache.GeoRemove(ctx, r.keyGeo(), h.Name)
			continue
		}
		ids = append(ids, id)
		distanceByID[id] = h.DistanceKm
	}

	vehicles, err := r.GetVehiclesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	availabilities, err := r.GetAvailabilitiesByVehicleIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	results := make([]entity.NearbyVehicle, 0, param.Limit)
	for _, h := range hits {
		id, pErr := uuid.Parse(h.Name)
		if pErr != nil {
			continue
		}

		v, ok := vehicles[id]
		if !ok {
			_ = r.cache.GeoRemove(ctx, r.keyGeo(), h.Name)
			continue
		}
		avail, ok := availabilities[id]
		if !ok || !avail.IsOnline {
			continue
		}

		if !matchesRequirement(v, avail, param) {
			continue
		}

		results = append(results, entity.NearbyVehicle{
			VehicleID:          v.ID,
			DriverID:           v.DriverID,
			LicensePlate:       v.LicensePlate,
			VehicleType:        v.VehicleType,
			DistanceKm:         h.DistanceKm,
			AvailableWeightKg:  avail.AvailableWeightKg,
			AvailableVolumeCbm: avail.AvailableVolumeCbm,
			Latitude:           h.Latitude,
			Longitude:          h.Longitude,
		})

		if len(results) >= param.Limit {
			break
		}
	}

	return results, nil
}

func matchesRequirement(v entity.Vehicle, avail entity.DriverAvailability, param *entity.SearchNearbyParam) bool {
	if v.Status != entity.VehicleStatusActive {
		return false
	}

	if v.VerificationStatus != entity.VerificationVerified {
		return false
	}
	if param.VehicleType != "" && v.VehicleType != param.VehicleType {
		return false
	}
	if param.MinWeightKg > 0 && avail.AvailableWeightKg < param.MinWeightKg {
		return false
	}
	if param.MinVolumeCbm > 0 && avail.AvailableVolumeCbm < param.MinVolumeCbm {
		return false
	}
	return true
}

func (r *vehicleRepoImpl) searchNearbyFallback(ctx context.Context, param *entity.SearchNearbyParam) ([]entity.NearbyVehicle, error) {
	zones := neighborZones(param.Latitude, param.Longitude, param.RadiusKm)

	q := r.client.DriverAvailability.Query().Where(driveravailability.IsOnlineEQ(true))
	if len(zones) > 0 {
		q = q.Where(driveravailability.ZoneIDIn(zones...))
	}

	daos, err := q.Limit(param.Limit * 5).All(ctx)
	if err != nil {
		return nil, wrapError(err, entity.ErrAvailabilityNotFound)
	}

	availabilities := r.mapper.EntAvailabilityListToEntityList(daos)
	if len(availabilities) == 0 {
		return []entity.NearbyVehicle{}, nil
	}

	ids := make([]uuid.UUID, 0, len(availabilities))
	for _, a := range availabilities {
		ids = append(ids, a.VehicleID)
	}

	vehicles, err := r.GetVehiclesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	results := make([]entity.NearbyVehicle, 0, param.Limit)
	for _, a := range availabilities {
		v, ok := vehicles[a.VehicleID]
		if !ok || !matchesRequirement(v, a, param) {
			continue
		}

		dist := haversineKm(param.Latitude, param.Longitude, a.CurrentLat, a.CurrentLng)
		if dist > param.RadiusKm {
			continue
		}

		results = append(results, entity.NearbyVehicle{
			VehicleID:          v.ID,
			DriverID:           v.DriverID,
			LicensePlate:       v.LicensePlate,
			VehicleType:        v.VehicleType,
			DistanceKm:         dist,
			AvailableWeightKg:  a.AvailableWeightKg,
			AvailableVolumeCbm: a.AvailableVolumeCbm,
			Latitude:           a.CurrentLat,
			Longitude:          a.CurrentLng,
		})
	}

	sortByDistance(results)
	if len(results) > param.Limit {
		results = results[:param.Limit]
	}
	return results, nil
}