package nats_jetstream

import (
	"context"
	"fmt"
	"log"
	"strings"

	"matching_service/internal/biz"
	"matching_service/internal/mapper"

	pb "github.com/logistic/api/logistic/matching_service/v1"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

// Mỗi đơn một subject riêng, nên bidID đọc từ subject chứ không có trong payload.
const OfferSubject = "matching.offers.*"

const offerSubjectPrefix = "matching.offers."

// StartOfferConsumer nối hàng đợi báo giá vào ProcessOfferQueue. Thiếu nó thì bid
// không bao giờ rời PENDING và cả nhánh thương lượng - chốt hợp đồng đứng im.
// Đi vòng qua hàng đợi để nhiều báo giá cùng đơn được xử lý tuần tự, ai trước thắng.
func StartOfferConsumer(
	ctx context.Context,
	consumer biz.EventConsumer,
	engine biz.MatchingEngine,
	appMapper mapper.MatchingMapper,
) error {
	handler := func(ctx context.Context, subject string, payload []byte) error {
		bidID, err := bidIDFromSubject(subject)
		if err != nil {
			return fmt.Errorf("%w: %v", biz.ErrNonRetryable, err)
		}

		var pbAsk pb.Ask
		if err := proto.Unmarshal(payload, &pbAsk); err != nil {
			return fmt.Errorf("%w: giải mã Ask thất bại: %v", biz.ErrNonRetryable, err)
		}

		ask, err := appMapper.PbAskToEntity(&pbAsk)
		if err != nil {
			return fmt.Errorf("%w: dựng entity Ask thất bại: %v", biz.ErrNonRetryable, err)
		}

		// Mapper bỏ ID (chặn client tự đặt ở SubmitAsk); ở đây ID là thật, phải giữ.
		if ask.ID, err = mapper.BytesToUUID(pbAsk.GetId()); err != nil {
			return fmt.Errorf("%w: askID trong payload không hợp lệ: %v", biz.ErrNonRetryable, err)
		}

		return engine.ProcessOfferQueue(ctx, bidID, &ask)
	}

	if err := consumer.Consume(ctx, OfferSubject, handler); err != nil {
		return err
	}

	log.Printf("[matching_service] đang lắng nghe hàng đợi báo giá trên %s", OfferSubject)
	return nil
}

func bidIDFromSubject(subject string) (uuid.UUID, error) {
	raw, ok := strings.CutPrefix(subject, offerSubjectPrefix)
	if !ok {
		return uuid.Nil, fmt.Errorf("subject %q không đúng dạng %s{bidID}", subject, offerSubjectPrefix)
	}

	bidID, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("bidID trong subject %q không hợp lệ: %w", subject, err)
	}
	return bidID, nil
}
