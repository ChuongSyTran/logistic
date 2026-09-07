package app

import (
	"context"
	"errors"
	"testing"

	"vehicle_service/internal/entity"

	"github.com/google/uuid"
)

type stubRepo struct {
	VehicleRepo
	vehicle entity.Vehicle
	doc     entity.VehicleDocument
	wrote   bool
}

func (r *stubRepo) GetVehicleByID(context.Context, uuid.UUID) (*entity.Vehicle, error) {
	v := r.vehicle
	return &v, nil
}

func (r *stubRepo) GetDocument(context.Context, uuid.UUID) (*entity.VehicleDocument, error) {
	d := r.doc
	return &d, nil
}

func (r *stubRepo) UpdateVehicle(_ context.Context, _ *entity.UpdateVehicleParam) (*entity.Vehicle, error) {
	r.wrote = true
	v := r.vehicle
	return &v, nil
}

func (r *stubRepo) UpdateVehicleStatus(_ context.Context, _ uuid.UUID, _ string) (*entity.Vehicle, error) {
	r.wrote = true
	v := r.vehicle
	return &v, nil
}

func (r *stubRepo) DeleteVehicle(context.Context, uuid.UUID) error {
	r.wrote = true
	return nil
}

func (r *stubRepo) CreateDocument(_ context.Context, _ *entity.UploadDocumentParam) (*entity.VehicleDocument, error) {
	r.wrote = true
	d := r.doc
	return &d, nil
}

func (r *stubRepo) ListDocuments(_ context.Context, _ *entity.ListDocumentsParam) ([]entity.VehicleDocument, error) {
	r.wrote = true
	return []entity.VehicleDocument{r.doc}, nil
}

func (r *stubRepo) DeleteDocument(context.Context, uuid.UUID) error {
	r.wrote = true
	return nil
}

func (r *stubRepo) GetLocation(_ context.Context, _ uuid.UUID) (*entity.VehicleLocation, error) {
	r.wrote = true
	return &entity.VehicleLocation{VehicleID: r.vehicle.ID, DriverID: r.vehicle.DriverID}, nil
}

func (r *stubRepo) UpsertLocation(_ context.Context, _ *entity.ReportLocationParam, _ string) (*entity.VehicleLocation, error) {
	r.wrote = true
	return &entity.VehicleLocation{VehicleID: r.vehicle.ID, DriverID: r.vehicle.DriverID}, nil
}

func (r *stubRepo) UpsertAvailability(_ context.Context, _ *entity.SetAvailabilityParam, _ string) (*entity.DriverAvailability, error) {
	r.wrote = true
	return &entity.DriverAvailability{VehicleID: r.vehicle.ID, DriverID: r.vehicle.DriverID}, nil
}

var (
	chuXe   = uuid.MustParse("11111111-1111-4111-8111-111111111111")
	nguoiLa = uuid.MustParse("22222222-2222-4222-8222-222222222222")
	xeID    = uuid.MustParse("33333333-3333-4333-8333-333333333333")
	giayID  = uuid.MustParse("44444444-4444-4444-8444-444444444444")
)

func newStub() *stubRepo {
	return &stubRepo{
		vehicle: entity.Vehicle{
			ID:                 xeID,
			DriverID:           chuXe,
			LicensePlate:       "51C-999.88",
			VehicleType:        entity.VehicleTypeTruck,
			Status:             entity.VehicleStatusActive,
			VerificationStatus: entity.VerificationVerified,
			CapacityWeightKg:   8000,
			CapacityVolumeCbm:  30,
		},
		doc: entity.VehicleDocument{ID: giayID, VehicleID: xeID, DocumentType: entity.DocTypeRegistration},
	}
}

