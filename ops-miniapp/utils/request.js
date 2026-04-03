/**
 * HTTP 请求封装，自动携带 token，处理 401 跳转
 */
const request = (options) => {
  return new Promise((resolve, reject) => {
    const token = wx.getStorageSync('token') || '';
    const serverUrl = wx.getStorageSync('serverUrl') || 'http://localhost:8080';
    const url = `${serverUrl}/api/v1${options.url}`;

    wx.request({
      url,
      method: options.method || 'GET',
      data: options.data,
      header: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      success(res) {
        if (res.statusCode === 401) {
          wx.removeStorageSync('token');
          wx.removeStorageSync('user');
          wx.reLaunch({ url: '/pages/login/login' });
          reject(new Error('未授权，请重新登录'));
          return;
        }
        if (res.statusCode >= 200 && res.statusCode < 300) {
          resolve(res.data);
        } else {
          const msg = res.data && res.data.message ? res.data.message : `请求失败 (${res.statusCode})`;
          reject(new Error(msg));
        }
      },
      fail(err) {
        reject(new Error(err.errMsg || '网络请求失败，请检查服务器地址'));
      },
    });
  });
};

const get = (url, data) => request({ url, method: 'GET', data });
const post = (url, data) => request({ url, method: 'POST', data });
const put = (url, data) => request({ url, method: 'PUT', data });
const patch = (url, data) => request({ url, method: 'PATCH', data });
const del = (url) => request({ url, method: 'DELETE' });

module.exports = { request, get, post, put, patch, del };
