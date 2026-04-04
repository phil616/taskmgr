const { get, patch, del, post } = require('../../utils/request');
const { formatDuration, formatDateTime } = require('../../utils/date');

Page({
  data: { unit: null, logs: [], loading: true },

  onLoad(options) {
    this.unitId = options.id;
  },

  onShow() {
    this.fetchUnit();
    this.fetchLogs();
  },

  onPullDownRefresh() {
    Promise.all([this.fetchUnit(), this.fetchLogs()]).finally(() => wx.stopPullDownRefresh());
  },

  async fetchUnit() {
    this.setData({ loading: true });
    try {
      const resp = await get(`/units/${this.unitId}`);
      const u = resp.data;
      if (u) {
        this.setData({ unit: this.formatUnit(u), loading: false });
      }
    } catch (err) {
      this.setData({ loading: false });
    }
  },

  async fetchLogs() {
    try {
      const resp = await get(`/units/${this.unitId}/logs`, { page_size: 20 });
      const logs = (resp.data || []).map(l => ({
        ...l,
        dateText: formatDateTime(l.operated_at),
        deltaSign: l.delta > 0 ? '+' : '',
      }));
      this.setData({ logs });
    } catch (_) {}
  },

  formatUnit(u) {
    const typeLabel = { time_countdown: '时间倒计时', time_countup: '时间正计时', count_countdown: '数量倒计', count_countup: '数量正计' }[u.type] || u.type;
    const priorityColor = { low: '#a6a6a6', normal: '#366ef4', high: '#e37318', critical: '#d54941' }[u.priority] || '#a6a6a6';
    let progressText = '';
    let progressPct = 0;

    if (u.type === 'time_countdown') {
      progressText = u.remaining_seconds !== undefined
        ? (u.remaining_seconds <= 0 ? '已超期' : `剩余 ${formatDuration(u.remaining_seconds)}`)
        : '';
      progressPct = u.progress ? Math.min(100, u.progress) : 0;
    } else if (u.type === 'time_countup') {
      progressText = u.elapsed_seconds ? `已计时 ${formatDuration(u.elapsed_seconds)}` : '计时中';
    } else {
      progressText = `${u.current_value || 0} / ${u.target_value || 0} ${u.unit_label || ''}`;
      progressPct = u.target_value ? Math.min(100, ((u.current_value || 0) / u.target_value) * 100) : 0;
    }

    return { ...u, typeLabel, priorityColor, progressText, progressPct: Math.round(progressPct) };
  },

  async stepUp() {
    try {
      await post(`/units/${this.unitId}/step`, { direction: 'up' });
      this.fetchUnit(); this.fetchLogs();
      wx.showToast({ title: '+1', icon: 'none', duration: 800 });
    } catch (err) { wx.showToast({ title: '操作失败', icon: 'none' }); }
  },

  async stepDown() {
    try {
      await post(`/units/${this.unitId}/step`, { direction: 'down' });
      this.fetchUnit(); this.fetchLogs();
      wx.showToast({ title: '-1', icon: 'none', duration: 800 });
    } catch (err) { wx.showToast({ title: '操作失败', icon: 'none' }); }
  },

  async togglePause() {
    const unit = this.data.unit;
    if (!unit) return;
    const newStatus = unit.status === 'active' ? 'paused' : 'active';
    try {
      await patch(`/units/${this.unitId}/status`, { status: newStatus });
      this.fetchUnit();
      wx.showToast({ title: newStatus === 'active' ? '已恢复' : '已暂停', icon: 'success' });
    } catch (err) { wx.showToast({ title: '操作失败', icon: 'none' }); }
  },

  async markComplete() {
    wx.showModal({
      title: '标记完成',
      content: '确认标记此单元为已完成？',
      success: async (res) => {
        if (res.confirm) {
          try {
            await patch(`/units/${this.unitId}/status`, { status: 'completed' });
            this.fetchUnit();
            wx.showToast({ title: '已完成', icon: 'success' });
          } catch (err) { wx.showToast({ title: '操作失败', icon: 'none' }); }
        }
      },
    });
  },

});
