package service

import (
	"fmt"
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/pkg/timeutil"
	"ops-timer-backend/internal/repository"
	"time"

	"github.com/google/uuid"
)

type BudgetService struct {
	walletRepo   *repository.WalletRepository
	categoryRepo *repository.BudgetCategoryRepository
	txRepo       *repository.TransactionRepository
}

func NewBudgetService(
	walletRepo *repository.WalletRepository,
	categoryRepo *repository.BudgetCategoryRepository,
	txRepo *repository.TransactionRepository,
) *BudgetService {
	return &BudgetService{
		walletRepo:   walletRepo,
		categoryRepo: categoryRepo,
		txRepo:       txRepo,
	}
}

// ==================== 钱包 ====================

func (s *BudgetService) CreateWallet(req *dto.CreateWalletRequest) (*dto.WalletResponse, error) {
	if req.Type == "" {
		req.Type = model.WalletTypeBank
	}
	if req.Currency == "" {
		req.Currency = "CNY"
	}
	sortOrder := 0
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}
	// 若设置为默认，先清除其他默认
	if req.IsDefault {
		if err := s.walletRepo.ClearDefault(); err != nil {
			return nil, err
		}
	}
	w := &model.Wallet{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Type:        req.Type,
		Balance:     req.Balance,
		Currency:    req.Currency,
		Color:       req.Color,
		Icon:        req.Icon,
		Description: req.Description,
		IsDefault:   req.IsDefault,
		SortOrder:   sortOrder,
	}
	if err := s.walletRepo.Create(w); err != nil {
		return nil, err
	}
	return s.walletToResponse(w), nil
}

func (s *BudgetService) GetWallet(id string) (*dto.WalletResponse, error) {
	w, err := s.walletRepo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("钱包不存在")
	}
	resp := s.walletToResponse(w)
	start, end := currentMonthRange()
	resp.TotalIncome = s.walletRepo.SumByType(id, model.TransactionTypeIncome, start, end)
	resp.TotalExpense = s.walletRepo.SumByType(id, model.TransactionTypeExpense, start, end)
	return resp, nil
}

func (s *BudgetService) ListWallets() ([]*dto.WalletResponse, error) {
	wallets, err := s.walletRepo.List()
	if err != nil {
		return nil, err
	}
	start, end := currentMonthRange()
	result := make([]*dto.WalletResponse, 0, len(wallets))
	for i := range wallets {
		resp := s.walletToResponse(&wallets[i])
		resp.TotalIncome = s.walletRepo.SumByType(wallets[i].ID, model.TransactionTypeIncome, start, end)
		resp.TotalExpense = s.walletRepo.SumByType(wallets[i].ID, model.TransactionTypeExpense, start, end)
		result = append(result, resp)
	}
	return result, nil
}

func (s *BudgetService) UpdateWallet(id string, req *dto.UpdateWalletRequest) (*dto.WalletResponse, error) {
	w, err := s.walletRepo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("钱包不存在")
	}
	if req.Name != "" {
		w.Name = req.Name
	}
	if req.Type != "" {
		w.Type = req.Type
	}
	if req.Color != nil {
		w.Color = *req.Color
	}
	if req.Icon != nil {
		w.Icon = *req.Icon
	}
	if req.Description != nil {
		w.Description = *req.Description
	}
	if req.IsDefault != nil && *req.IsDefault {
		if err := s.walletRepo.ClearDefault(); err != nil {
			return nil, err
		}
		w.IsDefault = true
	} else if req.IsDefault != nil {
		w.IsDefault = *req.IsDefault
	}
	if req.SortOrder != nil {
		w.SortOrder = *req.SortOrder
	}
	if err := s.walletRepo.Update(w); err != nil {
		return nil, err
	}
	return s.walletToResponse(w), nil
}

func (s *BudgetService) DeleteWallet(id string) error {
	if _, err := s.walletRepo.FindByID(id); err != nil {
		return fmt.Errorf("钱包不存在")
	}
	return s.walletRepo.Delete(id)
}

// ==================== 分类 ====================

func (s *BudgetService) CreateCategory(req *dto.CreateCategoryRequest) (*dto.CategoryResponse, error) {
	sortOrder := 0
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}
	c := &model.BudgetCategory{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Type:      req.Type,
		Color:     req.Color,
		Icon:      req.Icon,
		SortOrder: sortOrder,
	}
	if err := s.categoryRepo.Create(c); err != nil {
		return nil, err
	}
	return s.categoryToResponse(c), nil
}

func (s *BudgetService) ListCategories(catType string) ([]*dto.CategoryResponse, error) {
	list, err := s.categoryRepo.List(catType)
	if err != nil {
		return nil, err
	}
	result := make([]*dto.CategoryResponse, 0, len(list))
	for i := range list {
		result = append(result, s.categoryToResponse(&list[i]))
	}
	return result, nil
}

