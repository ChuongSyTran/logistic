package repo

import (
	"context"
	"log"

	"matching_service/ent"
	"matching_service/ent/asks"
	"matching_service/ent/bids"
	"matching_service/internal/biz"
	"matching_service/internal/entity"
	"matching_service/internal/mapper"
	"matching_service/internal/mapper/generated"

	"github.com/google/uuid"
)

type matchingRepoImpl struct {
	masterClient *ent.Client
	slaveClient  *ent.Client
	mapper       mapper.MatchingMapper
}

var _ biz.MatchingRepo = (*matchingRepoImpl)(nil)

func NewMatchingRepo(masterClient *ent.Client, slaveClient *ent.Client, mapper *generated.MatchingMapperImpl) biz.MatchingRepo {
	if masterClient == nil || slaveClient == nil {
		log.Fatalf("[SYSTEM] failed to create master and slave client")
	}

	return &matchingRepoImpl{
		masterClient: masterClient,
		slaveClient:  slaveClient,
		mapper:       mapper,
	}
}

func (r *matchingRepoImpl) CreateBid(ctx context.Context, bid *entity.Bid) error {
	dao, err := r.masterClient.Bids.Create().
		SetShipperID(bid.ShipperID).
		SetShipperPhone(bid.ShipperPhone).
		SetShipperMail(bid.ShipperMail).
		SetConsigneeID(bid.ConsigneeID).
		SetConsigneePhone(bid.ConsigneePhone).
		SetConsigneeMail(bid.ConsigneeMail).
		SetOriginLat(bid.Origin.Latitude).
		SetOriginLng(bid.Origin.Longitude).
		SetDestinationLat(bid.Destination.Latitude).
		SetDestinationLng(bid.Destination.Longitude).
		SetZoneID(bid.Origin.ZoneID).
		SetVolumeM3(bid.VolumeM3).
		SetWeightKg(bid.WeightKg).
		SetNillableMaxPrice(&bid.MaxPrice).
		SetCargoValue(bid.CargoValue).
		SetRequiredDeposit(bid.RequiredDeposit).
		SetDesiredDeposit(bid.DesiredDeposit).
		SetOfferedPrice(bid.OfferedPrice).
		SetOfferedAskID(bid.OfferedAskID).
		SetStatus(bid.Status).
		SetExpiresAt(bid.ExpiresAt).
		Save(ctx)

	if err != nil {
		return wrapError(err)
	}
	entity := r.mapper.EntBidToEntityBid(dao)
	*bid = entity

	return nil
}

func (r *matchingRepoImpl) CreateAsk(ctx context.Context, ask *entity.Ask) error {
	dao, err := r.masterClient.Asks.Create().
		SetID(ask.ID).
		SetDriverID(ask.DriverID).
		SetDriverPhone(ask.DriverPhone).
		SetDriverMail(ask.DriverMail).
		SetVehicleID(ask.VehicleID).
		SetVehicleType(ask.VehicleType).
		SetOriginLat(ask.CurrentLocation.Latitude).
		SetOriginLng(ask.CurrentLocation.Longitude).
		SetDestinationLat(ask.Destination.Latitude).
		SetDestinationLng(ask.Destination.Longitude).
		SetZoneID(ask.CurrentLocation.ZoneID).
		SetRouteID(ask.RouteID).
		SetCapacityVolumeCbm(ask.CapacityVolumeCbm).
		SetAvailableVolumeM3(ask.AvailableVolumeM3).
		SetCapacityWeightKg(ask.CapacityWeightKg).
		SetAvailableWeightKg(ask.AvailableWeightKg).
		SetNillableMinPrice(&ask.MinPrice).
		SetDesiredDeposit(ask.DesiredDeposit).
		SetStatus(ask.Status).
		SetExpiresAt(ask.ExpiresAt).
		Save(ctx)

	if err != nil {
		return wrapError(err)
	}
	entity := r.mapper.EntAskToEntityAsk(dao)
	*ask = entity

	return nil
}

func (r *matchingRepoImpl) FindAskForBid(ctx context.Context, bid *entity.Bid) ([]entity.Ask, error) {
	daoList, err := r.slaveClient.Asks.Query().
		Where(asks.Status(entity.AskStatusPending)).
		Where(asks.AvailableVolumeM3GT(bid.VolumeM3)).
		Where(asks.AvailableWeightKgGT(bid.WeightKg)).
		Where(asks.MinPriceLTE(bid.MaxPrice)).
		Where(withinRadiusKm("origin_lat", "origin_lng", bid.Origin.Latitude, bid.Origin.Longitude, coarseFilterRadiusKm)).
		Order(asks.ByMinPrice()).
		All(ctx)

	if err != nil {
		return nil, err
	}

	return r.mapper.EntAskListToEntityAskList(daoList), nil
}

func (r *matchingRepoImpl) FindBidForAsk(ctx context.Context, ask *entity.Ask) ([]entity.Bid, error) {
	daoList, err := r.slaveClient.Bids.Query().
		Where(bids.Status(entity.BidStatusPending)).
		Where(withinRadiusKm("origin_lat", "origin_lng", ask.CurrentLocation.Latitude, ask.CurrentLocation.Longitude, coarseFilterRadiusKm)).
		Where(withinRadiusKm("destination_lat", "destination_lng", ask.Destination.Latitude, ask.Destination.Longitude, coarseFilterRadiusKm)).
		Order(bids.ByMaxPrice()).All(ctx)

	if err != nil {
		return nil, err
	}

	return r.mapper.EntBidListToEntityBidList(daoList), nil
}

