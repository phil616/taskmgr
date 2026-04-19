package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/pkg/response"
	"ops-timer-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type BackupHandler struct {
	backupService *service.BackupService
}

func NewBackupHandler(backupService *service.BackupService) *BackupHandler {
	return &BackupHandler{backupService: backupService}
}

func (h *BackupHandler) Export(c *gin.Context) {
	payload, err := h.backupService.Export()
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	filename := fmt.Sprintf("task-manager-backup-%s.json", time.Now().UTC().Format("2006-01-02T15-04-05Z"))
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		response.InternalError(c, "序列化备份数据失败")
		return
	}

	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/json; charset=utf-8", append([]byte{0xEF, 0xBB, 0xBF}, raw...))
}

func (h *BackupHandler) Import(c *gin.Context) {
	strategy := c.DefaultPostForm("strategy", dto.BackupStrategyMerge)

	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "缺少备份文件", nil)
		return
	}

	src, err := file.Open()
	if err != nil {
		response.BadRequest(c, "打开备份文件失败", nil)
		return
	}
	defer src.Close()

	raw, err := service.ReadAllBackup(src)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	result, err := h.backupService.Import(strategy, raw)
	if err != nil {
		response.BusinessError(c, err.Error())
		return
	}

	response.Success(c, result)
}
