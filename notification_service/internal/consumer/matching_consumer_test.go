package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"notification_service/internal/entity"
	"notification_service/internal/repo"

	"github.com/google/uuid"
	"github.com/logistic/pkg/events"
	"github.com/logistic/pkg/mq"
)

type fakeEngine struct {
	dispatched []entity.CreateNotificationParam
	eventIDs   []string
	err        error
	callCount  int
}

func (f *fakeEngine) DispatchEvent(_ context.Context, eventID, _, _ string, params []entity.CreateNotificationParam) (int64, error) {
	f.callCount++
	if f.err != nil {
		return 0, f.err
	}
	f.eventIDs = append(f.eventIDs, eventID)
	f.dispatched = append(f.dispatched, params...)
	return int64(len(params)), nil
}

func (f *fakeEngine) List(context.Context, *entity.ListNotificationsParam) (*entity.ListNotificationsResult, error) {
	panic("không dùng trong test này")
}
func (f *fakeEngine) Get(context.Context, uuid.UUID, uuid.UUID) (*entity.Notification, error) {
	panic("không dùng trong test này")
}
func (f *fakeEngine) MarkAsRead(context.Context, uuid.UUID, uuid.UUID) (int64, error) {
	panic("không dùng trong test này")
}
func (f *fakeEngine) MarkAllAsRead(context.Context, uuid.UUID) (int64, error) {
	panic("không dùng trong test này")
}
func (f *fakeEngine) Delete(context.Context, uuid.UUID, uuid.UUID) error {
	panic("không dùng trong test này")
}
func (f *fakeEngine) GetUnreadCount(context.Context, uuid.UUID) (int64, error) {
	panic("không dùng trong test này")
}
func (f *fakeEngine) GetPreference(context.Context, uuid.UUID) (*entity.NotificationPreference, error) {
	panic("không dùng trong test này")
}
func (f *fakeEngine) UpdatePreference(context.Context, *entity.UpdatePreferenceParam) (*entity.NotificationPreference, error) {
	panic("không dùng trong test này")
}
func (f *fakeEngine) AdminSend(context.Context, *entity.SendNotificationParam) (*entity.SendNotificationResult, error) {
	panic("không dùng trong test này")
}
func (f *fakeEngine) AdminList(context.Context, *entity.AdminListNotificationsParam) (*entity.ListNotificationsResult, error) {
	panic("không dùng trong test này")
}
func (f *fakeEngine) AdminListTemplates(context.Context, *entity.ListTemplatesParam) ([]entity.NotificationTemplate, error) {
	panic("không dùng trong test này")
}
func (f *fakeEngine) AdminCreateTemplate(context.Context, *entity.CreateTemplateParam) (*entity.NotificationTemplate, error) {
	panic("không dùng trong test này")
}
func (f *fakeEngine) AdminUpdateTemplate(context.Context, *entity.UpdateTemplateParam) (*entity.NotificationTemplate, error) {
	panic("không dùng trong test này")
}
func (f *fakeEngine) AdminDeleteTemplate(context.Context, uuid.UUID) error {
	panic("không dùng trong test này")
}
func (f *fakeEngine) AdminGetStats(context.Context) (*entity.NotificationStats, error) {
	panic("không dùng trong test này")
}
func (f *fakeEngine) RenderFromTemplate(context.Context, string, string, string, map[string]string) (string, string, bool) {
	return "", "", false
}

func buildDelivery(t *testing.T, routingKey string, payload any) mq.Delivery {
	t.Helper()

	blob, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var data map[string]any
	if err := json.Unmarshal(blob, &data); err != nil {
		t.Fatalf("unmarshal payload về map: %v", err)
	}

	env := events.Envelope{
		EventID:    uuid.NewString(),
		EventType:  routingKey,
		OccurredAt: time.Now().UTC(),
		Source:     "matching_service",
		Data:       data,
	}

	body, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	return mq.Delivery{RoutingKey: routingKey, MessageID: env.EventID, Body: body}
}

