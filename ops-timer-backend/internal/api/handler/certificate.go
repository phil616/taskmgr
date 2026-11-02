package handler

import (
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/pkg/certparse"
	"ops-timer-backend/internal/pkg/response"
	"ops-timer-backend/internal/service"
)

type CertificateHandler struct{ service *service.CertificateService }

func NewCertificateHandler(s *service.CertificateService) *CertificateHandler {
	return &CertificateHandler{service: s}
}

func certificateError(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.NotFound(c, "证书附件或关联对象不存在")
		return
	}
	var inputError *service.CertificateInputError
	if errors.As(err, &inputError) {
		response.BadRequest(c, inputError.Error(), nil)
		return
	}
	response.InternalError(c, "证书操作失败")
}

func (h *CertificateHandler) Upload(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, certparse.MaxFileSize+(64<<10))
	if err := c.Request.ParseMultipartForm(certparse.MaxFileSize + (64 << 10)); err != nil {
		response.BadRequest(c, "上传格式错误或文件超过 4 MiB", nil)
		return
	}
	defer c.Request.MultipartForm.RemoveAll()
	files := c.Request.MultipartForm.File["file"]
	if len(files) != 1 {
		response.BadRequest(c, "每次请选择一个证书文件", nil)
		return
	}
	file, err := files[0].Open()
	if err != nil {
		response.BadRequest(c, "无法读取上传文件", nil)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, certparse.MaxFileSize+1))
	if err != nil {
		response.BadRequest(c, "无法读取上传文件", nil)
		return
	}
	var owner *dto.AttachmentOwner
	ownerType, ownerID := c.PostForm("owner_type"), c.PostForm("owner_id")
	if ownerType != "" || ownerID != "" {
		owner = &dto.AttachmentOwner{OwnerType: ownerType, OwnerID: ownerID}
	}
	a, err := h.service.Upload(files[0].Filename, data, owner)
	if err != nil {
		certificateError(c, err)
		return
	}
	response.Created(c, a)
}

func (h *CertificateHandler) List(c *gin.Context) {
	var q dto.AttachmentQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.BadRequest(c, "查询参数错误", nil)
		return
	}
	items, total, err := h.service.List(q)
	if err != nil {
		certificateError(c, err)
		return
	}
	response.SuccessWithMeta(c, items, &response.Meta{Page: q.Page, PageSize: q.PageSize, Total: total, TotalPages: response.CalculateTotalPages(total, q.PageSize)})
}

func (h *CertificateHandler) Download(c *gin.Context) {
	a, err := h.service.Download(c.Param("id"))
	if err != nil {
		certificateError(c, err)
		return
	}
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": a.Filename}))
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "application/octet-stream", a.Content)
}

func (h *CertificateHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Param("id")); err != nil {
		certificateError(c, err)
		return
	}
	response.NoContent(c)
}

func (h *CertificateHandler) Link(c *gin.Context) {
	var owner dto.AttachmentOwner
	if err := c.ShouldBindJSON(&owner); err != nil {
		response.BadRequest(c, "附件关联参数错误", nil)
		return
	}
	if err := h.service.Link(c.Param("id"), owner, c.Request.Method == http.MethodDelete); err != nil {
		certificateError(c, err)
		return
	}
	response.NoContent(c)
}

func (h *CertificateHandler) CreateCountdown(c *gin.Context) {
	u, err := h.service.CreateCountdown(c.Param("id"))
	if err != nil {
		certificateError(c, err)
		return
	}
	response.Success(c, u)
}
