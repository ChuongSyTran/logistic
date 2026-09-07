package controller

import (
	"bytes"
	"context"
	"media_service/internal/storage"

	pb "github.com/logistic/api/logistic/media_service/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MediaController struct {
	pb.UnimplementedMediaServiceServer
	storage storage.FileStorage
}

func NewMediaController(storage storage.FileStorage) *MediaController {
	return &MediaController{storage: storage}
}

func (c *MediaController) UploadFile(ctx context.Context, req *pb.UploadFileRequest) (*pb.UploadFileResponse, error) {
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

func (c *MediaController) DeleteFile(ctx context.Context, req *pb.DeleteFileRequest) (*pb.DeleteFileResponse, error) {
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