func (r *matchingRepoImpl) UpdateAsk(ctx context.Context, ask *entity.Ask) error {
	dao, err := r.masterClient.Asks.
		UpdateOneID(ask.ID).
		SetDriverID(ask.DriverID).
		SetDriverPhone(ask.DriverPhone).
		SetDriverMail(ask.DriverMail).
		SetVehicleID(ask.VehicleID).
		SetVehicleType(ask.VehicleType).
		SetOriginLat(ask.CurrentLocation.Latitude).
		SetOriginLng(ask.CurrentLocation.Longitude).
		SetDestinationLat(ask.Destination.Latitude).
		SetDestinationLng(ask.Destination.Longitude).
		SetZoneID(ask.CurrentLocation.ZoneID).
		SetRouteID(ask.RouteID).
		SetCapacityVolumeCbm(ask.CapacityVolumeCbm).
		SetAvailableVolumeM3(ask.AvailableVolumeM3).
		SetCapacityWeightKg(ask.CapacityWeightKg).
		SetAvailableWeightKg(ask.AvailableWeightKg).
		SetNillableMinPrice(&ask.MinPrice).
		SetDesiredDeposit(ask.DesiredDeposit).
		SetStatus(ask.Status).
		SetExpiresAt(ask.ExpiresAt).
		Save(ctx)

	if err != nil {
		return wrapError(err)
	}
	entity := r.mapper.EntAskToEntityAsk(dao)
	*ask = entity

	return nil
}

func (r *matchingRepoImpl) UpdateBid(ctx context.Context, bid *entity.Bid) error {
	dao, err := r.masterClient.Bids.
		UpdateOneID(bid.ID).
		SetShipperID(bid.ShipperID).
		SetShipperPhone(bid.ShipperPhone).
		SetShipperMail(bid.ShipperMail).
		SetConsigneeID(bid.ConsigneeID).
		SetConsigneePhone(bid.ConsigneePhone).
		SetConsigneeMail(bid.ConsigneeMail).
		SetOriginLat(bid.Origin.Latitude).
		SetOriginLng(bid.Origin.Longitude).
		SetDestinationLat(bid.Destination.Latitude).
		SetDestinationLng(bid.Destination.Longitude).
		SetZoneID(bid.Origin.ZoneID).
		SetVolumeM3(bid.VolumeM3).
		SetWeightKg(bid.WeightKg).
		SetNillableMaxPrice(&bid.MaxPrice).
		SetCargoValue(bid.CargoValue).
		SetRequiredDeposit(bid.RequiredDeposit).
		SetDesiredDeposit(bid.DesiredDeposit).
		SetOfferedPrice(bid.OfferedPrice).
		SetOfferedAskID(bid.OfferedAskID).
		SetStatus(bid.Status).
		SetExpiresAt(bid.ExpiresAt).
		Save(ctx)

	if err != nil {
		return wrapError(err)
	}
	entity := r.mapper.EntBidToEntityBid(dao)
	*bid = entity

	return nil
}

func (r *matchingRepoImpl) DeleteBid(ctx context.Context, id uuid.UUID) error {
	return r.masterClient.Bids.DeleteOneID(id).Exec(ctx)
}

func (r *matchingRepoImpl) DeleteAsk(ctx context.Context, id uuid.UUID) error {
	return r.masterClient.Asks.DeleteOneID(id).Exec(ctx)
}

func (r *matchingRepoImpl) GetBid(ctx context.Context, id uuid.UUID) (*entity.Bid, error) {
	dao, err := r.slaveClient.Bids.Get(ctx, id)
	if err != nil {
		return nil, wrapError(err)
	}
	ent := r.mapper.EntBidToEntityBid(dao)
	return &ent, nil
}

func (r *matchingRepoImpl) GetAsk(ctx context.Context, id uuid.UUID) (*entity.Ask, error) {
	dao, err := r.slaveClient.Asks.Get(ctx, id)
	if err != nil {
		return nil, wrapError(err)
	}
	ent := r.mapper.EntAskToEntityAsk(dao)
	return &ent, nil
}

func (r *matchingRepoImpl) CreateMatchContract(ctx context.Context, contract *entity.MatchContract) error {
	// Schema Match bắt buộc agreed_price, agreed_at và ba trường chữ ký, không cái
	// nào có mặc định — thiếu một cái là ent trả ValidationError.
	_, err := r.masterClient.Match.Create().
		SetID(contract.ID).
		SetBidID(contract.BidID).
		SetAskID(contract.AskID).
		SetAgreedPrice(contract.ConsensusPrice).
		SetConsensusPrice(contract.ConsensusPrice).
		SetConsensusDeposit(contract.ConsensusDeposit).
		SetShipperSignature(contract.ShipperSignature).
		SetDriverSignature(contract.DriverSignature).
		SetSystemSignature(contract.SystemSignature).
		SetAgreedAt(contract.AgreedAt).
		SetStatus(int(contract.Status)).
		Save(ctx)

	if err != nil {
		return wrapError(err)
	}
	return nil
}
