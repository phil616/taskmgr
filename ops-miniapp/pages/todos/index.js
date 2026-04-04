const { get, post, put, patch, del } = require('../../utils/request');
const { formatDate } = require('../../utils/date');

Page({
  data: {
    todos: [],
    groups: [],
    loading: false,
    selectedGroup: null, // null=全部, 'none'=未分组, or group_id
    statusFilter: '',
    statusFilterLabel: '全部状态',
    priorityFilter: '',
    priorityFilterLabel: '全部优先级',
    page: 1,
    totalPages: 0,
    totalCount: 0,

    // 新建/编辑 Todo 弹窗
    todoDialogVisible: false,
    editingTodo: null,
    todoForm: { title: '', description: '', priority: 'normal', due_date: '', group_id: '', groupName: '' },

    // 新建分组弹窗
    groupDialogVisible: false,
    groupForm: { name: '', color: '#0052d9' },

    saving: false,

    statusOptions: [
      { label: '全部', value: '' },
      { label: '待办', value: 'pending' },
      { label: '进行中', value: 'in_progress' },
      { label: '已完成', value: 'done' },
      { label: '已取消', value: 'cancelled' },
    ],
    priorityOptions: [
      { label: '全部', value: '' },
      { label: '低', value: 'low' },
      { label: '普通', value: 'normal' },
      { label: '高', value: 'high' },
      { label: '紧急', value: 'critical' },
    ],
    priorityFormOptions: [
      { label: '低', value: 'low' },
      { label: '普通', value: 'normal' },
      { label: '高', value: 'high' },
      { label: '紧急', value: 'critical' },
    ],
  },

  onLoad() {},

  onShow() {
    if (typeof this.getTabBar === 'function') {
      this.getTabBar().setData({ selected: 1 });
    }
    this.fetchGroups();
    this.fetchTodos();
  },

  onPullDownRefresh() {
    this.setData({ page: 1 });
    Promise.all([this.fetchGroups(), this.fetchTodos()])
      .finally(() => wx.stopPullDownRefresh());
  },

  async fetchTodos() {
    this.setData({ loading: true });
    try {
      const params = { page: this.data.page, page_size: 30 };
      if (this.data.selectedGroup === 'none') params.group_id = 'none';
      else if (this.data.selectedGroup) params.group_id = this.data.selectedGroup;
      if (this.data.statusFilter) params.status = this.data.statusFilter;
      if (this.data.priorityFilter) params.priority = this.data.priorityFilter;

      const resp = await get('/todos', params);
      const todos = (resp.data || []).map(t => ({
        ...t,
        dueDateText: t.due_date ? formatDate(t.due_date) : '',
        priorityColor: { low: '#a6a6a6', normal: '#366ef4', high: '#e37318', critical: '#d54941' }[t.priority] || '#a6a6a6',
        priorityLabel: { low: '低', normal: '普通', high: '高', critical: '紧急' }[t.priority] || t.priority,
        statusColor: { pending: '#366ef4', in_progress: '#008858', done: '#a6a6a6', cancelled: '#c5c5c5' }[t.status] || '#a6a6a6',
        statusLabel: { pending: '待办', in_progress: '进行中', done: '已完成', cancelled: '已取消' }[t.status] || t.status,
      }));
      this.setData({
        todos,
        totalPages: resp.meta?.total_pages || 1,
        totalCount: resp.meta?.total || todos.length,
        loading: false,
      });
    } catch (err) {
      this.setData({ loading: false });
      wx.showToast({ title: err.message || '加载失败', icon: 'none' });
    }
  },

  async fetchGroups() {
    try {
      const resp = await get('/todo-groups');
      this.setData({ groups: resp.data || [] });
    } catch (_) {}
  },

  selectGroup(e) {
    const val = e.currentTarget.dataset.val;
    this.setData({ selectedGroup: val, page: 1 });
    this.fetchTodos();
  },

  onStatusChange(e) {
    const opt = this.data.statusOptions[e.detail.value];
    this.setData({ statusFilter: opt.value, statusFilterLabel: opt.label, page: 1 });
    this.fetchTodos();
  },

  onPriorityChange(e) {
    const opt = this.data.priorityOptions[e.detail.value];
    this.setData({ priorityFilter: opt.value, priorityFilterLabel: opt.label, page: 1 });
    this.fetchTodos();
  },

  // ---- Todo 操作 ----
  async toggleTodo(e) {
    const { id, status } = e.currentTarget.dataset;
    const newStatus = status === 'done' ? 'pending' : 'done';
    try {
      await patch(`/todos/${id}/status`, { status: newStatus });
      this.fetchTodos();
      this.fetchGroups();
    } catch (err) {
      wx.showToast({ title: '操作失败', icon: 'none' });
    }
  },

  openCreateTodo() {
    this.setData({
      todoDialogVisible: true,
      editingTodo: null,
      todoForm: { title: '', description: '', priority: 'normal', due_date: '', group_id: '', groupName: '' },
    });
  },

  openEditTodo(e) {
    const todo = e.currentTarget.dataset.todo;
    const grpObj = this.data.groups.find(g => g.id === todo.group_id);
    this.setData({
      todoDialogVisible: true,
      editingTodo: todo,
      todoForm: {
        title: todo.title,
        description: todo.description || '',
        priority: todo.priority || 'normal',
        due_date: todo.due_date ? todo.due_date.slice(0, 10) : '',
        group_id: todo.group_id || '',
        groupName: grpObj ? grpObj.name : '',
      },
    });
  },

  closeTodoDialog() {
    this.setData({ todoDialogVisible: false, editingTodo: null });
  },

  onTodoTitleInput(e) { this.setData({ 'todoForm.title': e.detail.value }); },
  onTodoDescInput(e) { this.setData({ 'todoForm.description': e.detail.value }); },
  onTodoPriorityChange(e) { this.setData({ 'todoForm.priority': e.detail.value }); },
  onTodoDueDateChange(e) { this.setData({ 'todoForm.due_date': e.detail.value }); },
  onTodoGroupChange(e) {
    const grp = this.data.groups[e.detail.value];
    this.setData({ 'todoForm.group_id': grp ? grp.id : '', 'todoForm.groupName': grp ? grp.name : '' });
  },

  async saveTodo() {
    const { todoForm, editingTodo } = this.data;
    if (!todoForm.title.trim()) {
      wx.showToast({ title: '请输入标题', icon: 'none' });
      return;
    }
    this.setData({ saving: true });
    try {
      const payload = { ...todoForm };
      if (!payload.due_date) delete payload.due_date;
      if (!payload.group_id) delete payload.group_id;

      if (editingTodo) {
        await put(`/todos/${editingTodo.id}`, payload);
      } else {
        await post('/todos', payload);
      }
      this.setData({ todoDialogVisible: false });
      this.fetchTodos();
      this.fetchGroups();
      wx.showToast({ title: editingTodo ? '已更新' : '已创建', icon: 'success' });
    } catch (err) {
      wx.showToast({ title: err.message || '保存失败', icon: 'none' });
    } finally {
      this.setData({ saving: false });
    }
  },

  async deleteTodo(e) {
    const id = e.currentTarget.dataset.id;
    wx.showModal({
      title: '确认删除',
      content: '删除后不可恢复',
      success: async (res) => {
        if (res.confirm) {
          try {
            await del(`/todos/${id}`);
            this.fetchTodos();
            this.fetchGroups();
            wx.showToast({ title: '已删除', icon: 'success' });
          } catch (err) {
            wx.showToast({ title: '删除失败', icon: 'none' });
          }
        }
      },
    });
  },

  // ---- 分组操作 ----
  openCreateGroup() {
    this.setData({ groupDialogVisible: true, groupForm: { name: '', color: '#0052d9' } });
  },

  closeGroupDialog() {
    this.setData({ groupDialogVisible: false });
  },

  onGroupNameInput(e) { this.setData({ 'groupForm.name': e.detail.value }); },
  onGroupColorInput(e) { this.setData({ 'groupForm.color': e.detail.value }); },

  async saveGroup() {
    if (!this.data.groupForm.name.trim()) {
      wx.showToast({ title: '请输入分组名称', icon: 'none' });
      return;
    }
    try {
      await post('/todo-groups', this.data.groupForm);
      this.setData({ groupDialogVisible: false });
      this.fetchGroups();
      wx.showToast({ title: '分组已创建', icon: 'success' });
    } catch (err) {
      wx.showToast({ title: err.message || '创建失败', icon: 'none' });
    }
  },

  async deleteGroup(e) {
    const id = e.currentTarget.dataset.id;
    wx.showModal({
      title: '删除分组',
      content: '分组内的待办将移至未分组',
      success: async (res) => {
        if (res.confirm) {
          try {
            await del(`/todo-groups/${id}`);
            if (this.data.selectedGroup === id) this.setData({ selectedGroup: null });
            this.fetchGroups();
            this.fetchTodos();
          } catch (err) {
            wx.showToast({ title: '删除失败', icon: 'none' });
          }
        }
      },
    });
  },
});