func (s *BudgetService) UpdateCategory(id string, req *dto.UpdateCategoryRequest) (*dto.CategoryResponse, error) {
	c, err := s.categoryRepo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("分类不存在")
	}
	if c.IsSystem {
		return nil, fmt.Errorf("系统内置分类不可修改")
	}
	if req.Name != "" {
		c.Name = req.Name
	}
	if req.Type != "" {
		c.Type = req.Type
	}
	if req.Color != nil {
		c.Color = *req.Color
	}
	if req.Icon != nil {
		c.Icon = *req.Icon
	}
	if req.SortOrder != nil {
		c.SortOrder = *req.SortOrder
	}
	if err := s.categoryRepo.Update(c); err != nil {
		return nil, err
	}
	return s.categoryToResponse(c), nil
}

func (s *BudgetService) DeleteCategory(id string) error {
	c, err := s.categoryRepo.FindByID(id)
	if err != nil {
		return fmt.Errorf("分类不存在")
	}
	if c.IsSystem {
		return fmt.Errorf("系统内置分类不可删除")
	}
	if count := s.categoryRepo.CountByCategory(id); count > 0 {
		return fmt.Errorf("分类下还有 %d 条交易记录，请先移除后再删除", count)
	}
	return s.categoryRepo.Delete(id)
}

// InitDefaultCategories 初始化系统默认分类（首次使用调用）
func (s *BudgetService) InitDefaultCategories() error {
	list, _ := s.categoryRepo.List("")
	if len(list) > 0 {
		return nil // 已有分类，跳过
	}
	defaults := []model.BudgetCategory{
		{ID: uuid.New().String(), Name: "餐饮", Type: "expense", Icon: "mdi-food", Color: "#F4511E", IsSystem: true, SortOrder: 1},
		{ID: uuid.New().String(), Name: "交通", Type: "expense", Icon: "mdi-bus", Color: "#039BE5", IsSystem: true, SortOrder: 2},
		{ID: uuid.New().String(), Name: "购物", Type: "expense", Icon: "mdi-shopping", Color: "#E53935", IsSystem: true, SortOrder: 3},
		{ID: uuid.New().String(), Name: "娱乐", Type: "expense", Icon: "mdi-gamepad-variant", Color: "#8E24AA", IsSystem: true, SortOrder: 4},
		{ID: uuid.New().String(), Name: "居家", Type: "expense", Icon: "mdi-home", Color: "#43A047", IsSystem: true, SortOrder: 5},
		{ID: uuid.New().String(), Name: "医疗", Type: "expense", Icon: "mdi-hospital-box", Color: "#E91E63", IsSystem: true, SortOrder: 6},
		{ID: uuid.New().String(), Name: "教育", Type: "expense", Icon: "mdi-book-open", Color: "#00ACC1", IsSystem: true, SortOrder: 7},
		{ID: uuid.New().String(), Name: "其他支出", Type: "expense", Icon: "mdi-dots-horizontal", Color: "#757575", IsSystem: true, SortOrder: 99},
		{ID: uuid.New().String(), Name: "工资", Type: "income", Icon: "mdi-briefcase", Color: "#43A047", IsSystem: true, SortOrder: 1},
		{ID: uuid.New().String(), Name: "奖金", Type: "income", Icon: "mdi-trophy", Color: "#FB8C00", IsSystem: true, SortOrder: 2},
		{ID: uuid.New().String(), Name: "投资收益", Type: "income", Icon: "mdi-trending-up", Color: "#1E88E5", IsSystem: true, SortOrder: 3},
		{ID: uuid.New().String(), Name: "其他收入", Type: "income", Icon: "mdi-dots-horizontal", Color: "#757575", IsSystem: true, SortOrder: 99},
	}
	for i := range defaults {
		if err := s.categoryRepo.Create(&defaults[i]); err != nil {
			return err
		}
	}
	return nil
}

// ==================== 交易记录 ====================

