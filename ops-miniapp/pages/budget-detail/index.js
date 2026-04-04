const { get, post, put, del } = require('../../utils/request');
const { formatAmount, formatDateTime } = require('../../utils/date');

Page({
  data: {
    walletId: '',
    walletName: '',
    wallet: null,
    transactions: [],
    categories: [],
    loading: false,
    page: 1,
    totalPages: 1,
    hasMore: true,
    typeFilter: '',
    typeFilterLabel: '全部类型',

    // 新建/编辑交易弹窗
    txDialogVisible: false,
    editingTx: null,
    txForm: {
      type: 'expense',
      amount: '',
      category_id: '',
      categoryName: '',
      note: '',
      transaction_at: '',
      to_wallet_id: '',
      toWalletName: '',
    },
    saving: false,
    wallets: [],

    typeFilterOptions: [
      { label: '全部', value: '' },
      { label: '收入', value: 'income' },
      { label: '支出', value: 'expense' },
      { label: '转账', value: 'transfer' },
    ],
  },

  onLoad(options) {
    const walletId = options.id || '';
    const walletName = decodeURIComponent(options.name || '钱包');
    this.setData({ walletId, walletName });
    wx.setNavigationBarTitle({ title: walletName });
  },

  onShow() {
    this.fetchTransactions(true);
    this.fetchCategories();
    this.fetchWallets();
    this.fetchWalletInfo();
  },

  onPullDownRefresh() {
    this.fetchTransactions(true).finally(() => wx.stopPullDownRefresh());
  },

  onReachBottom() {
    if (this.data.hasMore && !this.data.loading) {
      this.setData({ page: this.data.page + 1 });
      this.fetchTransactions(false);
    }
  },

  async fetchWalletInfo() {
    try {
      const resp = await get(`/wallets/${this.data.walletId}`);
      const w = resp.data;
      if (w) {
        this.setData({
          wallet: {
            ...w,
            balanceText: formatAmount(w.balance),
            incomeText: formatAmount(w.total_income || 0),
            expenseText: formatAmount(w.total_expense || 0),
          },
        });
      }
    } catch (_) {}
  },

  async fetchTransactions(reset) {
    if (reset) this.setData({ page: 1, transactions: [], hasMore: true });
    this.setData({ loading: true });
    try {
      const params = {
        wallet_id: this.data.walletId,
        page: this.data.page,
        page_size: 20,
      };
      if (this.data.typeFilter) params.type = this.data.typeFilter;

      const resp = await get('/transactions', params);
      const newTxs = (resp.data || []).map(t => ({
        ...t,
        amountText: formatAmount(t.amount),
        dateText: formatDateTime(t.transaction_at),
        typeColor: { income: '#008858', expense: '#d54941', transfer: '#366ef4' }[t.type] || '#8b8b8b',
        typeLabel: { income: '收入', expense: '支出', transfer: '转账' }[t.type] || t.type,
        typeSign: { income: '+', expense: '-', transfer: '⇄' }[t.type] || '',
      }));

      const totalPages = resp.meta?.total_pages || 1;
      this.setData({
        transactions: reset ? newTxs : [...this.data.transactions, ...newTxs],
        totalPages,
        hasMore: this.data.page < totalPages,
        loading: false,
      });
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
      const wallets = (resp.data || []).filter(w => w.id !== this.data.walletId);
      this.setData({ wallets });
    } catch (_) {}
  },

  onTypeFilterChange(e) {
    const opt = this.data.typeFilterOptions[e.detail.value];
    this.setData({ typeFilter: opt.value, typeFilterLabel: opt.label || '全部类型', page: 1 });
    this.fetchTransactions(true);
  },

  // ---- 创建/编辑交易 ----
  openCreateTx() {
    const now = new Date();
    const pad = n => String(n).padStart(2, '0');
    const dt = `${now.getFullYear()}-${pad(now.getMonth()+1)}-${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())}`;
    this.setData({
      txDialogVisible: true,
      editingTx: null,
      txForm: { type: 'expense', amount: '', category_id: '', categoryName: '', note: '', transaction_at: dt, to_wallet_id: '', toWalletName: '' },
    });
  },

  openEditTx(e) {
    const tx = e.currentTarget.dataset.tx;
    const cats = this.data.categories;
    const ws = this.data.wallets;
    const catObj = cats.find(c => c.id === tx.category_id);
    const walletObj = ws.find(w => w.id === tx.to_wallet_id);
    this.setData({
      txDialogVisible: true,
      editingTx: tx,
      txForm: {
        type: tx.type,
        amount: String(tx.amount),
        category_id: tx.category_id || '',
        categoryName: catObj ? catObj.name : '',
        note: tx.note || '',
        transaction_at: tx.transaction_at ? tx.transaction_at.slice(0, 16).replace('T', ' ') : '',
        to_wallet_id: tx.to_wallet_id || '',
        toWalletName: walletObj ? walletObj.name : '',
      },
    });
  },

  closeTxDialog() {
    this.setData({ txDialogVisible: false });
  },

  onTxTypeChange(e) { this.setData({ 'txForm.type': ['expense', 'income', 'transfer'][e.detail.value] || 'expense' }); },
  onTxAmountInput(e) { this.setData({ 'txForm.amount': e.detail.value }); },
  onTxCategoryChange(e) {
    const cats = this.data.categories;
    const cat = cats[e.detail.value];
    this.setData({ 'txForm.category_id': cat?.id || '', 'txForm.categoryName': cat?.name || '' });
  },
  onTxNoteInput(e) { this.setData({ 'txForm.note': e.detail.value }); },
  onTxDateChange(e) {
    const cur = this.data.txForm.transaction_at;
    const time = cur.includes(' ') ? cur.split(' ')[1] : '00:00';
    this.setData({ 'txForm.transaction_at': `${e.detail.value} ${time}` });
  },
  onTxToWalletChange(e) {
    const ws = this.data.wallets;
    const w = ws[e.detail.value];
    this.setData({ 'txForm.to_wallet_id': w?.id || '', 'txForm.toWalletName': w?.name || '' });
  },

  async saveTx() {
    const { txForm, editingTx, walletId } = this.data;
    const amount = parseFloat(txForm.amount);
    if (!amount || amount <= 0) {
      wx.showToast({ title: '请输入有效金额', icon: 'none' });
      return;
    }
    this.setData({ saving: true });
    try {
      const payload = {
        wallet_id: walletId,
        type: txForm.type,
        amount,
        note: txForm.note,
        transaction_at: txForm.transaction_at + ':00',
      };
      if (txForm.category_id) payload.category_id = txForm.category_id;
      if (txForm.type === 'transfer' && txForm.to_wallet_id) payload.to_wallet_id = txForm.to_wallet_id;

      if (editingTx) {
        await put(`/transactions/${editingTx.id}`, {
          amount,
          note: txForm.note,
          category_id: txForm.category_id || null,
          transaction_at: txForm.transaction_at + ':00',
        });
      } else {
        await post('/transactions', payload);
      }
      this.setData({ txDialogVisible: false });
      this.fetchTransactions(true);
      this.fetchWalletInfo();
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
      content: '删除后不可恢复',
      success: async (res) => {
        if (res.confirm) {
          try {
            await del(`/transactions/${id}`);
            this.fetchTransactions(true);
            this.fetchWalletInfo();
            wx.showToast({ title: '已删除', icon: 'success' });
          } catch (err) {
            wx.showToast({ title: '删除失败', icon: 'none' });
          }
        }
      },
    });
  },
});
