const { get, patch } = require('../../utils/request');
const { formatDuration, formatDate } = require('../../utils/date');

Page({
  data: {
    summary: null,
    alertUnits: [],
    recentTodos: [],
    pendingCount: 0,
    unreadCount: 0,
    loading: true,
    user: null,
  },

  onLoad() {
    const userStr = wx.getStorageSync('user');
    if (userStr) {
      try { this.setData({ user: JSON.parse(userStr) }); } catch (e) {}
    }
  },

  onShow() {
    if (typeof this.getTabBar === 'function') {
      this.getTabBar().setData({ selected: 0 });
    }
    this.fetchDashboard();
  },

  async fetchDashboard() {
    this.setData({ loading: true });
    try {
      const [summaryResp, unitsResp, todosResp, notifResp] = await Promise.all([
        get('/units/summary'),
        get('/units', { status: 'active', sort_by: 'priority', sort_order: 'desc', page_size: 20 }),
        get('/todos', { status: 'pending', page_size: 10 }),
        get('/notifications/unread-count'),
      ]);

      const allUnits = (unitsResp.data || []);
      const alertUnits = allUnits
        .filter(u => u.type === 'time_countdown' && u.remaining_seconds !== undefined && u.remaining_seconds <= 7 * 86400)
        .map(u => ({
          ...u,
          alertColor: u.remaining_seconds <= 0 ? '#d54941' : u.remaining_seconds <= 86400 ? '#d54941' : '#e37318',
          alertLabel: u.remaining_seconds <= 0 ? '已超期' : u.remaining_seconds <= 86400 ? '紧急' : '即将到期',
          timeText: u.remaining_seconds <= 0 ? '已超期' : `距到期还剩 ${formatDuration(u.remaining_seconds)}`,
        }));

      const recentTodos = (todosResp.data || []).map(t => ({
        ...t,
        dueDateText: t.due_date ? `截止: ${formatDate(t.due_date)}` : '',
      }));

      this.setData({
        summary: summaryResp.data || null,
        alertUnits,
        recentTodos,
        pendingCount: todosResp.meta?.total || todosResp.data?.length || 0,
        unreadCount: notifResp.data?.count || 0,
        loading: false,
      });
    } catch (err) {
      console.error(err);
      this.setData({ loading: false });
    }
  },

  async toggleTodo(e) {
    const { id, status } = e.currentTarget.dataset;
    const newStatus = status === 'done' ? 'pending' : 'done';
    try {
      await patch(`/todos/${id}/status`, { status: newStatus });
      this.fetchDashboard();
    } catch (err) {
      wx.showToast({ title: '操作失败', icon: 'none' });
    }
  },

  goUnits() {
    wx.navigateTo({ url: '/pages/units/index' });
  },

  goTodos() {
    wx.switchTab({ url: '/pages/todos/index' });
  },

  goUnitDetail(e) {
    const id = e.currentTarget.dataset.id;
    wx.navigateTo({ url: `/pages/unit-detail/index?id=${id}` });
  },

  onPullDownRefresh() {
    this.fetchDashboard().then(() => wx.stopPullDownRefresh());
  },
});
