package app

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"matching_service/internal/entity"

	"github.com/google/uuid"
)

type matchingEngineImpl struct {
	repo         MatchingRepo
	spatial      SpatialEngine
	walletClient WalletClient
	matchChan    chan *entity.MatchContract
	kafkaPub     EventPublisher
	natsPub      EventPublisher

	notifier Notifier
	weights  ScoreWeights
	mu       sync.RWMutex
}

var _ MatchingEngine = (*matchingEngineImpl)(nil)

func NewMatchingEngine(
	repo MatchingRepo,
	spatial SpatialEngine,
	walletClient WalletClient,
	kafkaPub EventPublisher,
	natsPub EventPublisher,
	notifier Notifier,
) MatchingEngine {
	if notifier == nil {
		notifier = NoopNotifier{}
	}

	return &matchingEngineImpl{
		weights:      DefaultScoreWeights(),
		repo:         repo,
		spatial:      spatial,
		walletClient: walletClient,
		kafkaPub:     kafkaPub,
		natsPub:      natsPub,
		notifier:     notifier,
		matchChan:    make(chan *entity.MatchContract, 1000),
	}
}

func (e *matchingEngineImpl) broadcastBidToDrivers(ctx context.Context, bid *entity.Bid) {
	if bid == nil {
		return
	}

	nearbyAsks, err := e.repo.FindAskForBid(ctx, bid)
	if err != nil {
		log.Printf("[BROADCAST] tìm tài xế gần đơn %s thất bại: %v", bid.ID, err)
		return
	}
	if len(nearbyAsks) == 0 {
		return
	}

	for _, ask := range nearbyAsks {
		topic := fmt.Sprintf("matching.drivers.nearby.%s", ask.DriverID.String())
		e.natsPub.Publish(ctx, &EventMessage{
			Topic:   topic,
			Key:     bid.ID.String(),
			Payload: *bid,
		})
	}

	if err := e.notifier.NotifyDriverCandidates(ctx, bid, nearbyAsks); err != nil {
		log.Printf("[NOTIFY] gửi thông báo ứng viên tài xế cho Bid %s thất bại: %v", bid.ID, err)
	}

	log.Printf("[BROADCAST] Bid %s gửi tới %d tài xế (NATS + Notifier)", bid.ID, len(nearbyAsks))
}

func (e *matchingEngineImpl) suggestBidsToDriver(ctx context.Context, ask *entity.Ask) {
	if ask == nil {
		return
	}

	bids, err := e.repo.FindBidForAsk(ctx, ask)
	if err != nil {
		log.Printf("[SUGGESTION] tìm đơn cho Ask %s thất bại: %v", ask.ID, err)
		return
	}
	if len(bids) == 0 {
		return
	}

	e.mu.RLock()
	weights := e.weights
	e.mu.RUnlock()

	ranked := RankBidsForAsk(ask, bids, weights)
	if len(ranked) == 0 {
		return
	}

	suggestedBids := make([]entity.Bid, len(ranked))
	for i, r := range ranked {
		suggestedBids[i] = r.Bid
	}

	topic := fmt.Sprintf("matching.drivers.suggestions.%s", ask.DriverID.String())
	e.natsPub.Publish(ctx, &EventMessage{
		Topic:   topic,
		Key:     ask.ID.String(),
		Payload: suggestedBids,
	})

	if err := e.notifier.NotifyCargoSuggested(ctx, ask, suggestedBids); err != nil {
		log.Printf("[NOTIFY] gửi thông báo gợi ý đơn cho Ask %s thất bại: %v", ask.ID, err)
	}

	log.Printf("[SUGGESTION] Ask %s: chọn %d/%d đơn, điểm cao nhất %.4f",
		ask.ID, len(ranked), len(bids), ranked[0].Score)
}

func (e *matchingEngineImpl) SubmitBid(ctx context.Context, bid *entity.Bid) (*entity.Bid, error) {
	if bid == nil {
		return nil, fmt.Errorf("%w: %v", entity.ErrInvalidID, "bid is nil")
	}

	zoneID, err := e.spatial.GetZoneId(ctx, bid.Origin.Latitude, bid.Origin.Longitude)
	if err != nil {
		return nil, err
	}
	bid.Origin.ZoneID = zoneID
	bid.Status = entity.BidStatusPending

	if bid.ID == uuid.Nil {
		bid.ID = uuid.Must(uuid.NewV7())
	}

	err = e.repo.CreateBid(ctx, bid)
	if err != nil {
		return nil, err
	}

	e.broadcastBidToDrivers(ctx, bid)

	return bid, nil
}