// Mỗi thao tác tài xế làm được trên một xe, tham số hoá theo người gọi.
var thaoTac = map[string]func(e VehicleEngine, caller uuid.UUID) error{
	"GetVehicle": func(e VehicleEngine, caller uuid.UUID) error {
		_, err := e.GetVehicle(context.Background(), xeID, caller)
		return err
	},
	"UpdateVehicle": func(e VehicleEngine, caller uuid.UUID) error {
		_, err := e.UpdateVehicle(context.Background(), &entity.UpdateVehicleParam{
			ID: xeID, DriverID: caller, VehicleType: entity.VehicleTypeVan,
		})
		return err
	},
	"UpdateVehicleStatus": func(e VehicleEngine, caller uuid.UUID) error {
		_, err := e.UpdateVehicleStatus(context.Background(), xeID, caller, entity.VehicleStatusInactive)
		return err
	},
	"DeleteVehicle": func(e VehicleEngine, caller uuid.UUID) error {
		return e.DeleteVehicle(context.Background(), xeID, caller)
	},
	"UploadDocument": func(e VehicleEngine, caller uuid.UUID) error {
		_, err := e.UploadDocument(context.Background(), &entity.UploadDocumentParam{
			VehicleID: xeID, DriverID: caller,
			DocumentType: entity.DocTypeInsurance, FileURL: "https://cdn.logistic.vn/x.pdf",
		})
		return err
	},
	"ListDocuments": func(e VehicleEngine, caller uuid.UUID) error {
		_, err := e.ListDocuments(context.Background(), &entity.ListDocumentsParam{
			VehicleID: xeID, DriverID: caller,
		})
		return err
	},
	"DeleteDocument": func(e VehicleEngine, caller uuid.UUID) error {
		return e.DeleteDocument(context.Background(), giayID, caller)
	},
	"GetLocation": func(e VehicleEngine, caller uuid.UUID) error {
		_, err := e.GetLocation(context.Background(), xeID, caller)
		return err
	},
	"ReportLocation": func(e VehicleEngine, caller uuid.UUID) error {
		_, err := e.ReportLocation(context.Background(), &entity.ReportLocationParam{
			VehicleID: xeID, DriverID: caller, Latitude: 10.7769, Longitude: 106.7009,
		})
		return err
	},
	"SetAvailability": func(e VehicleEngine, caller uuid.UUID) error {
		_, err := e.SetAvailability(context.Background(), &entity.SetAvailabilityParam{
			VehicleID: xeID, DriverID: caller, IsOnline: true,
			CurrentLat: 10.7769, CurrentLng: 106.7009,
		})
		return err
	},
}

func TestNguoiLaKhongThaoTacDuocTrenXeNguoiKhac(t *testing.T) {
	for ten, chay := range thaoTac {
		t.Run(ten, func(t *testing.T) {
			repo := newStub()
			err := chay(NewVehicleEngine(repo), nguoiLa)

			if !errors.Is(err, entity.ErrVehicleNotOwned) {
				t.Fatalf("phải bị từ chối với ErrVehicleNotOwned, nhận: %v", err)
			}
			if repo.wrote {
				t.Fatal("đã chạm tới repo dù người gọi không phải chủ xe")
			}
		})
	}
}

func TestChuXeVanThaoTacBinhThuong(t *testing.T) {
	for ten, chay := range thaoTac {
		t.Run(ten, func(t *testing.T) {
			if err := chay(NewVehicleEngine(newStub()), chuXe); err != nil {
				t.Fatalf("chủ xe phải làm được, nhận: %v", err)
			}
		})
	}
}

// Quy ước sẵn có từ DeleteVehicle: driverID rỗng = luồng quản trị, bỏ kiểm tra.
func TestDriverIDRongLaLuongQuanTri(t *testing.T) {
	for ten, chay := range thaoTac {
		if ten == "SetAvailability" || ten == "ReportLocation" {
			continue // hai cái này bắt buộc có driver_id, đã kiểm riêng ở tầng trên
		}
		t.Run(ten, func(t *testing.T) {
			if err := chay(NewVehicleEngine(newStub()), uuid.Nil); err != nil {
				t.Fatalf("luồng quản trị phải đi qua được, nhận: %v", err)
			}
		})
	}
}
