package cloudinary

import (
	"context"
	"fmt"
	"log"
	"io"
	"time"

	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/cloudinary/cloudinary-go/v2"
)

type CloudinaryStorage struct {
	client *cloudinary.Cloudinary
}

func NewCloudinaryStorage(client *cloudinary.Cloudinary) *CloudinaryStorage {
	return &CloudinaryStorage{client: client}
}

func (c *CloudinaryStorage) Upload(ctx context.Context, file io.Reader, fileName string, folder string, prefix string) (string, string, string, error) {
	name := fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	result, err := c.client.Upload.Upload(ctx, file, uploader.UploadParams{
		PublicID: name,
		Folder:   folder,
	})

	if err != nil {
		return "", "", "", fmt.Errorf("gọi API Cloudinary thất bại: %w", err)
	}

	if result.Error.Message != "" {
		log.Printf("Cloudinary Upload Error: %s", result.Error.Message)
		return "", "", "", fmt.Errorf("cloudinary trả về lỗi: %s", result.Error.Message)
	}

	return name, result.PublicID, result.SecureURL, nil
}

func (c *CloudinaryStorage) Delete(ctx context.Context, publicID string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	result, err := c.client.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID: publicID,
	})
	if err != nil {
		return fmt.Errorf("gọi API Cloudinary thất bại: %w", err)
	}

	log.Printf("Cloudinary Destroy Result: %s", result.Result)

	if result.Error.Message != "" {
		log.Printf("Cloudinary Destroy Error Detail: %s", result.Error.Message)
		return fmt.Errorf("cloudinary xóa file thất bại: %s", result.Error.Message)
	}

	return nil
}
