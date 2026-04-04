const { get, post, put } = require('../../utils/request');

Page({
  data: {
    user: null,
    profileForm: { username: '', display_name: '', email: '' },
    passwordForm: { old_password: '', new_password: '', confirm_password: '' },
    apiToken: '',
    showToken: false,
    savingProfile: false,
    savingPassword: false,
    regenerating: false,
    testingEmail: false,
    smtpEnabled: false,
    serverUrl: '',
    showServerEdit: false,
    serverUrlInput: '',
  },

  onLoad() {
    const userStr = wx.getStorageSync('user');
    if (userStr) {
      try {
        const user = JSON.parse(userStr);
        this.setData({
          user,
          profileForm: {
            username: user.username || '',
            display_name: user.display_name || '',
            email: user.email || '',
          },
        });
      } catch (e) {}
    }
    const serverUrl = wx.getStorageSync('serverUrl') || 'http://localhost:8080';
    this.setData({ serverUrl, serverUrlInput: serverUrl });
  },

  onShow() {
    if (typeof this.getTabBar === 'function') {
      this.getTabBar().setData({ selected: 4 });
    }
    this.fetchToken();
    this.fetchSmtpStatus();
    this.fetchProfile();
  },

  async fetchProfile() {
    try {
      const resp = await get('/auth/profile');
      const user = resp.data;
      if (user) {
        wx.setStorageSync('user', JSON.stringify(user));
        this.setData({
          user,
          profileForm: {
            username: user.username || '',
            display_name: user.display_name || '',
            email: user.email || '',
          },
        });
      }
    } catch (_) {}
  },

  async fetchToken() {
    try {
      const resp = await get('/auth/token');
      this.setData({ apiToken: resp.data?.api_token || '' });
    } catch (_) {}
  },

  async fetchSmtpStatus() {
    try {
      const resp = await get('/auth/smtp-status');
      this.setData({ smtpEnabled: resp.data?.enabled || false });
    } catch (_) {}
  },

  // ---- 个人信息 ----
  onUsernameInput(e) { this.setData({ 'profileForm.username': e.detail.value }); },
  onDisplayNameInput(e) { this.setData({ 'profileForm.display_name': e.detail.value }); },
  onEmailInput(e) { this.setData({ 'profileForm.email': e.detail.value }); },

  async saveProfile() {
    this.setData({ savingProfile: true });
    try {
      await put('/auth/profile', this.data.profileForm);
      wx.showToast({ title: '个人信息已保存', icon: 'success' });
      this.fetchProfile();
    } catch (err) {
      wx.showToast({ title: err.message || '保存失败', icon: 'none' });
    } finally {
      this.setData({ savingProfile: false });
    }
  },

  // ---- 修改密码 ----
  onOldPwdInput(e) { this.setData({ 'passwordForm.old_password': e.detail.value }); },
  onNewPwdInput(e) { this.setData({ 'passwordForm.new_password': e.detail.value }); },
  onConfirmPwdInput(e) { this.setData({ 'passwordForm.confirm_password': e.detail.value }); },

  async changePassword() {
    const { old_password, new_password, confirm_password } = this.data.passwordForm;
    if (!old_password || !new_password) {
      wx.showToast({ title: '请填写密码信息', icon: 'none' });
      return;
    }
    if (new_password !== confirm_password) {
      wx.showToast({ title: '两次密码不一致', icon: 'none' });
      return;
    }
    this.setData({ savingPassword: true });
    try {
      await put('/auth/password', { old_password, new_password });
      wx.showToast({ title: '密码修改成功', icon: 'success' });
      this.setData({ passwordForm: { old_password: '', new_password: '', confirm_password: '' } });
    } catch (err) {
      wx.showToast({ title: err.message || '修改失败', icon: 'none' });
    } finally {
      this.setData({ savingPassword: false });
    }
  },

  // ---- API Token ----
  toggleToken() {
    this.setData({ showToken: !this.data.showToken });
  },

  async regenerateToken() {
    wx.showModal({
      title: '重新生成 Token',
      content: '重新生成后，旧 Token 将立即失效。确定继续？',
      success: async (res) => {
        if (res.confirm) {
          this.setData({ regenerating: true });
          try {
            const resp = await post('/auth/token/regenerate');
            this.setData({ apiToken: resp.data?.api_token || '' });
            wx.showToast({ title: 'Token 已重新生成', icon: 'success' });
          } catch (err) {
            wx.showToast({ title: '生成失败', icon: 'none' });
          } finally {
            this.setData({ regenerating: false });
          }
        }
      },
    });
  },

  copyToken() {
    wx.setClipboardData({
      data: this.data.apiToken,
      success() { wx.showToast({ title: '已复制到剪贴板', icon: 'success' }); },
    });
  },

  // ---- 邮件测试 ----
  async testEmail() {
    this.setData({ testingEmail: true });
    try {
      const resp = await post('/auth/test-email');
      wx.showToast({ title: resp.data?.message || '测试邮件已发送', icon: 'success' });
    } catch (err) {
      wx.showToast({ title: err.message || '发送失败', icon: 'none' });
    } finally {
      this.setData({ testingEmail: false });
    }
  },

  // ---- 服务器地址 ----
  toggleServerEdit() {
    this.setData({ showServerEdit: !this.data.showServerEdit });
  },
  onServerUrlInput(e) { this.setData({ serverUrlInput: e.detail.value }); },
  saveServerUrl() {
    const url = this.data.serverUrlInput.trim().replace(/\/$/, '');
    wx.setStorageSync('serverUrl', url);
    this.setData({ serverUrl: url, showServerEdit: false });
    wx.showToast({ title: '服务器地址已保存', icon: 'success' });
  },

  // ---- 退出登录 ----
  async logout() {
    wx.showModal({
      title: '退出登录',
      content: '确定要退出吗？',
      success: async (res) => {
        if (res.confirm) {
          try { await post('/auth/logout'); } catch (e) {}
          wx.removeStorageSync('token');
          wx.removeStorageSync('user');
          wx.reLaunch({ url: '/pages/login/login' });
        }
      },
    });
  },
});