func (s *BudgetService) CreateTransaction(req *dto.CreateTransactionRequest) (*dto.TransactionResponse, error) {
	// 验证钱包
	wallet, err := s.walletRepo.FindByID(req.WalletID)
	if err != nil {
		return nil, fmt.Errorf("钱包不存在")
	}

	// 转账需验证目标钱包
	if req.Type == model.TransactionTypeTransfer {
		if req.ToWalletID == nil {
			return nil, fmt.Errorf("转账必须指定目标钱包")
		}
		if _, err := s.walletRepo.FindByID(*req.ToWalletID); err != nil {
			return nil, fmt.Errorf("目标钱包不存在")
		}
	}

	// 验证分类
	if req.CategoryID != nil {
		if _, err := s.categoryRepo.FindByID(*req.CategoryID); err != nil {
			return nil, fmt.Errorf("分类不存在")
		}
	}

	txAt, err := parseTime(req.TransactionAt)
	if err != nil {
		return nil, fmt.Errorf("transaction_at 格式错误")
	}

	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	tx := &model.Transaction{
		ID:            uuid.New().String(),
		WalletID:      req.WalletID,
		CategoryID:    req.CategoryID,
		ProjectID:     req.ProjectID,
		Type:          req.Type,
		Amount:        req.Amount,
		Note:          req.Note,
		Tags:          tags,
		TransactionAt: txAt,
		ToWalletID:    req.ToWalletID,
	}

	if err := s.txRepo.Create(tx); err != nil {
		return nil, err
	}

	// 更新余额
	switch req.Type {
	case model.TransactionTypeIncome:
		_ = s.walletRepo.UpdateBalance(req.WalletID, req.Amount)
	case model.TransactionTypeExpense:
		_ = s.walletRepo.UpdateBalance(req.WalletID, -req.Amount)
	case model.TransactionTypeTransfer:
		_ = s.walletRepo.UpdateBalance(req.WalletID, -req.Amount)
		_ = s.walletRepo.UpdateBalance(*req.ToWalletID, req.Amount)
	}

	// 重新读取以包含关联
	saved, _ := s.txRepo.FindByID(tx.ID)
	if saved != nil {
		tx = saved
	}
	return s.txToResponse(tx, wallet), nil
}

func (s *BudgetService) GetTransaction(id string) (*dto.TransactionResponse, error) {
	tx, err := s.txRepo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("记录不存在")
	}
	return s.txToResponse(tx, tx.Wallet), nil
}

func (s *BudgetService) ListTransactions(params *dto.TransactionQueryParams) ([]*dto.TransactionResponse, int64, error) {
	filter := &repository.TransactionFilter{
		WalletID:   params.WalletID,
		CategoryID: params.CategoryID,
		ProjectID:  params.ProjectID,
		TxType:     params.Type,
		Page:       params.Page,
		PageSize:   params.PageSize,
		Keyword:    params.Keyword,
		MinAmount:  params.MinAmount,
		MaxAmount:  params.MaxAmount,
	}
	if params.StartDate != "" {
		if t, err := timeutil.ParseDate(params.StartDate); err == nil {
			filter.StartDate = t
		}
	}
	if params.EndDate != "" {
		if t, err := timeutil.ParseDate(params.EndDate); err == nil {
			filter.EndDate = t.Add(24 * time.Hour)
		}
	}

	list, total, err := s.txRepo.List(filter)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*dto.TransactionResponse, 0, len(list))
	for i := range list {
		result = append(result, s.txToResponse(&list[i], list[i].Wallet))
	}
	return result, total, nil
}

func (s *BudgetService) UpdateTransaction(id string, req *dto.UpdateTransactionRequest) (*dto.TransactionResponse, error) {
	tx, err := s.txRepo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("记录不存在")
	}

	oldAmount := tx.Amount

	if req.CategoryID != nil {
		if *req.CategoryID == "" {
			tx.CategoryID = nil
		} else {
			if _, err := s.categoryRepo.FindByID(*req.CategoryID); err != nil {
				return nil, fmt.Errorf("分类不存在")
			}
			tx.CategoryID = req.CategoryID
		}
	}
	if req.ProjectID != nil {
		if *req.ProjectID == "" {
			tx.ProjectID = nil
		} else {
			tx.ProjectID = req.ProjectID
		}
	}
	if req.Amount > 0 {
		tx.Amount = req.Amount
	}
	if req.Note != nil {
		tx.Note = *req.Note
	}
	if req.Tags != nil {
		tx.Tags = req.Tags
	}
	if req.TransactionAt != "" {
		t, err := parseTime(req.TransactionAt)
		if err == nil {
			tx.TransactionAt = t
		}
	}

	if err := s.txRepo.Update(tx); err != nil {
		return nil, err
	}

	// 调整余额差额
	diff := tx.Amount - oldAmount
	if diff != 0 {
		switch tx.Type {
		case model.TransactionTypeIncome:
			_ = s.walletRepo.UpdateBalance(tx.WalletID, diff)
		case model.TransactionTypeExpense:
			_ = s.walletRepo.UpdateBalance(tx.WalletID, -diff)
		}
	}

	updated, _ := s.txRepo.FindByID(id)
	if updated != nil {
		tx = updated
	}
	return s.txToResponse(tx, tx.Wallet), nil
}

