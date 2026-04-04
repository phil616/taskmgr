const { get, post, put, del } = require('../../utils/request');

Page({
  data: {
    categories: [],
    loading: false,
    dialogVisible: false,
    editingCat: null,
    catForm: { name: '', type: 'expense', color: '#0052d9', icon: '' },
    saving: false,
    typeOptions: [
      { label: '支出', value: 'expense' },
      { label: '收入', value: 'income' },
      { label: '通用', value: 'both' },
    ],
    colorOptions: ['#0052d9', '#008858', '#e37318', '#d54941', '#7b1fa2', '#0097a7', '#455a64', '#e91e63'],
    typeFilter: '',
  },

  onLoad() {},
  onShow() { this.fetchCategories(); },

  async fetchCategories() {
    this.setData({ loading: true });
    try {
      const resp = await get('/budget/categories', this.data.typeFilter ? { type: this.data.typeFilter } : {});
      this.setData({ categories: resp.data || [], loading: false });
    } catch (err) {
      this.setData({ loading: false });
    }
  },

  onTypeFilterChange(e) {
    this.setData({ typeFilter: ['', 'expense', 'income', 'both'][e.detail.value] || '' });
    this.fetchCategories();
  },

  openCreate() {
    this.setData({
      dialogVisible: true,
      editingCat: null,
      catForm: { name: '', type: 'expense', color: '#0052d9', icon: '' },
    });
  },

  openEdit(e) {
    const cat = e.currentTarget.dataset.cat;
    this.setData({
      dialogVisible: true,
      editingCat: cat,
      catForm: { name: cat.name, type: cat.type, color: cat.color || '#0052d9', icon: cat.icon || '' },
    });
  },

  closeDialog() { this.setData({ dialogVisible: false }); },

  onNameInput(e) { this.setData({ 'catForm.name': e.detail.value }); },
  onTypeChange(e) { this.setData({ 'catForm.type': this.data.typeOptions[e.detail.value].value }); },
  onColorSelect(e) { this.setData({ 'catForm.color': e.currentTarget.dataset.color }); },

  async saveCat() {
    const { catForm, editingCat } = this.data;
    if (!catForm.name.trim()) { wx.showToast({ title: '请输入名称', icon: 'none' }); return; }
    this.setData({ saving: true });
    try {
      if (editingCat) {
        await put(`/budget/categories/${editingCat.id}`, catForm);
      } else {
        await post('/budget/categories', catForm);
      }
      this.setData({ dialogVisible: false });
      this.fetchCategories();
      wx.showToast({ title: editingCat ? '已更新' : '已创建', icon: 'success' });
    } catch (err) {
      wx.showToast({ title: err.message || '保存失败', icon: 'none' });
    } finally {
      this.setData({ saving: false });
    }
  },

  async deleteCat(e) {
    const cat = e.currentTarget.dataset.cat;
    if (cat.is_system) { wx.showToast({ title: '系统分类不可删除', icon: 'none' }); return; }
    wx.showModal({
      title: '删除分类',
      content: `确认删除「${cat.name}」？`,
      success: async (res) => {
        if (res.confirm) {
          try {
            await del(`/budget/categories/${cat.id}`);
            this.fetchCategories();
            wx.showToast({ title: '已删除', icon: 'success' });
          } catch (err) {
            wx.showToast({ title: '删除失败', icon: 'none' });
          }
        }
      },
    });
  },
});