func (e *matchingEngineImpl) SubmitAsk(ctx context.Context, ask *entity.Ask) (*entity.Ask, error) {
	if ask == nil {
		return nil, fmt.Errorf("%w: %v", entity.ErrInvalidID, "ask is nil")
	}

	zoneID, err := e.spatial.GetZoneId(ctx, ask.CurrentLocation.Latitude, ask.CurrentLocation.Longitude)
	if err != nil {
		return nil, err
	}
	ask.CurrentLocation.ZoneID = zoneID
	ask.Status = entity.AskStatusPending

	if ask.ID == uuid.Nil {
		ask.ID = uuid.Must(uuid.NewV7())
	}

	err = e.repo.CreateAsk(ctx, ask)
	if err != nil {
		return nil, err
	}

	e.suggestBidsToDriver(ctx, ask)

	return ask, nil
}

func (e *matchingEngineImpl) MatchStream() <-chan *entity.MatchContract {
	return e.matchChan
}

func (e *matchingEngineImpl) SubmitOffer(ctx context.Context, bidID uuid.UUID, askID uuid.UUID, desiredPrice float64) error {
	ask, err := e.repo.GetAsk(ctx, askID)
	if err != nil {
		return fmt.Errorf("failed to retrieve ask: %w", err)
	}

	ask.MinPrice = desiredPrice
	topic := fmt.Sprintf("matching.offers.%s", bidID.String())

	err = e.natsPub.Publish(ctx, &EventMessage{
		Topic:   topic,
		Key:     ask.ID.String(),
		Payload: *ask,
	})
	if err != nil {
		return entity.ErrOfferQueueUnavailable.WithCause(err)
	}

	log.Printf("[OFFER SUBMITTED] Driver %s -> Bid %s. Price: %.2f", ask.DriverID, bidID, desiredPrice)
	return nil
}

func (e *matchingEngineImpl) ProcessOfferQueue(ctx context.Context, bidID uuid.UUID, offerAsk *entity.Ask) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	bid, err := e.repo.GetBid(ctx, bidID)
	if err != nil {
		return err
	}

	if bid.Status != entity.BidStatusPending {
		e.natsPub.Publish(ctx, &EventMessage{
			Topic:   fmt.Sprintf("matching.drivers.rejected.%s", offerAsk.DriverID.String()),
			Key:     bidID.String(),
			Payload: []byte("Bid is already under negotiation."),
		})
		log.Printf("[INSTANT RELEASE] Driver %s rejected for Bid %s (Already taken)", offerAsk.DriverID, bidID)
		return nil
	}

	// Giá báo chỉ tồn tại trong bản tin này; không ghi lại thì lúc chốt không còn nguồn nào biết đã thoả thuận bao nhiêu.
	bid.Status = entity.BidStatusNegotiating
	bid.OfferedPrice = offerAsk.MinPrice
	bid.OfferedAskID = offerAsk.ID
	if err := e.repo.UpdateBid(ctx, bid); err != nil {
		return err
	}

	e.natsPub.Publish(ctx, &EventMessage{
		Topic:   fmt.Sprintf("matching.shipper.offers_received.%s", bid.ShipperID.String()),
		Key:     offerAsk.ID.String(),
		Payload: *offerAsk,
	})

	if err := e.notifier.NotifyOfferReceived(ctx, bid, offerAsk, offerAsk.MinPrice); err != nil {
		log.Printf("[NOTIFY] gửi thông báo báo giá cho Bid %s thất bại: %v", bidID, err)
	}

	log.Printf("[OFFER FORWARDED] Driver %s sent to Shipper %s for Bid %s", offerAsk.DriverID, bid.ShipperID, bidID)

	return nil
}

func (e *matchingEngineImpl) RejectOffer(ctx context.Context, bidID uuid.UUID, askID uuid.UUID) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	bid, err := e.repo.GetBid(ctx, bidID)
	if err != nil {
		return err
	}

	if bid.Status != entity.BidStatusNegotiating {
		return entity.ErrBidNotNegotiating.WithDetail("status", bid.StatusString())
	}

	bid.Status = entity.BidStatusPending
	bid.OfferedPrice = 0
	bid.OfferedAskID = uuid.Nil
	if err := e.repo.UpdateBid(ctx, bid); err != nil {
		return err
	}

	ask, err := e.repo.GetAsk(ctx, askID)
	if err != nil {
		log.Printf("[OFFER REJECTED] Bid %s đã mở lại, nhưng không đọc được Ask %s: %v", bidID, askID, err)
		return nil
	}

	e.natsPub.Publish(ctx, &EventMessage{
		Topic:   fmt.Sprintf("matching.drivers.rejected.%s", ask.DriverID.String()),
		Key:     bidID.String(),
		Payload: []byte("Shipper has rejected your offer."),
	})

	if nErr := e.notifier.NotifyOfferRejected(ctx, bid, ask, ""); nErr != nil {
		log.Printf("[NOTIFY] gửi thông báo từ chối cho Ask %s thất bại: %v", askID, nErr)
	}

	log.Printf("[OFFER REJECTED] Shipper %s rejected Driver %s for Bid %s", bid.ShipperID, ask.DriverID, bidID)
	return nil
}

