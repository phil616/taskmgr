const { get, post, put, patch, del } = require('../../utils/request');
const { formatDuration, formatDate } = require('../../utils/date');

Page({
  data: {
    units: [],
    loading: false,
    statusFilter: 'active',
    statusFilterLabel: '活跃',
    statusOptions: [
      { label: '活跃', value: 'active' },
      { label: '暂停', value: 'paused' },
      { label: '已完成', value: 'completed' },
      { label: '已归档', value: 'archived' },
    ],
    // 新建/编辑弹窗
    dialogVisible: false,
    editingUnit: null,
    unitForm: { title: '', description: '', type: 'time_countdown', priority: 'normal', target_time: '', target_value: 100, step: 1, unit_label: '', allow_exceed: false },
    saving: false,
    typeOptions: [
      { label: '时间倒计时', value: 'time_countdown' },
      { label: '时间正计时', value: 'time_countup' },
      { label: '数量倒计', value: 'count_countdown' },
      { label: '数量正计', value: 'count_countup' },
    ],
    priorityOptions: [
      { label: '低', value: 'low' },
      { label: '普通', value: 'normal' },
      { label: '高', value: 'high' },
      { label: '紧急', value: 'critical' },
    ],
  },

  onLoad() {},

  onShow() {
    this.fetchUnits();
  },

  onPullDownRefresh() {
    this.fetchUnits().finally(() => wx.stopPullDownRefresh());
  },

  onStatusFilter(e) {
    const opt = this.data.statusOptions[e.detail.value];
    this.setData({ statusFilter: opt.value, statusFilterLabel: opt.label });
    this.fetchUnits();
  },

  async fetchUnits() {
    this.setData({ loading: true });
    try {
      const params = { page_size: 100 };
      if (this.data.statusFilter) params.status = this.data.statusFilter;
      const resp = await get('/units', params);
      const units = (resp.data || []).map(u => this.formatUnit(u));
      this.setData({ units, loading: false });
    } catch (err) {
      this.setData({ loading: false });
      wx.showToast({ title: err.message || '加载失败', icon: 'none' });
    }
  },

  formatUnit(u) {
    const typeLabel = { time_countdown: '时间倒计时', time_countup: '时间正计时', count_countdown: '数量倒计', count_countup: '数量正计' }[u.type] || u.type;
    const priorityColor = { low: '#a6a6a6', normal: '#366ef4', high: '#e37318', critical: '#d54941' }[u.priority] || '#a6a6a6';
    const statusColor = { active: '#008858', paused: '#e37318', completed: '#366ef4', archived: '#a6a6a6' }[u.status] || '#a6a6a6';

    let progressText = '';
    let alertColor = null;
    if (u.type === 'time_countdown') {
      progressText = u.remaining_seconds !== undefined
        ? (u.remaining_seconds <= 0 ? '已超期' : `剩余 ${formatDuration(u.remaining_seconds)}`)
        : (u.target_time ? `目标: ${formatDate(u.target_time)}` : '');
      if (u.remaining_seconds !== undefined) {
        if (u.remaining_seconds <= 0) alertColor = '#d54941';
        else if (u.remaining_seconds <= 86400) alertColor = '#d54941';
        else if (u.remaining_seconds <= 7 * 86400) alertColor = '#e37318';
      }
    } else if (u.type === 'time_countup') {
      progressText = u.elapsed_seconds ? `已计时 ${formatDuration(u.elapsed_seconds)}` : '';
    } else {
      const cur = u.current_value || 0;
      const tgt = u.target_value || 0;
      progressText = `${cur} / ${tgt} ${u.unit_label || ''}`;
    }

    return { ...u, typeLabel, priorityColor, statusColor, progressText, alertColor };
  },

  openCreate() {
    this.setData({
      dialogVisible: true,
      editingUnit: null,
      unitForm: { title: '', description: '', type: 'time_countdown', priority: 'normal', target_time: '', target_value: 100, step: 1, unit_label: '', allow_exceed: false },
    });
  },

  openEdit(e) {
    const unit = e.currentTarget.dataset.unit;
    this.setData({
      dialogVisible: true,
      editingUnit: unit,
      unitForm: {
        title: unit.title,
        description: unit.description || '',
        type: unit.type,
        priority: unit.priority,
        target_time: unit.target_time ? unit.target_time.slice(0, 10) : '',
        target_value: unit.target_value || 100,
        step: unit.step || 1,
        unit_label: unit.unit_label || '',
        allow_exceed: unit.allow_exceed || false,
      },
    });
  },

  closeDialog() { this.setData({ dialogVisible: false }); },

  onTitleInput(e) { this.setData({ 'unitForm.title': e.detail.value }); },
  onDescInput(e) { this.setData({ 'unitForm.description': e.detail.value }); },
  onTypeChange(e) { this.setData({ 'unitForm.type': this.data.typeOptions[e.detail.value].value }); },
  onPriorityChange(e) { this.setData({ 'unitForm.priority': this.data.priorityOptions[e.detail.value].value }); },
  onTargetTimeChange(e) { this.setData({ 'unitForm.target_time': e.detail.value }); },
  onTargetValueInput(e) { this.setData({ 'unitForm.target_value': parseInt(e.detail.value) || 0 }); },
  onStepInput(e) { this.setData({ 'unitForm.step': parseInt(e.detail.value) || 1 }); },
  onUnitLabelInput(e) { this.setData({ 'unitForm.unit_label': e.detail.value }); },

  async saveUnit() {
    const { unitForm, editingUnit } = this.data;
    if (!unitForm.title.trim()) {
      wx.showToast({ title: '请输入标题', icon: 'none' });
      return;
    }
    this.setData({ saving: true });
    try {
      const payload = {
        title: unitForm.title,
        description: unitForm.description,
        type: unitForm.type,
        priority: unitForm.priority,
      };
      if (unitForm.type === 'time_countdown' && unitForm.target_time) payload.target_time = unitForm.target_time + 'T00:00:00Z';
      if (unitForm.type.startsWith('count')) {
        payload.target_value = unitForm.target_value;
        payload.step = unitForm.step;
        payload.unit_label = unitForm.unit_label;
        payload.allow_exceed = unitForm.allow_exceed;
      }

      if (editingUnit) {
        await put(`/units/${editingUnit.id}`, payload);
      } else {
        await post('/units', payload);
      }
      this.setData({ dialogVisible: false });
      this.fetchUnits();
      wx.showToast({ title: editingUnit ? '已更新' : '已创建', icon: 'success' });
    } catch (err) {
      wx.showToast({ title: err.message || '保存失败', icon: 'none' });
    } finally {
      this.setData({ saving: false });
    }
  },

  goDetail(e) {
    const id = e.currentTarget.dataset.id;
    wx.navigateTo({ url: `/pages/unit-detail/index?id=${id}` });
  },

  async toggleStatus(e) {
    const { id, status } = e.currentTarget.dataset;
    const newStatus = status === 'active' ? 'paused' : 'active';
    try {
      await patch(`/units/${id}/status`, { status: newStatus });
      this.fetchUnits();
    } catch (err) {
      wx.showToast({ title: '操作失败', icon: 'none' });
    }
  },
});
