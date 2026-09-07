package nats_jetstream

import (
	"context"
	"errors"
	"testing"

	"matching_service/internal/biz"
	"matching_service/internal/entity"
	"matching_service/internal/mapper/generated"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

type fakeConsumer struct {
	topic   string
	handler biz.EventHandler
	err     error
}

func (f *fakeConsumer) Consume(_ context.Context, topic string, handler biz.EventHandler) error {
	f.topic = topic
	f.handler = handler
	return f.err
}

type fakeEngine struct {
	biz.MatchingEngine
	gotBidID uuid.UUID
	gotAsk   *entity.Ask
	calls    int
	err      error
}

func (f *fakeEngine) ProcessOfferQueue(_ context.Context, bidID uuid.UUID, offerAsk *entity.Ask) error {
	f.calls++
	f.gotBidID = bidID
	f.gotAsk = offerAsk
	return f.err
}

func askPayload(t *testing.T, ask entity.Ask) []byte {
	t.Helper()
	m := &generated.MatchingMapperImpl{}
	payload, err := proto.Marshal(m.EntityAskToPbAsk(ask))
	if err != nil {
		t.Fatalf("không mã hoá được Ask: %v", err)
	}
	return payload
}

func startWith(t *testing.T, engine biz.MatchingEngine) *fakeConsumer {
	t.Helper()
	consumer := &fakeConsumer{}
	if err := StartOfferConsumer(context.Background(), consumer, engine, &generated.MatchingMapperImpl{}); err != nil {
		t.Fatalf("StartOfferConsumer lỗi: %v", err)
	}
	if consumer.handler == nil {
		t.Fatal("không đăng ký handler nào")
	}
	return consumer
}

func TestConsumerDangKyDungSubjectBaoGia(t *testing.T) {
	consumer := startWith(t, &fakeEngine{})
	if consumer.topic != OfferSubject {
		t.Fatalf("subject = %q, mong đợi %q", consumer.topic, OfferSubject)
	}
}

func TestBaoGiaDuocChuyenToiProcessOfferQueue(t *testing.T) {
	engine := &fakeEngine{}
	consumer := startWith(t, engine)

	bidID := uuid.New()
	ask := entity.Ask{
		ID:       uuid.New(),
		DriverID: uuid.New(),
		MinPrice: 4_200_000,
	}

	err := consumer.handler(context.Background(), "matching.offers."+bidID.String(), askPayload(t, ask))
	if err != nil {
		t.Fatalf("handler lỗi: %v", err)
	}

	if engine.calls != 1 {
		t.Fatalf("ProcessOfferQueue được gọi %d lần, mong đợi 1", engine.calls)
	}
	if engine.gotBidID != bidID {
		t.Errorf("bidID = %s, mong đợi %s", engine.gotBidID, bidID)
	}
	if engine.gotAsk == nil || engine.gotAsk.ID != ask.ID {
		t.Errorf("ask truyền xuống không khớp: %+v", engine.gotAsk)
	}
	if engine.gotAsk != nil && engine.gotAsk.MinPrice != ask.MinPrice {
		t.Errorf("giá báo = %v, mong đợi %v", engine.gotAsk.MinPrice, ask.MinPrice)
	}
}

func TestBanTinHongBiLoaiKhongThuLai(t *testing.T) {
	cases := map[string]struct {
		subject string
		payload []byte
	}{
		"subject sai dạng":           {"matching.offers", []byte{}},
		"bidID không hợp lệ":         {"matching.offers.khong-phai-uuid", []byte{}},
		"payload không giải mã được": {"matching.offers." + uuid.New().String(), []byte{0xff, 0xfe, 0xfd}},
	}

	for ten, tc := range cases {
		t.Run(ten, func(t *testing.T) {
			engine := &fakeEngine{}
			consumer := startWith(t, engine)

			err := consumer.handler(context.Background(), tc.subject, tc.payload)
			if !errors.Is(err, biz.ErrNonRetryable) {
				t.Fatalf("phải là ErrNonRetryable, nhận: %v", err)
			}
			if engine.calls != 0 {
				t.Fatal("không được gọi ProcessOfferQueue với bản tin hỏng")
			}
		})
	}
}

func TestDurableNameHopLeVoiNats(t *testing.T) {
	cases := map[string]string{
		"matching.offers.*":           "matching-offers",
		"matching.>":                  "matching",
		"matching.drivers.rejected.*": "matching-drivers-rejected",
	}

	for topic, mongDoi := range cases {
		if got := durableName(topic); got != mongDoi {
			t.Errorf("durableName(%q) = %q, mong đợi %q", topic, got, mongDoi)
		}
	}
}
