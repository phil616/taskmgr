const { get, post, put, del } = require('../../utils/request');
const { formatTime, getWeekday, isToday, addMonths } = require('../../utils/date');

Page({
  data: {
    schedules: [],
    groupedSchedules: [],
    loading: false,
    currentMonth: '',    // 'YYYY-MM'
    currentMonthLabel: '',

    // 详情面板
    selectedSchedule: null,
    detailVisible: false,

    // 创建/编辑弹窗
    dialogVisible: false,
    editingSchedule: null,
    scheduleForm: {
      title: '',
      description: '',
      start_time: '',
      end_time: '',
      all_day: false,
      color: '#0052d9',
      location: '',
      status: 'planned',
      recurrence_type: 'none',
      tags: [],
    },
    tagInput: '',
    saving: false,

    colorOptions: ['#0052d9', '#008858', '#e37318', '#d54941', '#7b1fa2', '#0097a7'],
    statusOptions: [
      { label: '计划中', value: 'planned' },
      { label: '进行中', value: 'in_progress' },
      { label: '已完成', value: 'completed' },
      { label: '已取消', value: 'cancelled' },
    ],
    recurrenceOptions: [
      { label: '不重复', value: 'none' },
      { label: '每天', value: 'daily' },
      { label: '每周', value: 'weekly' },
      { label: '每月', value: 'monthly' },
      { label: '每年', value: 'yearly' },
    ],
  },

  onLoad() {
    const now = new Date();
    const currentMonth = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
    this.setData({ currentMonth });
    this.updateMonthLabel();
  },

  onShow() {
    if (typeof this.getTabBar === 'function') {
      this.getTabBar().setData({ selected: 2 });
    }
    this.fetchSchedules();
  },

  onPullDownRefresh() {
    this.fetchSchedules().finally(() => wx.stopPullDownRefresh());
  },

  updateMonthLabel() {
    const [year, month] = this.data.currentMonth.split('-');
    this.setData({ currentMonthLabel: `${year}年 ${month}月` });
  },

  prevMonth() {
    this.setData({ currentMonth: addMonths(this.data.currentMonth, -1) });
    this.updateMonthLabel();
    this.fetchSchedules();
  },

  nextMonth() {
    this.setData({ currentMonth: addMonths(this.data.currentMonth, 1) });
    this.updateMonthLabel();
    this.fetchSchedules();
  },

  goToday() {
    const now = new Date();
    const currentMonth = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
    this.setData({ currentMonth });
    this.updateMonthLabel();
    this.fetchSchedules();
  },

  async fetchSchedules() {
    this.setData({ loading: true });
    try {
      const [year, month] = this.data.currentMonth.split('-');
      const lastDay = new Date(parseInt(year), parseInt(month), 0).getDate();
      const start_date = `${this.data.currentMonth}-01`;
      const end_date = `${this.data.currentMonth}-${String(lastDay).padStart(2, '0')}`;

      const resp = await get('/schedules', { start_date, end_date, page_size: 500 });
      const schedules = Array.isArray(resp.data) ? resp.data : [];
      this.setData({ schedules });
      this.buildGrouped(schedules);
    } catch (err) {
      wx.showToast({ title: err.message || '加载失败', icon: 'none' });
    } finally {
      this.setData({ loading: false });
    }
  },

  buildGrouped(schedules) {
    const map = {};
    for (const s of schedules) {
      const d = s.start_time ? s.start_time.slice(0, 10) : '';
      if (!d) continue;
      if (!map[d]) map[d] = [];
      map[d].push({
        ...s,
        timeText: s.all_day ? '全天' : `${formatTime(s.start_time)} - ${formatTime(s.end_time)}`,
        statusLabel: { planned: '计划中', in_progress: '进行中', completed: '已完成', cancelled: '已取消' }[s.status] || s.status,
        statusColor: { planned: '#366ef4', in_progress: '#008858', completed: '#a6a6a6', cancelled: '#d54941' }[s.status] || '#a6a6a6',
        recurrenceLabel: { daily: '每天', weekly: '每周', monthly: '每月', yearly: '每年' }[s.recurrence_type] || '',
      });
    }
    const grouped = Object.keys(map)
      .sort()
      .map(date => {
        const d = new Date(date);
        return {
          date,
          dateLabel: `${d.getMonth() + 1}月${d.getDate()}日 ${getWeekday(date)}${isToday(date) ? ' (今天)' : ''}`,
          schedules: map[date].sort((a, b) => a.start_time.localeCompare(b.start_time)),
        };
      });
    this.setData({ groupedSchedules: grouped });
  },

  // ---- 详情面板 ----
  openDetail(e) {
    const schedule = e.currentTarget.dataset.schedule;
    this.setData({ selectedSchedule: schedule, detailVisible: true });
  },

  closeDetail() {
    this.setData({ detailVisible: false });
  },

  // ---- 创建/编辑弹窗 ----
  openCreateDialog() {
    const now = new Date();
    const pad = n => String(n).padStart(2, '0');
    const dt = `${now.getFullYear()}-${pad(now.getMonth()+1)}-${pad(now.getDate())}T${pad(now.getHours())}:00`;
    const dt2 = `${now.getFullYear()}-${pad(now.getMonth()+1)}-${pad(now.getDate())}T${pad(now.getHours()+1)}:00`;
    this.setData({
      dialogVisible: true,
      editingSchedule: null,
      scheduleForm: { title: '', description: '', start_time: dt, end_time: dt2, all_day: false, color: '#0052d9', location: '', status: 'planned', recurrence_type: 'none', tags: [] },
      tagInput: '',
    });
  },

  openEditDialog() {
    const s = this.data.selectedSchedule;
    this.setData({
      dialogVisible: true,
      editingSchedule: s,
      detailVisible: false,
      scheduleForm: {
        title: s.title || '',
        description: s.description || '',
        start_time: (s.start_time || '').replace(' ', 'T').slice(0, 16),
        end_time: (s.end_time || '').replace(' ', 'T').slice(0, 16),
        all_day: s.all_day || false,
        color: s.color || '#0052d9',
        location: s.location || '',
        status: s.status || 'planned',
        recurrence_type: s.recurrence_type || 'none',
        tags: s.tags || [],
      },
      tagInput: '',
    });
  },

  closeDialog() {
    this.setData({ dialogVisible: false });
  },

  onTitleInput(e) { this.setData({ 'scheduleForm.title': e.detail.value }); },
  onDescInput(e) { this.setData({ 'scheduleForm.description': e.detail.value }); },
  onLocationInput(e) { this.setData({ 'scheduleForm.location': e.detail.value }); },
  onStartTimeChange(e) { this.setData({ 'scheduleForm.start_time': e.detail.value }); },
  onEndTimeChange(e) { this.setData({ 'scheduleForm.end_time': e.detail.value }); },
  onColorSelect(e) { this.setData({ 'scheduleForm.color': e.currentTarget.dataset.color }); },
  onStatusChange(e) { this.setData({ 'scheduleForm.status': this.data.statusOptions[e.detail.value].value }); },
  onRecurrenceChange(e) { this.setData({ 'scheduleForm.recurrence_type': this.data.recurrenceOptions[e.detail.value].value }); },
  onTagInput(e) { this.setData({ tagInput: e.detail.value }); },

  addTag() {
    const tag = this.data.tagInput.trim();
    if (tag && !this.data.scheduleForm.tags.includes(tag)) {
      this.setData({
        'scheduleForm.tags': [...this.data.scheduleForm.tags, tag],
        tagInput: '',
      });
    }
  },

  removeTag(e) {
    const idx = e.currentTarget.dataset.idx;
    const tags = [...this.data.scheduleForm.tags];
    tags.splice(idx, 1);
    this.setData({ 'scheduleForm.tags': tags });
  },

  async saveSchedule() {
    const { scheduleForm, editingSchedule } = this.data;
    if (!scheduleForm.title.trim()) {
      wx.showToast({ title: '请输入标题', icon: 'none' });
      return;
    }
    this.setData({ saving: true });
    try {
      const payload = {
        title: scheduleForm.title,
        description: scheduleForm.description,
        start_time: scheduleForm.start_time.replace('T', ' ') + ':00',
        end_time: scheduleForm.end_time.replace('T', ' ') + ':00',
        all_day: scheduleForm.all_day,
        color: scheduleForm.color,
        location: scheduleForm.location,
        status: scheduleForm.status,
        recurrence_type: scheduleForm.recurrence_type,
        tags: scheduleForm.tags,
      };

      if (editingSchedule) {
        await put(`/schedules/${editingSchedule.id}`, payload);
      } else {
        await post('/schedules', payload);
      }
      this.setData({ dialogVisible: false });
      this.fetchSchedules();
      wx.showToast({ title: editingSchedule ? '已更新' : '已创建', icon: 'success' });
    } catch (err) {
      wx.showToast({ title: err.message || '保存失败', icon: 'none' });
    } finally {
      this.setData({ saving: false });
    }
  },

  async deleteSchedule() {
    const s = this.data.selectedSchedule || this.data.editingSchedule;
    if (!s) return;
    wx.showModal({
      title: '确认删除',
      content: `删除日程「${s.title}」？`,
      success: async (res) => {
        if (res.confirm) {
          try {
            await del(`/schedules/${s.id}`);
            this.setData({ detailVisible: false, dialogVisible: false });
            this.fetchSchedules();
            wx.showToast({ title: '已删除', icon: 'success' });
          } catch (err) {
            wx.showToast({ title: '删除失败', icon: 'none' });
          }
        }
      },
    });
  },
});
