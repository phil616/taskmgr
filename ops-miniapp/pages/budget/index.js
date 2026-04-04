const { get, post, put, del } = require('../../utils/request');
const { formatAmount, currentMonthStart, currentMonthEnd } = require('../../utils/date');

Page({
  data: {
    wallets: [],
    loading: false,
    overviewIncome: 0,
    overviewExpense: 0,
    overviewNet: 0,

    // 钱包弹窗
    walletDialogVisible: false,
    editingWallet: null,
    walletForm: { name: '', type: 'bank', typeLabel: '银行卡', balance: 0, currency: 'CNY', color: '#0052d9', description: '', is_default: false },
    saving: false,

    typeOptions: [
      { label: '银行卡', value: 'bank' },
      { label: '现金', value: 'cash' },
      { label: '信用卡', value: 'credit' },
      { label: '支付宝', value: 'alipay' },
      { label: '微信', value: 'wechat' },
      { label: '其他', value: 'other' },
    ],
    colorOptions: ['#0052d9', '#008858', '#e37318', '#d54941', '#7b1fa2', '#0097a7', '#455a64', '#e91e63'],
  },

  onLoad() {},

  onShow() {
    if (typeof this.getTabBar === 'function') {
      this.getTabBar().setData({ selected: 3 });
    }
    this.fetchWallets();
    this.fetchOverview();
  },

  onPullDownRefresh() {
    Promise.all([this.fetchWallets(), this.fetchOverview()])
      .finally(() => wx.stopPullDownRefresh());
  },

  async fetchWallets() {
    this.setData({ loading: true });
    try {
      const resp = await get('/wallets');
      const wallets = (resp.data || []).map(w => ({
        ...w,
        balanceText: formatAmount(w.balance),
        incomeText: formatAmount(w.total_income || 0),
        expenseText: formatAmount(w.total_expense || 0),
        typeLabel: { bank: '银行卡', cash: '现金', credit: '信用卡', alipay: '支付宝', wechat: '微信', other: '其他' }[w.type] || w.type,
        typeIcon: { bank: 'bank', cash: 'money-circle', credit: 'creditcard', alipay: 'secured', wechat: 'chat', other: 'wallet' }[w.type] || 'wallet',
      }));
      this.setData({ wallets, loading: false });
    } catch (err) {
      this.setData({ loading: false });
      wx.showToast({ title: err.message || '加载失败', icon: 'none' });
    }
  },

  async fetchOverview() {
    try {
      const resp = await get('/budget/stats', {
        start_date: currentMonthStart(),
        end_date: currentMonthEnd(),
      });
      const d = resp.data || {};
      this.setData({
        overviewIncome: formatAmount(d.total_income || 0),
        overviewExpense: formatAmount(d.total_expense || 0),
        overviewNet: formatAmount((d.total_income || 0) - (d.total_expense || 0)),
        overviewNetPositive: (d.total_income || 0) >= (d.total_expense || 0),
      });
    } catch (_) {}
  },

  openWallet(e) {
    const wallet = e.currentTarget.dataset.wallet;
    wx.navigateTo({ url: `/pages/budget-detail/index?id=${wallet.id}&name=${wallet.name}` });
  },

  openCreateWallet() {
    this.setData({
      walletDialogVisible: true,
      editingWallet: null,
      walletForm: { name: '', type: 'bank', typeLabel: '银行卡', balance: 0, currency: 'CNY', color: '#0052d9', description: '', is_default: false },
    });
  },

  openEditWallet(e) {
    const wallet = e.currentTarget.dataset.wallet;
    const typeOpt = this.data.typeOptions.find(t => t.value === wallet.type);
    this.setData({
      walletDialogVisible: true,
      editingWallet: wallet,
      walletForm: {
        name: wallet.name,
        type: wallet.type,
        typeLabel: typeOpt ? typeOpt.label : wallet.type,
        balance: wallet.balance,
        currency: wallet.currency || 'CNY',
        color: wallet.color || '#0052d9',
        description: wallet.description || '',
        is_default: wallet.is_default || false,
      },
    });
  },

  closeWalletDialog() {
    this.setData({ walletDialogVisible: false });
  },

  onWalletNameInput(e) { this.setData({ 'walletForm.name': e.detail.value }); },
  onWalletDescInput(e) { this.setData({ 'walletForm.description': e.detail.value }); },
  onWalletBalanceInput(e) { this.setData({ 'walletForm.balance': parseFloat(e.detail.value) || 0 }); },
  onWalletTypeChange(e) {
    const opt = this.data.typeOptions[e.detail.value];
    this.setData({ 'walletForm.type': opt.value, 'walletForm.typeLabel': opt.label });
  },
  onWalletColorSelect(e) { this.setData({ 'walletForm.color': e.currentTarget.dataset.color }); },
  onDefaultChange(e) { this.setData({ 'walletForm.is_default': e.detail.value }); },

  async saveWallet() {
    const { walletForm, editingWallet } = this.data;
    if (!walletForm.name.trim()) {
      wx.showToast({ title: '请输入钱包名称', icon: 'none' });
      return;
    }
    this.setData({ saving: true });
    try {
      if (editingWallet) {
        await put(`/wallets/${editingWallet.id}`, {
          name: walletForm.name,
          type: walletForm.type,
          color: walletForm.color,
          description: walletForm.description,
          is_default: walletForm.is_default,
        });
      } else {
        await post('/wallets', walletForm);
      }
      this.setData({ walletDialogVisible: false });
      this.fetchWallets();
      this.fetchOverview();
      wx.showToast({ title: editingWallet ? '已更新' : '已创建', icon: 'success' });
    } catch (err) {
      wx.showToast({ title: err.message || '保存失败', icon: 'none' });
    } finally {
      this.setData({ saving: false });
    }
  },

  async deleteWallet() {
    const wallet = this.data.editingWallet;
    if (!wallet) return;
    wx.showModal({
      title: '删除钱包',
      content: `确认删除「${wallet.name}」？关联记录不会被删除。`,
      success: async (res) => {
        if (res.confirm) {
          try {
            await del(`/wallets/${wallet.id}`);
            this.setData({ walletDialogVisible: false });
            this.fetchWallets();
            this.fetchOverview();
            wx.showToast({ title: '已删除', icon: 'success' });
          } catch (err) {
            wx.showToast({ title: '删除失败', icon: 'none' });
          }
        }
      },
    });
  },

  goCategories() {
    wx.navigateTo({ url: '/pages/budget-categories/index' });
  },
});
