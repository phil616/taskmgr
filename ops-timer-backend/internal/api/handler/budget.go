package handler

import (
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/pkg/response"
	"ops-timer-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type BudgetHandler struct {
	budgetService *service.BudgetService
}

func NewBudgetHandler(budgetService *service.BudgetService) *BudgetHandler {
	return &BudgetHandler{budgetService: budgetService}
}

// ==================== Wallet 钱包 ====================

// ListWallets GET /wallets
func (h *BudgetHandler) ListWallets(c *gin.Context) {
	list, err := h.budgetService.ListWallets()
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, list)
}

// CreateWallet POST /wallets
func (h *BudgetHandler) CreateWallet(c *gin.Context) {
	var req dto.CreateWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}
	wallet, err := h.budgetService.CreateWallet(&req)
	if err != nil {
		response.BusinessError(c, err.Error())
		return
	}
	response.Created(c, wallet)
}

// GetWallet GET /wallets/:id
func (h *BudgetHandler) GetWallet(c *gin.Context) {
	id := c.Param("id")
	wallet, err := h.budgetService.GetWallet(id)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, wallet)
}

// UpdateWallet PUT /wallets/:id
func (h *BudgetHandler) UpdateWallet(c *gin.Context) {
	id := c.Param("id")
	var req dto.UpdateWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}
	wallet, err := h.budgetService.UpdateWallet(id, &req)
	if err != nil {
		response.BusinessError(c, err.Error())
		return
	}
	response.Success(c, wallet)
}

// DeleteWallet DELETE /wallets/:id
func (h *BudgetHandler) DeleteWallet(c *gin.Context) {
	id := c.Param("id")
	if err := h.budgetService.DeleteWallet(id); err != nil {
		response.BusinessError(c, err.Error())
		return
	}
	response.NoContent(c)
}

// ==================== BudgetCategory 分类 ====================

// ListCategories GET /budget/categories
func (h *BudgetHandler) ListCategories(c *gin.Context) {
	catType := c.Query("type")
	list, err := h.budgetService.ListCategories(catType)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, list)
}

// CreateCategory POST /budget/categories
func (h *BudgetHandler) CreateCategory(c *gin.Context) {
	var req dto.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}
	cat, err := h.budgetService.CreateCategory(&req)
	if err != nil {
		response.BusinessError(c, err.Error())
		return
	}
	response.Created(c, cat)
}

// UpdateCategory PUT /budget/categories/:id
func (h *BudgetHandler) UpdateCategory(c *gin.Context) {
	id := c.Param("id")
	var req dto.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}
	cat, err := h.budgetService.UpdateCategory(id, &req)
	if err != nil {
		response.BusinessError(c, err.Error())
		return
	}
	response.Success(c, cat)
}

// DeleteCategory DELETE /budget/categories/:id
func (h *BudgetHandler) DeleteCategory(c *gin.Context) {
	id := c.Param("id")
	if err := h.budgetService.DeleteCategory(id); err != nil {
		response.BusinessError(c, err.Error())
		return
	}
	response.NoContent(c)
}

// ==================== Transaction 收支记录 ====================

// ListTransactions GET /transactions
func (h *BudgetHandler) ListTransactions(c *gin.Context) {
	var params dto.TransactionQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	list, total, err := h.budgetService.ListTransactions(&params)
	if err != nil {
		response.BusinessError(c, err.Error())
		return
	}
	response.SuccessWithMeta(c, list, &response.Meta{
		Page:       params.Page,
		PageSize:   params.PageSize,
		Total:      total,
		TotalPages: response.CalculateTotalPages(total, params.PageSize),
	})
}

// CreateTransaction POST /transactions
func (h *BudgetHandler) CreateTransaction(c *gin.Context) {
	var req dto.CreateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}
	tx, err := h.budgetService.CreateTransaction(&req)
	if err != nil {
		response.BusinessError(c, err.Error())
		return
	}
	response.Created(c, tx)
}

// GetTransaction GET /transactions/:id
func (h *BudgetHandler) GetTransaction(c *gin.Context) {
	id := c.Param("id")
	tx, err := h.budgetService.GetTransaction(id)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, tx)
}

// UpdateTransaction PUT /transactions/:id
func (h *BudgetHandler) UpdateTransaction(c *gin.Context) {
	id := c.Param("id")
	var req dto.UpdateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}
	tx, err := h.budgetService.UpdateTransaction(id, &req)
	if err != nil {
		response.BusinessError(c, err.Error())
		return
	}
	response.Success(c, tx)
}

// DeleteTransaction DELETE /transactions/:id
func (h *BudgetHandler) DeleteTransaction(c *gin.Context) {
	id := c.Param("id")
	if err := h.budgetService.DeleteTransaction(id); err != nil {
		response.BusinessError(c, err.Error())
		return
	}
	response.NoContent(c)
}

// GetStats GET /budget/stats
func (h *BudgetHandler) GetStats(c *gin.Context) {
	var params dto.WalletStatRequest
	if err := c.ShouldBindQuery(&params); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}
	stats, err := h.budgetService.GetStats(&params)
	if err != nil {
		response.BusinessError(c, err.Error())
		return
	}
	response.Success(c, stats)
}
