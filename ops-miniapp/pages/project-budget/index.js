const { get, post, put, del } = require('../../utils/request');
const { formatAmount, formatDateTime } = require('../../utils/date');

Page({
  data: {
    projectId: '',
    project: null,
    budgetStats: null,
    transactions: [],
    categories: [],
    wallets: [],
    loading: false,

    // 预算进度
    usagePercent: 0,
    progressColor: '#008858',

    // 设置预算弹窗
    budgetDialogVisible: false,
    budgetForm: { max_budget: '' },
    savingBudget: false,

    // 新建/编辑交易弹窗
    txDialogVisible: false,
    editingTx: null,
    txForm: {
      type: 'expense',
      amount: '',
      category_id: '',
      categoryName: '',
      wallet_id: '',
      walletName: '',
      note: '',
      transaction_at: '',
    },
    saving: false,
  },

  onLoad(options) {
    const projectId = options.id || '';
    const projectName = decodeURIComponent(options.name || '项目预算');
    this.setData({ projectId });
    wx.setNavigationBarTitle({ title: projectName });
  },

  onShow() {
    this.fetchProject();
    this.fetchTransactions();
    this.fetchCategories();
    this.fetchWallets();
  },

  onPullDownRefresh() {
    Promise.all([this.fetchProject(), this.fetchTransactions()])
      .finally(() => wx.stopPullDownRefresh());
  },

  async fetchProject() {
    try {
      const resp = await get(`/projects/${this.data.projectId}`);
      const project = resp.data;
      if (!project) return;

      const stats = project.budget_stats || {};
      const usageRate = stats.usage_rate || 0;
      let progressColor = '#008858';
      if (usageRate >= 1) progressColor = '#d54941';
      else if (usageRate >= 0.8) progressColor = '#e37318';

      this.setData({
        project,
        budgetStats: {
          ...stats,
          totalIncomeText: formatAmount(stats.total_income || 0),
          totalExpenseText: formatAmount(stats.total_expense || 0),
          netAmountText: formatAmount(Math.abs(stats.net_amount || 0)),
          netPositive: (stats.net_amount || 0) >= 0,
          maxBudgetText: formatAmount(stats.max_budget || 0),
          remainingText: formatAmount(Math.abs(stats.remaining || 0)),
          remainingPositive: (stats.remaining || 0) >= 0,
          usagePercentText: ((usageRate) * 100).toFixed(1),
        },
        usagePercent: Math.min(usageRate * 100, 100),
        progressColor,
      });
    } catch (err) {
      wx.showToast({ title: err.message || '加载失败', icon: 'none' });
    }
  },

  async fetchTransactions() {
    this.setData({ loading: true });
    try {
      const resp = await get('/transactions', {
        project_id: this.data.projectId,
        page_size: 200,
      });
      const transactions = (resp.data || []).map(t => ({
        ...t,
        amountText: formatAmount(t.amount),
        dateText: formatDateTime(t.transaction_at),
        typeColor: { income: '#008858', expense: '#d54941', transfer: '#366ef4' }[t.type] || '#8b8b8b',
        typeLabel: { income: '收入', expense: '支出', transfer: '转账' }[t.type] || t.type,
        typeSign: { income: '+', expense: '-', transfer: '⇄' }[t.type] || '',
      }));
      this.setData({ transactions, loading: false });
    } catch (err) {
      this.setData({ loading: false });
    }
  },

  async fetchCategories() {
    try {
      const resp = await get('/budget/categories');
      this.setData({ categories: resp.data || [] });
    } catch (_) {}
  },

  async fetchWallets() {
    try {
      const resp = await get('/wallets');
      this.setData({ wallets: resp.data || [] });
    } catch (_) {}
  },

  // ---- 设置预算 ----
  openBudgetDialog() {
    this.setData({
      budgetDialogVisible: true,
      budgetForm: { max_budget: String(this.data.project?.max_budget || 0) },
    });
  },
  closeBudgetDialog() { this.setData({ budgetDialogVisible: false }); },
  onBudgetInput(e) { this.setData({ 'budgetForm.max_budget': e.detail.value }); },

  async saveBudget() {
    const val = parseFloat(this.data.budgetForm.max_budget) || 0;
    if (val < 0) {
      wx.showToast({ title: '预算不能为负', icon: 'none' });
      return;
    }
    this.setData({ savingBudget: true });
    try {
      await put(`/projects/${this.data.projectId}`, { max_budget: val });
      this.setData({ budgetDialogVisible: false });
      this.fetchProject();
      wx.showToast({ title: '已更新', icon: 'success' });
    } catch (err) {
      wx.showToast({ title: err.message || '保存失败', icon: 'none' });
    } finally {
      this.setData({ savingBudget: false });
    }
  },

  // ---- 新建/编辑交易 ----
  openCreateTx() {
    const now = new Date();
    const pad = n => String(n).padStart(2, '0');
    const dt = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())}`;
    const defaultWallet = this.data.wallets[0];
    this.setData({
      txDialogVisible: true,
      editingTx: null,
      txForm: {
        type: 'expense',
        amount: '',
        category_id: '',
        categoryName: '',
        wallet_id: defaultWallet?.id || '',
        walletName: defaultWallet?.name || '',
        note: '',
        transaction_at: dt,
      },
    });
  },

  openEditTx(e) {
    const tx = e.currentTarget.dataset.tx;
    const cats = this.data.categories;
    const catObj = cats.find(c => c.id === tx.category_id);
    const walletObj = this.data.wallets.find(w => w.id === tx.wallet_id);
    this.setData({
      txDialogVisible: true,
      editingTx: tx,
      txForm: {
        type: tx.type,
        amount: String(tx.amount),
        category_id: tx.category_id || '',
        categoryName: catObj ? catObj.name : '',
        wallet_id: tx.wallet_id || '',
        walletName: walletObj ? walletObj.name : '',
        note: tx.note || '',
        transaction_at: tx.transaction_at ? tx.transaction_at.slice(0, 16).replace('T', ' ') : '',
      },
    });
  },

  closeTxDialog() { this.setData({ txDialogVisible: false }); },

  onTxTypeChange(e) {
    const idx = e.currentTarget?.dataset?.value ?? e.detail?.value;
    this.setData({ 'txForm.type': ['expense', 'income', 'transfer'][idx] || 'expense' });
  },
  onTxAmountInput(e) { this.setData({ 'txForm.amount': e.detail.value }); },
  onTxCategoryChange(e) {
    const cat = this.data.categories[e.detail.value];
    this.setData({ 'txForm.category_id': cat?.id || '', 'txForm.categoryName': cat?.name || '' });
  },
  onTxWalletChange(e) {
    const w = this.data.wallets[e.detail.value];
    this.setData({ 'txForm.wallet_id': w?.id || '', 'txForm.walletName': w?.name || '' });
  },
  onTxNoteInput(e) { this.setData({ 'txForm.note': e.detail.value }); },
  onTxDateChange(e) {
    const cur = this.data.txForm.transaction_at;
    const time = cur.includes(' ') ? cur.split(' ')[1] : '00:00';
    this.setData({ 'txForm.transaction_at': `${e.detail.value} ${time}` });
  },

  async saveTx() {
    const { txForm, editingTx, projectId } = this.data;
    const amount = parseFloat(txForm.amount);
    if (!amount || amount <= 0) {
      wx.showToast({ title: '请输入有效金额', icon: 'none' });
      return;
    }
    if (!txForm.wallet_id) {
      wx.showToast({ title: '请选择钱包', icon: 'none' });
      return;
    }
    this.setData({ saving: true });
    try {
      const payload = {
        wallet_id: txForm.wallet_id,
        type: txForm.type,
        amount,
        note: txForm.note,
        transaction_at: txForm.transaction_at + ':00',
        project_id: projectId,
      };
      if (txForm.category_id) payload.category_id = txForm.category_id;

      if (editingTx) {
        await put(`/transactions/${editingTx.id}`, {
          amount,
          note: txForm.note,
          category_id: txForm.category_id || null,
          project_id: projectId,
          transaction_at: txForm.transaction_at + ':00',
        });
      } else {
        await post('/transactions', payload);
      }
      this.setData({ txDialogVisible: false });
      this.fetchProject();
      this.fetchTransactions();
      wx.showToast({ title: editingTx ? '已更新' : '已创建', icon: 'success' });
    } catch (err) {
      wx.showToast({ title: err.message || '保存失败', icon: 'none' });
    } finally {
      this.setData({ saving: false });
    }
  },

  async deleteTx(e) {
    const id = e.currentTarget.dataset.id;
    wx.showModal({
      title: '确认删除',
      content: '删除后将回滚对应钱包余额。',
      success: async (res) => {
        if (res.confirm) {
          try {
            await del(`/transactions/${id}`);
            this.fetchProject();
            this.fetchTransactions();
            wx.showToast({ title: '已删除', icon: 'success' });
          } catch (err) {
            wx.showToast({ title: '删除失败', icon: 'none' });
          }
        }
      },
    });
  },
});
