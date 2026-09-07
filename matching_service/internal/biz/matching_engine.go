package biz

import (
	"context"
	"fmt"
	"log"
	cerr "matching_service/internal/common/errors"
	"matching_service/internal/entity"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

type MatchingEngine interface {
	SubmitBid(ctx context.Context, bid *entity.Bid) (*entity.Bid, error)
	SubmitAsk(ctx context.Context, ask *entity.Ask) (*entity.Ask, error)
	SubmitOffer(ctx context.Context, bidID uuid.UUID, askID uuid.UUID, desiredPrice float64) error
	// Do consumer NATS gọi khi lấy báo giá khỏi hàng đợi, không phải controller.
	ProcessOfferQueue(ctx context.Context, bidID uuid.UUID, offerAsk *entity.Ask) error
	AcceptOffer(ctx context.Context, bidID uuid.UUID, askID uuid.UUID, consensusPrice float64, shipperSignature string) (*entity.MatchContract, error)
	RejectOffer(ctx context.Context, bidID uuid.UUID, askID uuid.UUID) error
	MatchStream() <-chan *entity.MatchContract
}

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

	asks, err := e.repo.FindAskForBid(ctx, bid)
	if err != nil {
		log.Printf("Failed to find asks for bid %s: %v", bid.ID, err)
		return
	}

	if len(asks) == 0 {
		log.Printf("No available drivers found for Bid %s", bid.ID)
		return
	}

	ranked := RankAsksForBid(bid, asks, e.weights)
	if len(ranked) == 0 {
		log.Printf("Bid %s: %d tài xế trong vùng nhưng không ai đạt ngưỡng điểm", bid.ID, len(asks))
		return
	}
	if len(ranked) > maxCandidatesPerBid {
		ranked = ranked[:maxCandidatesPerBid]
	}

	asks = asks[:0]
	for _, r := range ranked {
		asks = append(asks, r.Ask)
	}

	e.natsPub.Publish(ctx, &EventMessage{
		Topic:   "matching.drivers.notified",
		Key:     bid.ID.String(),
		Payload: ranked,
	})

	e.kafkaPub.Publish(ctx, &EventMessage{
		Topic:   "matching.bid_created",
		Key:     bid.ID.String(),
		Payload: *bid,
	})

	if err := e.notifier.NotifyDriverCandidates(ctx, bid, asks); err != nil {
		log.Printf("[NOTIFY] gửi thông báo ứng viên cho Bid %s thất bại: %v", bid.ID, err)
	}

	log.Printf("[BROADCAST] Bid %s: chọn %d/%d tài xế, điểm cao nhất %.4f",
		bid.ID, len(ranked), cap(asks), ranked[0].Score)
}

func (e *matchingEngineImpl) suggestBidsToDriver(ctx context.Context, ask *entity.Ask) {
	if ask == nil {
		return
	}

	bids, err := e.repo.FindBidForAsk(ctx, ask)
	if err != nil {
		log.Printf("Failed to find bids for ask %s: %v", ask.ID, err)
		return
	}

	if len(bids) == 0 {
		return
	}

	ranked := RankBidsForAsk(ask, bids, e.weights)
	if len(ranked) == 0 {
		log.Printf("Ask %s: %d đơn trong vùng nhưng không đơn nào đạt ngưỡng điểm", ask.ID, len(bids))
		return
	}
	if len(ranked) > maxSuggestionsPerAsk {
		ranked = ranked[:maxSuggestionsPerAsk]
	}

	suggested := make([]entity.Bid, 0, len(ranked))
	for _, r := range ranked {
		suggested = append(suggested, r.Bid)
	}

	{
		e.natsPub.Publish(ctx, &EventMessage{
			Topic:   fmt.Sprintf("matching.suggested_cargos.%s", ask.DriverID.String()),
			Key:     ask.ID.String(),
			Payload: ranked,
		})

		e.kafkaPub.Publish(ctx, &EventMessage{
			Topic:   "matching.ask_created",
			Key:     ask.ID.String(),
			Payload: *ask,
		})

		if err := e.notifier.NotifyCargoSuggested(ctx, ask, suggested); err != nil {
			log.Printf("[NOTIFY] gửi gợi ý đơn hàng cho Ask %s thất bại: %v", ask.ID, err)
		}

		log.Printf("[SUGGESTION] Ask %s: chọn %d/%d đơn, điểm cao nhất %.4f",
			ask.ID, len(ranked), len(bids), ranked[0].Score)
	}
}

func (e *matchingEngineImpl) SubmitBid(ctx context.Context, bid *entity.Bid) (*entity.Bid, error) {
	if bid == nil {
		return nil, fmt.Errorf("%w: %v", cerr.ErrInvalidID, "bid is nil")
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
		return nil, fmt.Errorf("%w: %v", cerr.ErrInvalidID, "ask is nil")
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
		return cerr.ErrOfferQueueUnavailable.WithCause(err)
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

	// Giá báo chỉ tồn tại trong bản tin này; không ghi lại thì lúc chốt không còn
	// nguồn nào biết đã thoả thuận bao nhiêu.
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
		return cerr.ErrBidNotNegotiating.WithDetail("status", bid.StatusString())
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
		return nil, cerr.ErrBidNotNegotiating.WithDetail("status", bid.StatusString())
	}

	// Không có bước này thì chủ hàng chốt được với một tài xế chưa từng báo giá,
	// và chốt ở mức giá sàn của người đó.
	if bid.OfferedAskID != askID {
		return nil, cerr.ErrOfferAskMismatch.WithDetail("negotiating_ask_id", bid.OfferedAskID.String())
	}

	// Giá do server lưu là giá chuẩn; số client gửi lên chỉ để xác nhận.
	if consensusPrice > 0 && consensusPrice != bid.OfferedPrice {
		return nil, cerr.ErrPriceMismatch.
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
		// Chưa có bước tài xế ký và ký số hệ thống; để rỗng thay vì bịa.
		DriverSignature: "",
		SystemSignature: "",
	}

	balance, err := e.walletClient.CheckBalance(ctx, bid.ShipperID)
	if err != nil {
		return nil, cerr.ErrWalletUnavailable.WithCause(err)
	}
	if balance < contract.ConsensusDeposit {
		return nil, fmt.Errorf("%w: shipper %s needs %.2f, has %.2f", cerr.ErrInsufficientBalance, bid.ShipperID, contract.ConsensusDeposit, balance)
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

	// Ghi hợp đồng trước rồi mới lật trạng thái: ngược lại thì lỗi ở bước này để
	// đơn và chuyến MATCHED mà không có hợp đồng nào.
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