func (s *BudgetService) DeleteTransaction(id string) error {
	tx, err := s.txRepo.FindByID(id)
	if err != nil {
		return fmt.Errorf("记录不存在")
	}
	// 回滚余额
	switch tx.Type {
	case model.TransactionTypeIncome:
		_ = s.walletRepo.UpdateBalance(tx.WalletID, -tx.Amount)
	case model.TransactionTypeExpense:
		_ = s.walletRepo.UpdateBalance(tx.WalletID, tx.Amount)
	case model.TransactionTypeTransfer:
		_ = s.walletRepo.UpdateBalance(tx.WalletID, tx.Amount)
		if tx.ToWalletID != nil {
			_ = s.walletRepo.UpdateBalance(*tx.ToWalletID, -tx.Amount)
		}
	}
	return s.txRepo.Delete(id)
}

func (s *BudgetService) GetStats(params *dto.WalletStatRequest) (*dto.WalletStatResponse, error) {
	var start, end time.Time
	if params.StartDate != "" {
		if t, err := timeutil.ParseDate(params.StartDate); err == nil {
			start = t
		}
	}
	if params.EndDate != "" {
		if t, err := timeutil.ParseDate(params.EndDate); err == nil {
			end = t.Add(24 * time.Hour)
		}
	}

	stats := s.txRepo.StatByWallet(params.WalletID, start, end)
	resp := &dto.WalletStatResponse{CategoryStats: []dto.CategoryStatItem{}}
	for _, st := range stats {
		switch st.Type {
		case model.TransactionTypeIncome:
			resp.TotalIncome = st.Total
			resp.TransactionCount += int(st.Count)
		case model.TransactionTypeExpense:
			resp.TotalExpense = st.Total
			resp.TransactionCount += int(st.Count)
		}
	}
	resp.NetAmount = resp.TotalIncome - resp.TotalExpense

	catStats := s.txRepo.StatByCategory(params.WalletID, start, end)
	for _, cs := range catStats {
		item := dto.CategoryStatItem{
			Type:  cs.TxType,
			Total: cs.Total,
			Count: cs.Count,
		}
		if cs.CategoryID != nil {
			item.CategoryID = *cs.CategoryID
			if cat, err := s.categoryRepo.FindByID(*cs.CategoryID); err == nil {
				item.CategoryName = cat.Name
				item.CategoryIcon = cat.Icon
				item.CategoryColor = cat.Color
			}
		} else {
			item.CategoryName = "未分类"
			item.CategoryIcon = "mdi-tag-off"
		}
		resp.CategoryStats = append(resp.CategoryStats, item)
	}

	return resp, nil
}

// ==================== 内部工具 ====================

func (s *BudgetService) walletToResponse(w *model.Wallet) *dto.WalletResponse {
	return &dto.WalletResponse{
		ID:          w.ID,
		Name:        w.Name,
		Type:        w.Type,
		Balance:     w.Balance,
		Currency:    w.Currency,
		Color:       w.Color,
		Icon:        w.Icon,
		Description: w.Description,
		IsDefault:   w.IsDefault,
		SortOrder:   w.SortOrder,
		CreatedAt:   w.CreatedAt,
		UpdatedAt:   w.UpdatedAt,
	}
}

func (s *BudgetService) categoryToResponse(c *model.BudgetCategory) *dto.CategoryResponse {
	return &dto.CategoryResponse{
		ID:        c.ID,
		Name:      c.Name,
		Type:      c.Type,
		Color:     c.Color,
		Icon:      c.Icon,
		SortOrder: c.SortOrder,
		IsSystem:  c.IsSystem,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func (s *BudgetService) txToResponse(t *model.Transaction, wallet *model.Wallet) *dto.TransactionResponse {
	tags := []string{}
	if t.Tags != nil {
		tags = t.Tags
	}
	resp := &dto.TransactionResponse{
		ID:            t.ID,
		WalletID:      t.WalletID,
		CategoryID:    t.CategoryID,
		ProjectID:     t.ProjectID,
		Type:          t.Type,
		Amount:        t.Amount,
		Note:          t.Note,
		Tags:          tags,
		TransactionAt: t.TransactionAt,
		ToWalletID:    t.ToWalletID,
		CreatedAt:     t.CreatedAt,
	}
	if wallet != nil {
		resp.WalletName = wallet.Name
	}
	if t.Category != nil {
		resp.CategoryName = t.Category.Name
		resp.CategoryIcon = t.Category.Icon
		resp.CategoryColor = t.Category.Color
	}
	if t.ToWallet != nil {
		resp.ToWalletName = t.ToWallet.Name
	}
	if t.Project != nil {
		resp.ProjectName = t.Project.Title
	}
	return resp
}

// currentMonthRange 返回当前月份的起始和结束时间（左闭右开）
func currentMonthRange() (start, end time.Time) {
	now := timeutil.Now()
	start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	end = start.AddDate(0, 1, 0)
	return
}
