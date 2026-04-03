# TDesign 微信小程序开发指南

> 本文档基于 `tdesign-miniprogram` 官方组件库模板，指导开发者如何在微信小程序项目中高效使用 TDesign 组件体系，包含：新建页面、注册组件、调用 API、交互开发、样式定制等完整流程。

---

## 目录

1. [项目简介](#1-项目简介)
2. [环境准备与项目结构](#2-环境准备与项目结构)
3. [路由配置](#3-路由配置)
4. [如何新建一个页面](#4-如何新建一个页面)
5. [如何使用 TDesign 组件](#5-如何使用-tdesign-组件)
6. [常用组件 API 示例](#6-常用组件-api-示例)
7. [如何创建自定义组件](#7-如何创建自定义组件)
8. [交互开发](#8-交互开发)
9. [样式与主题定制](#9-样式与主题定制)
10. [暗黑模式](#10-暗黑模式)
11. [Skyline 渲染框架支持](#11-skyline-渲染框架支持)
12. [TDesign 组件完整清单](#12-tdesign-组件完整清单)

---

## 1. 项目简介

本项目是腾讯 **TDesign** 微信小程序组件库的官方示例模板，基于 `tdesign-miniprogram` npm 包构建。项目特点：

- 内置 **50+ TDesign UI 组件** 完整演示
- 同时支持 **Skyline** 和 **WebView** 双渲染框架
- 内置 **暗黑模式（Dark Mode）** 支持
- 使用 **CSS 变量**（`--td-*`）统一管理设计 Token
- 支持**分包加载**以优化启动性能

---

## 2. 环境准备与项目结构

### 2.1 环境要求

- 微信开发者工具（最新版）
- 基础库版本 ≥ 3.4.3（Skyline 功能需要）
- 已在工具中构建 npm（`工具 → 构建 npm`）

### 2.2 目录结构

```
CookingClaw/
├── app.js              # 小程序入口，全局逻辑
├── app.json            # 全局配置：路由、全局组件、窗口样式
├── app.wxss            # 全局样式：TDesign CSS 变量定义
├── theme.json          # 暗黑主题 token 配置
├── sitemap.json        # 搜索爬虫配置
├── project.config.json # 开发工具项目配置
├── behaviors/
│   └── skyline.js      # Skyline 渲染检测 Behavior
├── components/
│   ├── demo-block/     # 演示容器组件（t-demo）
│   ├── demo-header/    # 演示页标题组件（t-demo-header）
│   ├── pull-down-list/ # 首页下拉列表组件
│   └── trd-privacy/    # 隐私协议弹窗组件
├── pages/
│   ├── home/           # 首页（组件目录入口）
│   ├── button/         # 按钮组件示例（含 base/、size/、skyline/ 子页）
│   ├── input/          # 输入框示例
│   ├── dialog/         # 对话框示例
│   └── ...             # 每个 TDesign 组件对应一个子目录
└── utils/
    └── gulpError.js    # 构建错误处理工具
```

### 2.3 页面目录约定

每个组件演示页的目录结构如下：

```
pages/button/
├── button.js       # 页面逻辑
├── button.json     # 页面配置（注册子组件）
├── button.wxml     # 页面结构
├── button.wxss     # 页面样式
├── base/           # "基础用法"子组件示例
│   ├── index.js
│   ├── index.json
│   ├── index.wxml
│   └── index.wxss
├── size/           # "尺寸"子组件示例
└── skyline/        # Skyline 渲染版本
    └── button.*
```

---

## 3. 路由配置

### 3.1 主包页面（app.json → pages）

在 `app.json` 的 `pages` 数组中声明路径，格式为 `"pages/<组件名>/<组件名>"`：

```json
{
  "pages": [
    "pages/home/home",
    "pages/button/button",
    "pages/input/input",
    "pages/tabs/tabs"
  ]
}
```

### 3.2 分包页面（app.json → subpackages）

对于使用频率较低或体积较大的页面，使用分包方式减少主包体积：

```json
{
  "subpackages": [
    {
      "root": "pages/dialog/",
      "pages": ["dialog", "skyline/dialog"]
    },
    {
      "root": "pages/calendar/",
      "pages": ["calendar"]
    }
  ]
}
```

> **规则**：分包 `root` 是目录路径（带 `/`），`pages` 里的路径是相对于该 `root` 的路径。

### 3.3 页面跳转

```javascript
// 跳转到指定页面（保留当前页）
wx.navigateTo({ url: '/pages/button/button' });

// 跳转并传递参数
wx.navigateTo({ url: '/pages/input/input?type=search&id=123' });

// 返回上一页
wx.navigateBack();

// 重定向（替换当前页，无法返回）
wx.redirectTo({ url: '/pages/home/home' });
```

---

## 4. 如何新建一个页面

以新建一个名为 `my-page` 的页面为例：

### 第一步：创建页面文件

在 `pages/` 下新建目录 `my-page/`，创建四个标准文件：

**`pages/my-page/my-page.js`**
```javascript
Page({
  data: {
    title: '我的页面',
  },

  onLoad(options) {
    console.log('页面加载', options);
  },

  handleButtonTap() {
    wx.showToast({ title: '点击了按钮' });
  },
});
```

**`pages/my-page/my-page.json`**
```json
{
  "navigationBarTitleText": "我的页面",
  "usingComponents": {
    "t-cell": "tdesign-miniprogram/cell/cell",
    "t-cell-group": "tdesign-miniprogram/cell-group/cell-group"
  }
}
```

**`pages/my-page/my-page.wxml`**
```xml
<t-navbar title="{{title}}" leftArrow />
<view class="demo">
  <t-cell-group title="基础用法">
    <t-cell title="条目一" arrow bind:tap="handleButtonTap" />
    <t-cell title="条目二" note="备注" arrow />
  </t-cell-group>
</view>
```

**`pages/my-page/my-page.wxss`**
```css
.demo {
  padding-bottom: 56rpx;
}
```

### 第二步：注册到路由

在 `app.json` 的 `pages` 数组中添加路径：

```json
{
  "pages": [
    "pages/home/home",
    "pages/my-page/my-page"
  ]
}
```

### 第三步（可选）：设置导航为自定义样式

项目已在 `app.json` 中全局设置了 `"navigationStyle": "custom"`，因此所有页面使用自定义导航栏（`t-navbar` 组件），无需额外配置。

---

## 5. 如何使用 TDesign 组件

### 5.1 全局注册（app.json）

在 `app.json` 的 `usingComponents` 中注册后，所有页面和组件无需重复注册即可直接使用：

```json
{
  "usingComponents": {
    "t-button": "tdesign-miniprogram/button/button",
    "t-icon": "tdesign-miniprogram/icon/icon",
    "t-navbar": "tdesign-miniprogram/navbar/navbar"
  }
}
```

> 本项目已全局注册了 `t-button`、`t-icon`、`t-navbar`，可直接在任何页面中使用。

### 5.2 页面级注册（推荐方式）

在页面 `.json` 文件的 `usingComponents` 中注册，仅当前页面可用，有利于按需加载：

```json
{
  "usingComponents": {
    "t-input": "tdesign-miniprogram/input/input",
    "t-cell": "tdesign-miniprogram/cell/cell",
    "t-dialog": "tdesign-miniprogram/dialog/dialog"
  }
}
```

### 5.3 组件引用路径规则

所有 TDesign 组件的路径格式为：

```
tdesign-miniprogram/<组件目录>/<同名文件>
```

例如：
| 组件 | 注册路径 | WXML 标签 |
|------|----------|-----------|
| 按钮 | `tdesign-miniprogram/button/button` | `<t-button>` |
| 输入框 | `tdesign-miniprogram/input/input` | `<t-input>` |
| 弹出层 | `tdesign-miniprogram/popup/popup` | `<t-popup>` |
| 标签页 | `tdesign-miniprogram/tabs/tabs` | `<t-tabs>` |

---

## 6. 常用组件 API 示例

### 6.1 Button 按钮

```xml
<!-- 主题变体 -->
<t-button theme="primary">主色按钮</t-button>
<t-button theme="light">浅色按钮</t-button>
<t-button theme="danger">危险按钮</t-button>
<t-button theme="default">默认按钮</t-button>

<!-- 类型变体 -->
<t-button variant="base">填充按钮</t-button>
<t-button variant="outline">描边按钮</t-button>
<t-button variant="text">文字按钮</t-button>

<!-- 尺寸 -->
<t-button size="large">大号</t-button>
<t-button size="medium">中号（默认）</t-button>
<t-button size="small">小号</t-button>

<!-- 带图标 -->
<t-button icon="home">带图标按钮</t-button>

<!-- 块级（占满宽度） -->
<t-button block>块级按钮</t-button>

<!-- 禁用状态 -->
<t-button disabled>禁用按钮</t-button>

<!-- 加载状态 -->
<t-button loading>加载中</t-button>

<!-- 事件绑定 -->
<t-button bind:tap="handleTap">点击我</t-button>
```

### 6.2 Input 输入框

```xml
<!-- 基础输入 -->
<t-input label="标签" placeholder="请输入" />

<!-- 带清除按钮 -->
<t-input label="手机号" placeholder="请输入手机号" clearable />

<!-- 密码输入 -->
<t-input label="密码" placeholder="请输入密码" type="password" />

<!-- 带前置图标 -->
<t-input placeholder="搜索" prefixIcon="search" />

<!-- 只读 -->
<t-input label="姓名" value="张三" readonly />

<!-- 自定义 label 插槽 -->
<t-input placeholder="请输入">
  <view slot="label" class="custom-label">自定义标签</view>
</t-input>

<!-- 双向绑定 + 事件 -->
<t-input
  label="备注"
  value="{{inputValue}}"
  bind:change="onInputChange"
  bind:blur="onInputBlur"
/>
```

```javascript
Page({
  data: { inputValue: '' },
  onInputChange(e) {
    this.setData({ inputValue: e.detail.value });
  },
});
```

### 6.3 Toast 轻提示（命令式 API）

Toast 需要在 WXML 中放置占位节点，再通过 JS 调用：

```xml
<!-- 1. 在 WXML 中放置占位符（id 需与 JS 中 selector 一致） -->
<t-toast id="t-toast" />
```

```javascript
// 2. 在页面/组件 json 中注册
// "t-toast": "tdesign-miniprogram/toast/toast"

// 3. 在 JS 中导入并调用
import { Toast } from 'tdesign-miniprogram';

Page({
  showToast() {
    Toast({
      context: this,          // 当前页面/组件实例
      selector: '#t-toast',   // 对应 WXML 中的 id
      message: '操作成功',
    });
  },

  showLoadingToast() {
    Toast({
      context: this,
      selector: '#t-toast',
      message: '加载中...',
      theme: 'loading',
      direction: 'column',    // 图标在文字上方
    });
  },

  showIconToast() {
    Toast({
      context: this,
      selector: '#t-toast',
      message: '操作成功',
      icon: 'check-circle',   // 使用 TDesign 图标名
      direction: 'column',
    });
  },
});
```

### 6.4 Dialog 对话框（命令式 API）

```xml
<!-- WXML 中放置占位符 -->
<t-dialog id="t-dialog" />
```

```javascript
import { Dialog } from 'tdesign-miniprogram';

Page({
  showConfirmDialog() {
    Dialog.confirm({
      context: this,
      selector: '#t-dialog',
      title: '确认操作',
      content: '是否确认执行此操作？',
    }).then((action) => {
      if (action === 'confirm') {
        console.log('用户点击了确认');
      }
    });
  },

  showAlertDialog() {
    Dialog.alert({
      context: this,
      selector: '#t-dialog',
      title: '提示',
      content: '操作已完成',
      confirmBtn: '我知道了',
    });
  },
});
```

### 6.5 Cell / CellGroup 单元格

```xml
<t-cell-group title="个人信息">
  <t-cell title="姓名" note="张三" />
  <t-cell title="手机号" note="138****8888" arrow />
  <t-cell title="邮箱" note="example@mail.com" arrow bind:tap="handleEdit" />
</t-cell-group>

<!-- 带图标 -->
<t-cell title="设置" leftIcon="setting" arrow />

<!-- 带自定义内容插槽 -->
<t-cell title="状态">
  <view slot="right-icon">
    <t-switch checked />
  </view>
</t-cell>
```

### 6.6 Tabs 标签页

```xml
<t-tabs value="{{tabIndex}}" bind:change="onTabChange">
  <t-tab-panel label="全部" value="0">
    <view>全部内容</view>
  </t-tab-panel>
  <t-tab-panel label="进行中" value="1">
    <view>进行中内容</view>
  </t-tab-panel>
  <t-tab-panel label="已完成" value="2">
    <view>已完成内容</view>
  </t-tab-panel>
</t-tabs>
```

```javascript
Page({
  data: { tabIndex: '0' },
  onTabChange(e) {
    this.setData({ tabIndex: e.detail.value });
  },
});
```

### 6.7 Navbar 导航栏

```xml
<!-- 基础导航栏（自定义 navigationStyle 时使用） -->
<t-navbar title="页面标题" leftArrow />

<!-- 带右侧操作 -->
<t-navbar title="页面标题" leftArrow>
  <view slot="right">
    <t-icon name="more" size="48rpx" />
  </view>
</t-navbar>
```

> 因 `app.json` 中设置了 `"navigationStyle": "custom"`，所有页面必须自行放置 `<t-navbar>`。

### 6.8 Icon 图标

```xml
<!-- 基础图标 -->
<t-icon name="home" />

<!-- 自定义大小和颜色 -->
<t-icon name="star-filled" size="48rpx" color="#366ef4" />
```

常用图标名称：`home`、`search`、`person`、`setting`、`close`、`check`、`arrow-right`、`star`、`star-filled`、`heart`、`share`、`more`、`delete`、`edit`

### 6.9 Loading 加载

```xml
<!-- 基础加载动画 -->
<t-loading />

<!-- 带文字 -->
<t-loading text="加载中..." />

<!-- 圆形加载 -->
<t-loading theme="circular" />
```

### 6.10 Badge 徽标

```xml
<!-- 数字徽标 -->
<t-badge count="8">
  <t-icon name="notification" />
</t-badge>

<!-- 红点 -->
<t-badge dot>
  <t-icon name="notification" />
</t-badge>

<!-- 超出最大值 -->
<t-badge count="999" max-count="99">
  <t-icon name="notification" />
</t-badge>
```

### 6.11 Switch 开关

```xml
<t-switch
  checked="{{isEnabled}}"
  bind:change="onSwitchChange"
/>
```

```javascript
Page({
  data: { isEnabled: false },
  onSwitchChange(e) {
    this.setData({ isEnabled: e.detail.value });
  },
});
```

### 6.12 Checkbox / Radio 复选框与单选框

```xml
<!-- 复选框组 -->
<t-checkbox-group value="{{checkedList}}" bind:change="onCheckboxChange">
  <t-checkbox value="apple">苹果</t-checkbox>
  <t-checkbox value="banana">香蕉</t-checkbox>
  <t-checkbox value="orange">橙子</t-checkbox>
</t-checkbox-group>

<!-- 单选框组 -->
<t-radio-group value="{{radioValue}}" bind:change="onRadioChange">
  <t-radio value="male">男</t-radio>
  <t-radio value="female">女</t-radio>
</t-radio-group>
```

```javascript
Page({
  data: {
    checkedList: ['apple'],
    radioValue: 'male',
  },
  onCheckboxChange(e) {
    this.setData({ checkedList: e.detail.value });
  },
  onRadioChange(e) {
    this.setData({ radioValue: e.detail.value });
  },
});
```

---

## 7. 如何创建自定义组件

### 7.1 创建组件文件

在 `components/` 下新建目录，以 `my-card` 为例：

**`components/my-card/index.wxml`**
```xml
<view class="my-card">
  <view class="my-card__header">
    <text class="my-card__title">{{ title }}</text>
    <text wx:if="{{ subtitle }}" class="my-card__subtitle">{{ subtitle }}</text>
  </view>
  <view class="my-card__body">
    <slot />
  </view>
  <view wx:if="{{ showFooter }}" class="my-card__footer">
    <t-button size="small" bind:tap="handleAction">{{ actionText }}</t-button>
  </view>
</view>
```

**`components/my-card/index.js`**
```javascript
Component({
  properties: {
    title: { type: String, value: '' },
    subtitle: { type: String, value: '' },
    showFooter: { type: Boolean, value: false },
    actionText: { type: String, value: '查看详情' },
  },

  data: {
    isExpanded: false,
  },

  methods: {
    handleAction() {
      // 触发自定义事件，向父组件传值
      this.triggerEvent('action', { title: this.data.title });
    },
  },
});
```

**`components/my-card/index.json`**
```json
{
  "component": true,
  "styleIsolation": "apply-shared",
  "usingComponents": {
    "t-button": "tdesign-miniprogram/button/button"
  }
}
```

**`components/my-card/index.wxss`**
```css
.my-card {
  background: var(--td-bg-color-container);
  border-radius: var(--td-radius-default);
  padding: 32rpx;
  margin: 24rpx 32rpx;
  box-shadow: var(--td-shadow-1);
}

.my-card__title {
  font-size: var(--td-font-size-title-medium);
  color: var(--td-text-color-primary);
  font-weight: 600;
}

.my-card__subtitle {
  font-size: var(--td-font-size-body-small);
  color: var(--td-text-color-secondary);
  margin-top: 8rpx;
}

.my-card__body {
  margin-top: 24rpx;
}

.my-card__footer {
  margin-top: 24rpx;
  text-align: right;
}
```

### 7.2 在页面中使用自定义组件

**`pages/my-page/my-page.json`**
```json
{
  "usingComponents": {
    "my-card": "../../components/my-card/index"
  }
}
```

**`pages/my-page/my-page.wxml`**
```xml
<my-card
  title="卡片标题"
  subtitle="副标题内容"
  showFooter
  bind:action="onCardAction"
>
  <text>这是卡片内容，通过 slot 插入</text>
</my-card>
```

**`pages/my-page/my-page.js`**
```javascript
Page({
  onCardAction(e) {
    console.log('卡片操作事件', e.detail.title);
  },
});
```

### 7.3 styleIsolation 样式隔离说明

| 值 | 说明 | 适用场景 |
|---|---|---|
| `isolated`（默认）| 样式完全隔离，互不影响 | 通用组件库组件 |
| `apply-shared` | 页面样式可影响组件，组件样式不影响页面 | 需要继承全局 CSS 变量的自定义组件（**推荐**） |
| `shared` | 样式完全共享，互相影响 | 简单内联组件或演示组件 |

> 使用 TDesign CSS 变量（`--td-*`）时，自定义组件的 `styleIsolation` 应设置为 `apply-shared` 或 `shared`，否则无法读取到 `app.wxss` 中定义的变量。

---

## 8. 交互开发

### 8.1 事件绑定

```xml
<!-- 点击事件 -->
<view bind:tap="handleTap">点击</view>

<!-- 长按事件 -->
<view bind:longpress="handleLongPress">长按</view>

<!-- 阻止事件冒泡 -->
<view catch:tap="handleTap">阻止冒泡的点击</view>

<!-- 传递参数（通过 data- 属性） -->
<view bind:tap="handleItemTap" data-id="{{item.id}}" data-name="{{item.name}}">
  {{item.name}}
</view>
```

```javascript
Page({
  handleItemTap(e) {
    const { id, name } = e.currentTarget.dataset;
    console.log('id:', id, 'name:', name);
  },
});
```

### 8.2 数据绑定与 setData

```javascript
Page({
  data: {
    count: 0,
    userInfo: { name: '张三', age: 18 },
    list: ['a', 'b', 'c'],
  },

  increment() {
    // 更新基础数据
    this.setData({ count: this.data.count + 1 });
  },

  updateName() {
    // 更新嵌套对象属性（使用路径字符串）
    this.setData({ 'userInfo.name': '李四' });
  },

  updateListItem() {
    // 更新数组中的特定元素
    this.setData({ 'list[1]': 'B' });
  },
});
```

### 8.3 列表渲染

```xml
<view wx:for="{{list}}" wx:key="id">
  <t-cell title="{{item.name}}" note="{{item.note}}" />
</view>
```

```javascript
Page({
  data: {
    list: [
      { id: 1, name: '选项一', note: '备注1' },
      { id: 2, name: '选项二', note: '备注2' },
    ],
  },
});
```

### 8.4 条件渲染

```xml
<!-- wx:if：条件为假时节点从 DOM 移除 -->
<view wx:if="{{isLoggedIn}}">已登录内容</view>
<view wx:else>请先登录</view>

<!-- hidden：条件为真时隐藏（节点保留在 DOM） -->
<view hidden="{{!isVisible}}">可见内容</view>
```

### 8.5 使用 Behaviors 复用逻辑

项目内置了 `themeChangeBehavior` 用于监听主题切换：

```javascript
import themeChangeBehavior from 'tdesign-miniprogram/mixins/theme-change';

Page({
  behaviors: [themeChangeBehavior],

  // 引入后，可在 data 中访问 this.data.theme（值为 'light' 或 'dark'）
  // 主题切换时组件会自动重渲染
});
```

---

## 9. 样式与主题定制

### 9.1 TDesign CSS 变量系统

所有设计 Token 以 `--td-` 前缀定义在 `app.wxss` 中，可直接在自定义样式中引用：

```css
/* 颜色 */
var(--td-brand-color)           /* 品牌主色 #0052d9 */
var(--td-text-color-primary)    /* 主要文字色 */
var(--td-text-color-secondary)  /* 次要文字色 */
var(--td-text-color-placeholder)/* 占位符色 */
var(--td-text-color-disabled)   /* 禁用文字色 */

/* 背景 */
var(--td-bg-color-page)         /* 页面背景色 */
var(--td-bg-color-container)    /* 容器背景色（白色/深色） */

/* 圆角 */
var(--td-radius-small)          /* 6rpx */
var(--td-radius-default)        /* 12rpx */
var(--td-radius-large)          /* 18rpx */
var(--td-radius-round)          /* 999px（全圆角） */

/* 字号 */
var(--td-font-size-body-small)  /* 24rpx */
var(--td-font-size-body-medium) /* 28rpx */
var(--td-font-size-title-small) /* 28rpx */
var(--td-font-size-title-medium)/* 32rpx */
var(--td-font-size-title-large) /* 36rpx */

/* 间距 */
var(--td-spacer)                /* 16rpx */
var(--td-spacer-1)              /* 24rpx */
var(--td-spacer-2)              /* 32rpx */
var(--td-spacer-3)              /* 48rpx */

/* 阴影 */
var(--td-shadow-1)              /* 轻阴影 */
var(--td-shadow-2)              /* 中阴影 */
var(--td-shadow-3)              /* 重阴影 */
```

### 9.2 覆盖 TDesign 组件样式

TDesign 组件暴露了 CSS 变量供外部覆盖。在页面中定义即可覆盖该页面内的组件样式：

```css
/* 在页面 .wxss 中覆盖 Navbar 背景色 */
.demo-navbar {
  --td-navbar-bg-color: var(--td-bg-color-container);
  --td-navbar-color: var(--td-text-color-primary);
}

/* 覆盖 Button 主色 */
page {
  --td-button-primary-color: #ff6b00;
}
```

### 9.3 编写响应式样式

使用 `rpx` 单位，微信小程序会根据屏幕宽度自动缩放（750rpx = 屏幕宽度）：

```css
.card {
  width: 686rpx;           /* 左右各 32rpx 边距 */
  padding: 32rpx;
  border-radius: var(--td-radius-default);
  font-size: var(--td-font-size-body-medium);
}
```

---

## 10. 暗黑模式

项目通过 `app.json` 中的 `"darkmode": true` + `app.wxss` 中的 `@media (prefers-color-scheme: dark)` 实现暗黑模式自动切换。

### 自定义组件支持暗黑模式

在自定义组件样式中使用 CSS 变量（而非硬编码颜色值），暗黑模式将自动生效：

```css
/* 推荐：使用变量，暗黑模式自动适配 */
.my-text {
  color: var(--td-text-color-primary);
  background: var(--td-bg-color-container);
}

/* 不推荐：硬编码颜色，暗黑模式下会失效 */
.my-text {
  color: #000000;
  background: #ffffff;
}
```

### 监听主题变化（JS 逻辑层）

```javascript
import themeChangeBehavior from 'tdesign-miniprogram/mixins/theme-change';

Page({
  behaviors: [themeChangeBehavior],

  onLoad() {
    // this.data.theme 值为 'light' 或 'dark'
    console.log('当前主题:', this.data.theme);
  },

  // 主题变化时的处理逻辑（如切换图片资源）
  // themeChangeBehavior 会自动更新 this.data.theme
});
```

```xml
<!-- 根据主题切换图片 -->
<image src="/assets/{{theme === 'dark' ? 'logo_dark' : 'logo_light'}}.png" />
```

---

## 11. Skyline 渲染框架支持

Skyline 是微信小程序新一代渲染引擎，性能优于 WebView。本项目已配置好双框架支持。

### 11.1 配置（已内置，无需修改）

`app.json` 中已启用：
```json
{
  "rendererOptions": {
    "skyline": {
      "disableABTest": true,
      "defaultDisplayBlock": true,
      "defaultContentBox": true,
      "sdkVersionBegin": "3.4.3",
      "sdkVersionEnd": "15.255.255"
    }
  }
}
```

### 11.2 检测当前渲染框架

```javascript
import SkylineBehavior from '@behaviors/skyline.js';

Component({
  behaviors: [SkylineBehavior],

  // 引入后可通过 this.data.skylineRender 判断当前是否为 Skyline 渲染
  methods: {
    handleSomething() {
      if (this.data.skylineRender) {
        // Skyline 特定逻辑
      } else {
        // WebView 逻辑
      }
    },
  },
});
```

```xml
<!-- 根据渲染框架条件显示 -->
<view wx:if="{{!skylineRender}}">
  仅 WebView 下显示的内容
</view>
```

### 11.3 Skyline 页面的布局要点

Skyline 使用 Flex 布局，需要显式设置高度以支持滚动：

```css
/* Skyline 页面容器 */
.skyline {
  display: flex;
  flex-direction: column;
  height: 100vh;
}

/* Skyline 内部滚动区域 */
.skyline .scroll-view {
  flex: 1;
  height: 0; /* 配合 flex: 1 实现填充剩余高度 */
}
```

---

## 12. TDesign 组件完整清单

以下为本项目中已集成的全部 TDesign 组件，均可通过 `tdesign-miniprogram/<组件名>/<组件名>` 路径引用：

### 基础组件
| 组件名 | 标签 | 注册路径 |
|--------|------|----------|
| 按钮 | `<t-button>` | `tdesign-miniprogram/button/button` |
| 图标 | `<t-icon>` | `tdesign-miniprogram/icon/icon` |
| 图片 | `<t-image>` | `tdesign-miniprogram/image/image` |
| 链接 | `<t-link>` | `tdesign-miniprogram/link/link` |
| 分割线 | `<t-divider>` | `tdesign-miniprogram/divider/divider` |

### 导航组件
| 组件名 | 标签 | 注册路径 |
|--------|------|----------|
| 导航栏 | `<t-navbar>` | `tdesign-miniprogram/navbar/navbar` |
| 标签页 | `<t-tabs>` | `tdesign-miniprogram/tabs/tabs` |
| 底部导航 | `<t-tab-bar>` | `tdesign-miniprogram/tab-bar/tab-bar` |
| 侧边栏 | `<t-side-bar>` | `tdesign-miniprogram/side-bar/side-bar` |
| 步骤条 | `<t-steps>` | `tdesign-miniprogram/steps/steps` |
| 返回顶部 | `<t-back-top>` | `tdesign-miniprogram/back-top/back-top` |
| 吸顶容器 | `<t-sticky>` | `tdesign-miniprogram/sticky/sticky` |
| 索引 | `<t-indexes>` | `tdesign-miniprogram/indexes/indexes` |

### 数据录入
| 组件名 | 标签 | 注册路径 |
|--------|------|----------|
| 输入框 | `<t-input>` | `tdesign-miniprogram/input/input` |
| 多行输入 | `<t-textarea>` | `tdesign-miniprogram/textarea/textarea` |
| 复选框 | `<t-checkbox>` | `tdesign-miniprogram/checkbox/checkbox` |
| 单选框 | `<t-radio>` | `tdesign-miniprogram/radio/radio` |
| 开关 | `<t-switch>` | `tdesign-miniprogram/switch/switch` |
| 滑块 | `<t-slider>` | `tdesign-miniprogram/slider/slider` |
| 评分 | `<t-rate>` | `tdesign-miniprogram/rate/rate` |
| 搜索 | `<t-search>` | `tdesign-miniprogram/search/search` |
| 上传 | `<t-upload>` | `tdesign-miniprogram/upload/upload` |
| 选色器 | `<t-color-picker>` | `tdesign-miniprogram/color-picker/color-picker` |
| 选择器 | `<t-picker>` | `tdesign-miniprogram/picker/picker` |
| 日期选择 | `<t-date-time-picker>` | `tdesign-miniprogram/date-time-picker/date-time-picker` |
| 级联选择 | `<t-cascader>` | `tdesign-miniprogram/cascader/cascader` |

### 数据展示
| 组件名 | 标签 | 注册路径 |
|--------|------|----------|
| 单元格 | `<t-cell>` | `tdesign-miniprogram/cell/cell` |
| 单元格组 | `<t-cell-group>` | `tdesign-miniprogram/cell-group/cell-group` |
| 徽标 | `<t-badge>` | `tdesign-miniprogram/badge/badge` |
| 标签 | `<t-tag>` | `tdesign-miniprogram/tag/tag` |
| 进度条 | `<t-progress>` | `tdesign-miniprogram/progress/progress` |
| 倒计时 | `<t-count-down>` | `tdesign-miniprogram/count-down/count-down` |
| 折叠面板 | `<t-collapse>` | `tdesign-miniprogram/collapse/collapse` |
| 宫格 | `<t-grid>` | `tdesign-miniprogram/grid/grid` |
| 轮播图 | `<t-swiper>` | `tdesign-miniprogram/swiper/swiper` |
| 头像 | `<t-avatar>` | `tdesign-miniprogram/avatar/avatar` |
| 图片预览 | `<t-image-viewer>` | `tdesign-miniprogram/image-viewer/image-viewer` |
| 骨架屏 | `<t-skeleton>` | `tdesign-miniprogram/skeleton/skeleton` |
| 树形选择 | `<t-tree-select>` | `tdesign-miniprogram/tree-select/tree-select` |

### 反馈组件
| 组件名 | 标签 | 注册路径 |
|--------|------|----------|
| 轻提示 | `<t-toast>` | `tdesign-miniprogram/toast/toast` |
| 对话框 | `<t-dialog>` | `tdesign-miniprogram/dialog/dialog` |
| 弹出层 | `<t-popup>` | `tdesign-miniprogram/popup/popup` |
| 遮罩层 | `<t-overlay>` | `tdesign-miniprogram/overlay/overlay` |
| 消息通知 | `<t-message>` | `tdesign-miniprogram/message/message` |
| 通知栏 | `<t-notice-bar>` | `tdesign-miniprogram/notice-bar/notice-bar` |
| 加载 | `<t-loading>` | `tdesign-miniprogram/loading/loading` |
| 下拉刷新 | `<t-pull-down-refresh>` | `tdesign-miniprogram/pull-down-refresh/pull-down-refresh` |
| 动作面板 | `<t-action-sheet>` | `tdesign-miniprogram/action-sheet/action-sheet` |
| 抽屉 | `<t-drawer>` | `tdesign-miniprogram/drawer/drawer` |
| 下拉菜单 | `<t-dropdown-menu>` | `tdesign-miniprogram/dropdown-menu/dropdown-menu` |
| 空状态 | `<t-empty>` | `tdesign-miniprogram/empty/empty` |
| 结果页 | `<t-result>` | `tdesign-miniprogram/result/result` |
| 引导 | `<t-guide>` | `tdesign-miniprogram/guide/guide` |

### 布局组件
| 组件名 | 标签 | 注册路径 |
|--------|------|----------|
| 栅格行 | `<t-row>` | `tdesign-miniprogram/row/row` |
| 栅格列 | `<t-col>` | `tdesign-miniprogram/col/col` |
| 过渡动画 | `<t-transition>` | `tdesign-miniprogram/transition/transition` |
| 浮动按钮 | `<t-fab>` | `tdesign-miniprogram/fab/fab` |
| 底部信息 | `<t-footer>` | `tdesign-miniprogram/footer/footer` |
| 滑动单元格 | `<t-swipe-cell>` | `tdesign-miniprogram/swipe-cell/swipe-cell` |
| 日历 | `<t-calendar>` | `tdesign-miniprogram/calendar/calendar` |

---

## 附录：快速上手示例

以下是一个完整的最小可运行页面，展示了导航栏、列表、按钮和 Toast 的综合用法：

**`pages/example/example.json`**
```json
{
  "usingComponents": {
    "t-cell": "tdesign-miniprogram/cell/cell",
    "t-cell-group": "tdesign-miniprogram/cell-group/cell-group",
    "t-toast": "tdesign-miniprogram/toast/toast"
  }
}
```

**`pages/example/example.wxml`**
```xml
<t-navbar title="示例页面" leftArrow class="demo-navbar" />
<view class="demo">
  <t-cell-group title="功能列表">
    <t-cell
      wx:for="{{items}}"
      wx:key="id"
      title="{{item.title}}"
      note="{{item.note}}"
      arrow
      bind:tap="onCellTap"
      data-id="{{item.id}}"
    />
  </t-cell-group>

  <t-button
    block
    theme="primary"
    style="margin: 32rpx"
    bind:tap="showToast"
  >
    显示提示
  </t-button>
</view>

<t-toast id="t-toast" />
```

**`pages/example/example.js`**
```javascript
import { Toast } from 'tdesign-miniprogram';

Page({
  data: {
    items: [
      { id: 1, title: '设置', note: '账号与安全' },
      { id: 2, title: '消息', note: '查看全部消息' },
      { id: 3, title: '帮助', note: '常见问题解答' },
    ],
  },

  onCellTap(e) {
    const { id } = e.currentTarget.dataset;
    console.log('点击了第', id, '项');
  },

  showToast() {
    Toast({
      context: this,
      selector: '#t-toast',
      message: '操作成功！',
      icon: 'check-circle',
      direction: 'column',
    });
  },
});
```