func (e *matchingEngineImpl) AcceptOffer(ctx context.Context, bidID uuid.UUID, askID uuid.UUID, consensusPrice float64, shipperSignature string) (*entity.MatchContract, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	bid, err := e.repo.GetBid(ctx, bidID)
	if err != nil {
		return nil, err
	}

	if bid.Status != entity.BidStatusNegotiating {
		return nil, entity.ErrBidNotNegotiating.WithDetail("status", bid.StatusString())
	}

	// Không có bước này thì chủ hàng chốt được với một tài xế chưa từng báo giá, và chốt ở mức giá sàn của người đó.
	if bid.OfferedAskID != askID {
		return nil, entity.ErrOfferAskMismatch.WithDetail("negotiating_ask_id", bid.OfferedAskID.String())
	}

	// Giá do server lưu là giá chuẩn; số client gửi lên chỉ để xác nhận.
	if consensusPrice > 0 && consensusPrice != bid.OfferedPrice {
		return nil, entity.ErrPriceMismatch.
			WithDetail("offered_price", strconv.FormatFloat(bid.OfferedPrice, 'f', -1, 64)).
			WithDetail("consensus_price", strconv.FormatFloat(consensusPrice, 'f', -1, 64))
	}

	ask, err := e.repo.GetAsk(ctx, askID)
	if err != nil {
		return nil, err
	}

	contract := &entity.MatchContract{
		ID:               uuid.Must(uuid.NewV7()),
		BidID:            bid.ID,
		AskID:            ask.ID,
		ConsensusPrice:   bid.OfferedPrice,
		ConsensusDeposit: bid.OfferedPrice * 0.1,
		Status:           entity.MatchStatusAccepted,
		AgreedAt:         time.Now(),
		ShipperSignature: shipperSignature,
		DriverSignature:  "",
		SystemSignature:  "",
	}

	balance, err := e.walletClient.CheckBalance(ctx, bid.ShipperID)
	if err != nil {
		return nil, entity.ErrWalletUnavailable.WithCause(err)
	}
	if balance < contract.ConsensusDeposit {
		return nil, fmt.Errorf("%w: shipper %s needs %.2f, has %.2f", entity.ErrInsufficientBalance, bid.ShipperID, contract.ConsensusDeposit, balance)
	}

	_ = e.kafkaPub.Publish(ctx, &EventMessage{
		Topic: "wallet.hold_deposit",
		Key:   contract.ID.String(),
		Payload: map[string]any{
			"driver_id":   bid.ShipperID.String(),
			"amount":      contract.ConsensusDeposit,
			"contract_id": contract.ID.String(),
		},
	})

	// Ghi hợp đồng trước rồi mới lật trạng thái: ngược lại thì lỗi ở bước này để đơn và chuyến MATCHED mà không có hợp đồng nào.
	if err := e.repo.CreateMatchContract(ctx, contract); err != nil {
		return nil, err
	}

	bid.Status = entity.BidStatusMatched
	ask.Status = entity.AskStatusMatched

	if err := e.repo.UpdateBid(ctx, bid); err != nil {
		return nil, err
	}
	if err := e.repo.UpdateAsk(ctx, ask); err != nil {
		return nil, err
	}

	select {
	case e.matchChan <- contract:
	default:
		log.Println("WARNING: matchChan is full!")
	}

	e.natsPub.Publish(ctx, &EventMessage{
		Topic:   fmt.Sprintf("matching.drivers.accepted.%s", ask.DriverID.String()),
		Key:     contract.ID.String(),
		Payload: *contract,
	})

	if nErr := e.notifier.NotifyMatchFound(ctx, contract, bid, ask); nErr != nil {
		log.Printf("[NOTIFY] gửi thông báo ghép đơn cho Contract %s thất bại: %v", contract.ID, nErr)
	}

	log.Printf("[DEAL CLOSED] MatchContract %s created! Shipper: %s, Driver: %s, Price: %.2f",
		contract.ID, bid.ShipperID, ask.DriverID, contract.ConsensusPrice)

	return contract, nil
}
