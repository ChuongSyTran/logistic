package di

import (
	"fmt"
	"media_service/internal/adapter/grpcserver"
	"media_service/internal/adapter/storage/cloudinary"
	"media_service/internal/conf"

	cld "github.com/cloudinary/cloudinary-go/v2"
	pb "github.com/logistic/api/logistic/media_service/v1"
	"google.golang.org/grpc"
)

func Injection(grpcServer *grpc.Server, cfg *conf.Config) error {
	cldClient, err := cld.NewFromURL(cfg.Cloudinary.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to cloudinary: %w", err)
	}

	cloudStorage := cloudinary.NewCloudinaryStorage(cldClient)
	mediaServer := grpcserver.NewMediaServer(cloudStorage)
	pb.RegisterMediaServiceServer(grpcServer, mediaServer)

	return nil
}
