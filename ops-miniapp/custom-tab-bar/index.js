Component({
  data: {
    selected: 0,
    list: [
      { pagePath: '/pages/dashboard/index', text: '首页', icon: 'home', activeIcon: 'home' },
      { pagePath: '/pages/todos/index', text: '待办', icon: 'check-circle', activeIcon: 'check-circle-filled' },
      { pagePath: '/pages/schedule/index', text: '日程', icon: 'calendar', activeIcon: 'calendar' },
      { pagePath: '/pages/budget/index', text: '预算', icon: 'wallet', activeIcon: 'wallet' },
      { pagePath: '/pages/settings/index', text: '我的', icon: 'user', activeIcon: 'user' },
    ],
  },
  methods: {
    switchTab(e) {
      const data = e.currentTarget.dataset;
      const url = data.path;
      wx.switchTab({ url });
    },
  },
});
