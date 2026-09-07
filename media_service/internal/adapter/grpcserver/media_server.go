package grpcserver

import (
	"bytes"
	"context"

	"media_service/internal/app"

	pb "github.com/logistic/api/logistic/media_service/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MediaServer struct {
	pb.UnimplementedMediaServiceServer
	storage app.FileStoragePort
}

func NewMediaServer(storage app.FileStoragePort) *MediaServer {
	return &MediaServer{storage: storage}
}

// Aliases for backwards compatibility
type MediaController = MediaServer
var NewMediaController = NewMediaServer

func (c *MediaServer) UploadFile(ctx context.Context, req *pb.UploadFileRequest) (*pb.UploadFileResponse, error) {
	if len(req.FileContent) == 0 {
		return nil, status.Error(codes.InvalidArgument, "file content is empty")
	}

	reader := bytes.NewReader(req.FileContent)
	fileName, publicID, url, err := c.storage.Upload(ctx, reader, req.FileName, req.Folder, req.Prefix)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to upload file: %v", err)
	}

	return &pb.UploadFileResponse{
		FileName: fileName,
		PublicId: publicID,
		Url:      url,
		Message:  "file uploaded successfully",
	}, nil
}

func (c *MediaServer) DeleteFile(ctx context.Context, req *pb.DeleteFileRequest) (*pb.DeleteFileResponse, error) {
	if req.PublicId == "" {
		return nil, status.Error(codes.InvalidArgument, "public_id is required")
	}

	err := c.storage.Delete(ctx, req.PublicId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete file: %v", err)
	}

	return &pb.DeleteFileResponse{
		Message: "file deleted successfully",
	}, nil
}