func TestHandleDriverCandidatesFound(t *testing.T) {
	driverA := uuid.NewString()
	driverB := uuid.NewString()
	bidID := uuid.NewString()

	payload := events.DriverCandidatesFound{
		BidID:     bidID,
		ShipperID: uuid.NewString(),
		WeightKg:  1200,
		VolumeM3:  8.5,
		MaxPrice:  3_500_000,
		Candidates: []events.DriverCandidate{
			{DriverID: driverA, AskID: uuid.NewString(), VehicleID: uuid.NewString(), DistanceKm: 2.4},
			{DriverID: driverB, AskID: uuid.NewString(), VehicleID: uuid.NewString(), DistanceKm: 4.1},
		},
	}

	engine := &fakeEngine{}
	c := NewMatchingConsumer(engine)

	err := c.Handle(context.Background(), buildDelivery(t, events.RoutingKeyDriverCandidatesFound, payload))
	if err != nil {
		t.Fatalf("Handle trả lỗi: %v", err)
	}

	if len(engine.dispatched) != 2 {
		t.Fatalf("mong đợi 2 thông báo (mỗi tài xế một cái), nhận %d", len(engine.dispatched))
	}

	seen := map[string]bool{}
	for _, n := range engine.dispatched {
		seen[n.UserID.String()] = true

		if n.Type != entity.TypeDriverCandidate {
			t.Errorf("type = %q, mong đợi %q", n.Type, entity.TypeDriverCandidate)
		}
		if n.RecipientRole != entity.RoleDriver {
			t.Errorf("recipient_role = %q, mong đợi %q", n.RecipientRole, entity.RoleDriver)
		}

		if n.Channel != entity.ChannelPush {
			t.Errorf("channel = %q, mong đợi %q", n.Channel, entity.ChannelPush)
		}
		if n.RefType != entity.RefTypeBid || n.RefID != bidID {
			t.Errorf("ref = %s/%s, mong đợi bid/%s", n.RefType, n.RefID, bidID)
		}
		if n.Title == "" || n.Body == "" {
			t.Error("thông báo thiếu tiêu đề hoặc nội dung")
		}

		var deepLink map[string]string
		if err := json.Unmarshal([]byte(n.Data), &deepLink); err != nil {
			t.Errorf("data không phải JSON hợp lệ: %v", err)
		} else if deepLink["bid_id"] != bidID {
			t.Errorf("deep-link bid_id = %q, mong đợi %q", deepLink["bid_id"], bidID)
		}
	}

	if !seen[driverA] || !seen[driverB] {
		t.Errorf("thiếu tài xế trong danh sách nhận: %v", seen)
	}
}

func TestHandleMatchFoundNotifiesBothSides(t *testing.T) {
	shipperID := uuid.NewString()
	driverID := uuid.NewString()
	contractID := uuid.NewString()

	payload := events.MatchFound{
		ContractID:       contractID,
		BidID:            uuid.NewString(),
		AskID:            uuid.NewString(),
		ShipperID:        shipperID,
		DriverID:         driverID,
		VehicleID:        uuid.NewString(),
		ConsensusPrice:   3_200_000,
		ConsensusDeposit: 320_000,
	}

	engine := &fakeEngine{}
	c := NewMatchingConsumer(engine)

	if err := c.Handle(context.Background(), buildDelivery(t, events.RoutingKeyMatchFound, payload)); err != nil {
		t.Fatalf("Handle trả lỗi: %v", err)
	}

	if len(engine.dispatched) != 2 {
		t.Fatalf("mong đợi 2 thông báo (chủ hàng + tài xế), nhận %d", len(engine.dispatched))
	}

	byUser := map[string]entity.CreateNotificationParam{}
	for _, n := range engine.dispatched {
		byUser[n.UserID.String()] = n
	}

	shipperNotif, ok := byUser[shipperID]
	if !ok {
		t.Fatal("chủ hàng không nhận được thông báo")
	}
	if shipperNotif.RecipientRole != entity.RoleShipper {
		t.Errorf("vai trò người nhận phía chủ hàng = %q", shipperNotif.RecipientRole)
	}

	driverNotif, ok := byUser[driverID]
	if !ok {
		t.Fatal("tài xế không nhận được thông báo")
	}
	if driverNotif.RecipientRole != entity.RoleDriver {
		t.Errorf("vai trò người nhận phía tài xế = %q", driverNotif.RecipientRole)
	}

	if shipperNotif.Title == driverNotif.Title {
		t.Errorf("hai phía nhận cùng một tiêu đề %q — đáng lẽ phải khác nhau", shipperNotif.Title)
	}

	for _, n := range []entity.CreateNotificationParam{shipperNotif, driverNotif} {
		if n.Type != entity.TypeMatchFound {
			t.Errorf("type = %q, mong đợi %q", n.Type, entity.TypeMatchFound)
		}
		if n.RefType != entity.RefTypeMatch || n.RefID != contractID {
			t.Errorf("ref = %s/%s, mong đợi match/%s", n.RefType, n.RefID, contractID)
		}
	}
}

