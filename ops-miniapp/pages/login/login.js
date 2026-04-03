const { post } = require('../../utils/request');

Page({
  data: {
    username: '',
    password: '',
    showPassword: false,
    loading: false,
    errorMsg: '',
    serverUrl: '',
    showServerSetting: false,
  },

  onLoad() {
    const serverUrl = wx.getStorageSync('serverUrl') || 'http://localhost:8080';
    this.setData({ serverUrl });
  },

  onUsernameInput(e) {
    this.setData({ username: e.detail.value, errorMsg: '' });
  },

  onPasswordInput(e) {
    this.setData({ password: e.detail.value, errorMsg: '' });
  },

  onServerUrlInput(e) {
    this.setData({ serverUrl: e.detail.value });
  },

  togglePassword() {
    this.setData({ showPassword: !this.data.showPassword });
  },

  toggleServerSetting() {
    this.setData({ showServerSetting: !this.data.showServerSetting });
  },

  saveServerUrl() {
    const url = this.data.serverUrl.trim().replace(/\/$/, '');
    wx.setStorageSync('serverUrl', url);
    this.setData({ showServerSetting: false });
    wx.showToast({ title: '服务器地址已保存', icon: 'success' });
  },

  async handleLogin() {
    const { username, password } = this.data;
    if (!username || !password) {
      this.setData({ errorMsg: '请输入用户名和密码' });
      return;
    }
    this.setData({ loading: true, errorMsg: '' });
    try {
      const resp = await post('/auth/login', { username, password });
      if (resp && resp.data && resp.data.token) {
        wx.setStorageSync('token', resp.data.token);
        wx.setStorageSync('user', JSON.stringify(resp.data.user || {}));
        wx.switchTab({ url: '/pages/dashboard/index' });
      } else {
        this.setData({ errorMsg: '登录失败，返回数据异常' });
      }
    } catch (err) {
      this.setData({ errorMsg: err.message || '登录失败，请检查用户名和密码' });
    } finally {
      this.setData({ loading: false });
    }
  },
});
