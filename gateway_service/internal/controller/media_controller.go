package controller

import (
	"io"

	"gateway_service/internal/response"

	"github.com/gin-gonic/gin"
	pb "github.com/logistic/api/logistic/media_service/v1"
)

type MediaController struct {
	mediaClient pb.MediaServiceClient
}

func NewMediaController(mediaClient pb.MediaServiceClient) *MediaController {
	return &MediaController{
		mediaClient: mediaClient,
	}
}

// UploadFile godoc
// @Summary      Tải file lên hệ thống
// @Description  Nhận file qua multipart/form-data và gọi gRPC sang media_service để lưu trữ (ví dụ Cloudinary).
// @Tags         Media
// @Accept       multipart/form-data
// @Produce      json
// @Param        file formData file true "File cần tải lên"
// @Param        folder formData string false "Thư mục lưu trữ"
// @Param        prefix formData string false "Tiền tố tên file"
// @Success      200 {object} map[string]interface{} "Tải lên thành công, trả về URL"
// @Failure      400 {object} map[string]interface{} "Lỗi thiếu file hoặc file không hợp lệ"
// @Failure      500 {object} map[string]interface{} "Lỗi server nội bộ"
// @Router       /api/v1/media/upload [post]
func (c *MediaController) UploadFile(ctx *gin.Context) {
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		response.BadRequest(ctx, "FILE_REQUIRED", "cần đính kèm file trong trường \"file\"")
		return
	}

	folder := ctx.PostForm("folder")
	prefix := ctx.PostForm("prefix")

	file, err := fileHeader.Open()
	if err != nil {
		response.BadRequest(ctx, "FILE_UNREADABLE", "không mở được file đã tải lên")
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		response.BadRequest(ctx, "FILE_UNREADABLE", "không đọc được nội dung file")
		return
	}

	resp, err := c.mediaClient.UploadFile(ctx.Request.Context(), &pb.UploadFileRequest{
		FileContent: fileBytes,
		FileName:    fileHeader.Filename,
		Folder:      folder,
		Prefix:      prefix,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}

	response.OKMessage(ctx, gin.H{
		"file_name": resp.FileName,
		"public_id": resp.PublicId,
		"url":       resp.Url,
	}, resp.Message)
}

// DeleteFile godoc
// @Summary      Xóa file
// @Description  Xóa file trên hệ thống lưu trữ thông qua public_id.
// @Tags         Media
// @Produce      json
// @Param        publicID path string true "Public ID của file cần xóa"
// @Success      200 {object} map[string]interface{} "Xóa thành công"
// @Failure      400 {object} map[string]interface{} "Thiếu public_id"
// @Failure      404 {object} map[string]interface{} "Không tìm thấy file"
// @Failure      500 {object} map[string]interface{} "Lỗi server nội bộ"
// @Router       /api/v1/media/files/{publicID} [delete]
func (c *MediaController) DeleteFile(ctx *gin.Context) {
	publicID := ctx.Param("publicID")
	if publicID == "" {
		response.BadRequest(ctx, "PUBLIC_ID_REQUIRED", "publicID là bắt buộc")
		return
	}

	resp, err := c.mediaClient.DeleteFile(ctx.Request.Context(), &pb.DeleteFileRequest{
		PublicId: publicID,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}

	response.OKMessage(ctx, nil, resp.Message)
}