func TestHandleAcksUnparsableMessage(t *testing.T) {
	engine := &fakeEngine{}
	c := NewMatchingConsumer(engine)

	err := c.Handle(context.Background(), mq.Delivery{
		RoutingKey: events.RoutingKeyMatchFound,
		MessageID:  uuid.NewString(),
		Body:       []byte("{ không phải json"),
	})
	if err != nil {
		t.Fatalf("message hỏng phải được ACK (nil), nhận lỗi: %v", err)
	}
	if engine.callCount != 0 {
		t.Errorf("không được gọi engine với message hỏng, đã gọi %d lần", engine.callCount)
	}
}

func TestHandleIgnoresUnknownRoutingKey(t *testing.T) {
	engine := &fakeEngine{}
	c := NewMatchingConsumer(engine)

	err := c.Handle(context.Background(), buildDelivery(t, "matching.something.unknown", map[string]string{"x": "y"}))
	if err != nil {
		t.Fatalf("routing key lạ phải được bỏ qua, nhận lỗi: %v", err)
	}
	if engine.callCount != 0 {
		t.Errorf("không được gọi engine với routing key lạ, đã gọi %d lần", engine.callCount)
	}
}

func TestHandleAcksDuplicateEvent(t *testing.T) {
	engine := &fakeEngine{err: repo.ErrDuplicateEvent}
	c := NewMatchingConsumer(engine)

	payload := events.MatchFound{
		ContractID: uuid.NewString(),
		BidID:      uuid.NewString(),
		AskID:      uuid.NewString(),
		ShipperID:  uuid.NewString(),
		DriverID:   uuid.NewString(),
	}

	err := c.Handle(context.Background(), buildDelivery(t, events.RoutingKeyMatchFound, payload))
	if err != nil {
		t.Fatalf("event trùng phải được ACK (nil), nhận lỗi: %v", err)
	}
}

func TestHandleNacksOnTransientError(t *testing.T) {
	engine := &fakeEngine{err: errors.New("connection refused")}
	c := NewMatchingConsumer(engine)

	payload := events.MatchFound{
		ContractID: uuid.NewString(),
		BidID:      uuid.NewString(),
		AskID:      uuid.NewString(),
		ShipperID:  uuid.NewString(),
		DriverID:   uuid.NewString(),
	}

	err := c.Handle(context.Background(), buildDelivery(t, events.RoutingKeyMatchFound, payload))
	if err == nil {
		t.Fatal("lỗi tạm thời phải trả error để message được retry")
	}
}

func TestHandleSkipsInvalidDriverID(t *testing.T) {
	goodDriver := uuid.NewString()

	payload := events.DriverCandidatesFound{
		BidID:     uuid.NewString(),
		ShipperID: uuid.NewString(),
		Candidates: []events.DriverCandidate{
			{DriverID: "không-phải-uuid", AskID: uuid.NewString()},
			{DriverID: goodDriver, AskID: uuid.NewString()},
		},
	}

	engine := &fakeEngine{}
	c := NewMatchingConsumer(engine)

	if err := c.Handle(context.Background(), buildDelivery(t, events.RoutingKeyDriverCandidatesFound, payload)); err != nil {
		t.Fatalf("Handle trả lỗi: %v", err)
	}

	if len(engine.dispatched) != 1 {
		t.Fatalf("mong đợi 1 thông báo (bỏ qua ứng viên id hỏng), nhận %d", len(engine.dispatched))
	}
	if engine.dispatched[0].UserID.String() != goodDriver {
		t.Errorf("người nhận = %s, mong đợi %s", engine.dispatched[0].UserID, goodDriver)
	}
}