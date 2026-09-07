package di

import (
	"context"
	"log"

	"notification_service/ent"
	"notification_service/ent/notificationtemplate"
)

type templateSeed struct {
	code          string
	name          string
	channel       notificationtemplate.Channel
	titleTemplate string
	bodyTemplate  string
}

var defaultTemplates = []templateSeed{
	{
		code:          "DRIVER_CANDIDATE",
		name:          "Tài xế — có đơn hàng phù hợp gần bạn",
		channel:       notificationtemplate.ChannelPush,
		titleTemplate: "Có đơn hàng phù hợp gần bạn",
		bodyTemplate:  "Đơn hàng {{weight_kg}} kg / {{volume_m3}} m³ cách bạn {{distance_km}} km. Giá tối đa {{max_price}} đ. Vào xem ngay để báo giá.",
	},
	{
		code:          "MATCH_FOUND_SHIPPER",
		name:          "Chủ hàng — đã tìm được xe",
		channel:       notificationtemplate.ChannelPush,
		titleTemplate: "Đã tìm được xe cho đơn hàng của bạn",
		bodyTemplate:  "Đơn hàng của bạn đã được ghép với một tài xế. Giá chốt {{price}} đ, đặt cọc {{deposit}} đ.",
	},
	{
		code:          "MATCH_FOUND_DRIVER",
		name:          "Tài xế — vừa nhận được đơn hàng",
		channel:       notificationtemplate.ChannelPush,
		titleTemplate: "Bạn vừa nhận được một đơn hàng",
		bodyTemplate:  "Chuyến hàng đã được xác nhận. Giá chốt {{price}} đ. Mở app để xem điểm lấy hàng.",
	},
	{
		code:          "OFFER_RECEIVED",
		name:          "Chủ hàng — nhận được báo giá mới",
		channel:       notificationtemplate.ChannelPush,
		titleTemplate: "Bạn nhận được một báo giá mới",
		bodyTemplate:  "Một tài xế vừa báo giá {{price}} đ cho đơn hàng của bạn.",
	},
	{
		code:          "OFFER_REJECTED",
		name:          "Tài xế — báo giá không được chọn",
		channel:       notificationtemplate.ChannelInApp,
		titleTemplate: "Báo giá của bạn không được chọn",
		bodyTemplate:  "Chủ hàng đã chọn tài xế khác cho đơn này.",
	},
	{
		code:          "CARGO_SUGGESTED",
		name:          "Tài xế — có đơn hàng cho chuyến vừa đăng",
		channel:       notificationtemplate.ChannelPush,
		titleTemplate: "Có đơn hàng cho chuyến của bạn",
		bodyTemplate:  "Tìm thấy {{total_found}} đơn hàng phù hợp với chuyến bạn vừa đăng. Xem và báo giá ngay.",
	},
}

func seedDefaultTemplates(ctx context.Context, client *ent.Client) {
	for _, seed := range defaultTemplates {
		exists, err := client.NotificationTemplate.Query().
			Where(
				notificationtemplate.CodeEQ(seed.code),
				notificationtemplate.ChannelEQ(seed.channel),
				notificationtemplate.LocaleEQ(localeVI),
			).
			Exist(ctx)
		if err != nil {
			log.Printf("[notification_service] không kiểm được mẫu %s: %v", seed.code, err)
			continue
		}
		if exists {
			continue
		}

		_, err = client.NotificationTemplate.Create().
			SetCode(seed.code).
			SetName(seed.name).
			SetChannel(seed.channel).
			SetLocale(localeVI).
			SetTitleTemplate(seed.titleTemplate).
			SetBodyTemplate(seed.bodyTemplate).
			SetIsActive(true).
			Save(ctx)
		if err != nil {
			log.Printf("[notification_service] không tạo được mẫu %s: %v", seed.code, err)
			continue
		}
		log.Printf("[notification_service] đã tạo mẫu thông báo mặc định %s", seed.code)
	}
}

const localeVI = "vi"
