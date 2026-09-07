package http

import (
	"net/http"

	"gateway_service/internal/conf"
	"gateway_service/internal/controller"
	"gateway_service/internal/middleware"

	pbauth "github.com/logistic/api/logistic/auth_service/v1"
	pbmatching "github.com/logistic/api/logistic/matching_service/v1"
	pbmedia "github.com/logistic/api/logistic/media_service/v1"
	pbnotification "github.com/logistic/api/logistic/notification_service/v1"
	pbuser "github.com/logistic/api/logistic/user_service/v1"
	pbvehicle "github.com/logistic/api/logistic/vehicle_service/v1"
	pbkyc "github.com/logistic/api/logistic/kyc_service/v1"
	"github.com/logistic/pkg/authn"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	_ "gateway_service/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Clients struct {
	Auth         pbauth.AuthServiceClient
	Matching     pbmatching.MatchingEngineServiceClient
	Media        pbmedia.MediaServiceClient
	User         pbuser.UserServiceClient
	Vehicle      pbvehicle.VehicleServiceClient
	Notification pbnotification.NotificationServiceClient
	Kyc          pbkyc.KycServiceClient
}

func RegisterGatewayRoutes(
	engine *gin.Engine,
	clients Clients,
	auth *middleware.Authenticator,
	cfg *conf.Config,
) {
	engine.Use(
		otelgin.Middleware("gateway_service"),
		middleware.RequestID(),
		middleware.Recovery(),
		middleware.AccessLog(),
		middleware.StripClientIdentity(),
		middleware.ErrorGuard(),
	)

	authController := controller.NewAuthController(clients.Auth, clients.User, cfg.Server.IsProduction)
	mediaController := controller.NewMediaController(clients.Media)
	matchingController := controller.NewMatchingController(clients.Matching, clients.Vehicle)
	userController := controller.NewUserController(clients.User)
	kycController := controller.NewKycController(clients.Kyc)
	vehicleController := controller.NewVehicleController(clients.Vehicle, clients.User, clients.Kyc)
	notifController := controller.NewNotificationController(clients.Notification)

	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	engine.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := engine.Group("/api/v1")

	publicAuth := api.Group("/auth")
	{
		publicAuth.POST("/register", authController.Register)
		publicAuth.POST("/login", authController.Login)
		publicAuth.POST("/refresh", authController.Refresh)
		publicAuth.GET("/google/login", authController.GoogleLogin)
		publicAuth.GET("/google/callback", authController.GoogleCallback)
	}

	authedAuth := api.Group("/auth", auth.Required())
	{
		authedAuth.GET("/me", authController.GetInfo)
		authedAuth.POST("/logout", authController.Logout)
	}

	secured := api.Group("", auth.Required())

	media := secured.Group("/media")
	{
		media.POST("/upload", mediaController.UploadFile)
		media.DELETE("/files/:publicID", mediaController.DeleteFile)
	}

	users := secured.Group("/users")
	{
		users.GET("/:user_id", userController.GetUser)
		users.PUT("/:user_id", userController.UpdateUser)

		users.GET("/:user_id/driver-profile", userController.GetDriverProfile)
		users.PUT("/:user_id/driver-profile", userController.UpdateDriverProfile)
		users.GET("/:user_id/shipper-profile", userController.GetShipperProfile)
		users.PUT("/:user_id/shipper-profile", userController.UpdateShipperProfile)

		users.PUT("/:user_id/kyc", kycController.SubmitKYC)
		users.GET("/:user_id/kyc", kycController.GetKYC)

		users.POST("/:user_id/addresses", userController.CreateAddress)
		users.GET("/:user_id/addresses", userController.ListAddresses)

		users.POST("/:user_id/devices", userController.RegisterDevice)
		users.GET("/:user_id/devices", userController.ListDevices)

		users.GET("/:user_id/notifications", notifController.ListNotifications)
		users.PUT("/:user_id/notifications/read-all", notifController.MarkAllAsRead)
		users.GET("/:user_id/notifications/unread-count", notifController.GetUnreadCount)
		users.GET("/:user_id/notification-preferences", notifController.GetPreferences)
		users.PUT("/:user_id/notification-preferences", notifController.UpdatePreferences)
	}

	api.POST("/users/register", userController.RegisterUser)

	addresses := secured.Group("/addresses")
	{
		addresses.PUT("/:id", userController.UpdateAddress)
		addresses.DELETE("/:id", userController.DeleteAddress)
	}

	devices := secured.Group("/devices")
	{
		devices.DELETE("/:id", userController.DeleteDevice)
	}

	notifications := secured.Group("/notifications")
	{
		notifications.GET("/:id", notifController.GetNotification)
		notifications.PUT("/:id/read", notifController.MarkAsRead)
		notifications.DELETE("/:id", notifController.DeleteNotification)
	}

	vehicles := secured.Group("/vehicles")
	{
		vehicles.POST("", vehicleController.RegisterVehicle)
		vehicles.GET("", vehicleController.ListVehicles)

		vehicles.POST("/nearby", vehicleController.SearchNearbyVehicles)

		vehicles.GET("/:id", vehicleController.GetVehicle)
		vehicles.PUT("/:id", vehicleController.UpdateVehicle)
		vehicles.DELETE("/:id", vehicleController.DeleteVehicle)
		vehicles.PUT("/:id/status", vehicleController.UpdateVehicleStatus)

		vehicles.POST("/:id/documents", vehicleController.UploadVehicleDocument)
		vehicles.GET("/:id/documents", vehicleController.ListVehicleDocuments)

		vehicles.POST("/:id/location", vehicleController.ReportLocation)
		vehicles.GET("/:id/location", vehicleController.GetVehicleLocation)
	}

	vehicleDocs := secured.Group("/vehicle-documents")
	{
		vehicleDocs.DELETE("/:id", vehicleController.DeleteVehicleDocument)
	}

	drivers := secured.Group("/drivers")
	{
		drivers.POST("/:driver_id/availability", vehicleController.SetDriverAvailability)
		drivers.GET("/:driver_id/availability", vehicleController.GetDriverAvailability)
	}

	matching := secured.Group("/matching")
	{
		matching.POST("/bids", matchingController.SubmitBid)
		matching.POST("/asks", matchingController.SubmitAsk)
		matching.POST("/offers", matchingController.SubmitOffer)
		matching.POST("/offers/reject", matchingController.RejectOffer)
		matching.POST("/matches/accept", matchingController.AcceptMatch)
	}

	admin := api.Group("/admin", auth.Required(), middleware.RequireRole(authn.RoleAdmin))
	{
		adminUsers := admin.Group("/users")
		{
			adminUsers.GET("", userController.AdminListUsers)
			adminUsers.GET("/stats", userController.AdminGetUserStats)
			adminUsers.PUT("/:id/status", userController.AdminUpdateUserStatus)
			adminUsers.DELETE("/:id", userController.AdminDeleteUser)
		}

		adminKyc := admin.Group("/kyc")
		{
			adminKyc.GET("/pending", kycController.ListPendingKYC)
			adminKyc.PUT("/:user_id/review", kycController.ReviewKYC)
			adminKyc.GET("/count-pending", kycController.CountPendingKYC)
		}

		adminVehicles := admin.Group("/vehicles")
		{
			adminVehicles.GET("", vehicleController.AdminListVehicles)
			adminVehicles.GET("/stats", vehicleController.AdminGetVehicleStats)
			adminVehicles.PUT("/:id/verify", vehicleController.AdminVerifyVehicle)
		}

		adminVehicleDocs := admin.Group("/vehicle-documents")
		{
			adminVehicleDocs.GET("/pending", vehicleController.AdminListPendingDocuments)
			adminVehicleDocs.PUT("/:id/review", vehicleController.AdminReviewDocument)
		}

		adminNotifications := admin.Group("/notifications")
		{
			adminNotifications.GET("", notifController.AdminListNotifications)
			adminNotifications.GET("/stats", notifController.AdminGetNotificationStats)
			adminNotifications.POST("/send", notifController.AdminSendNotification)
		}

		adminTemplates := admin.Group("/notification-templates")
		{
			adminTemplates.GET("", notifController.AdminListTemplates)
			adminTemplates.POST("", notifController.AdminCreateTemplate)
			adminTemplates.PUT("/:id", notifController.AdminUpdateTemplate)
			adminTemplates.DELETE("/:id", notifController.AdminDeleteTemplate)
		}
	}
}