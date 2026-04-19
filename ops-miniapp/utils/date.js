/**
 * 日期/时间格式化工具
 */
const APP_TIMEZONE = 'Asia/Shanghai';

function getParts(dateLike, options = {}) {
  const date = typeof dateLike === 'string' && /^\d{4}-\d{2}-\d{2}$/.test(dateLike)
    ? new Date(`${dateLike}T00:00:00+08:00`)
    : new Date(dateLike);
  const formatter = new Intl.DateTimeFormat('zh-CN', {
    timeZone: APP_TIMEZONE,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: options.withTime ? '2-digit' : undefined,
    minute: options.withTime ? '2-digit' : undefined,
    weekday: options.withWeekday ? 'short' : undefined,
    hour12: false,
  });
  const parts = {};
  for (const part of formatter.formatToParts(date)) {
    if (part.type !== 'literal') parts[part.type] = part.value;
  }
  return parts;
}

function shanghaiNowParts() {
  return getParts(new Date(), { withTime: true });
}

function toShanghaiDateTimeInput(dateStr) {
  if (!dateStr) return '';
  const parts = getParts(dateStr, { withTime: true });
  return `${parts.year}-${parts.month}-${parts.day}T${parts.hour}:${parts.minute}`;
}

function toShanghaiApiDateTime(dateTimeInput) {
  if (!dateTimeInput) return '';
  return `${dateTimeInput}:00+08:00`;
}

function toShanghaiApiDateStart(dateInput) {
  if (!dateInput) return '';
  return `${dateInput}T00:00:00+08:00`;
}

function formatDate(dateStr) {
  if (!dateStr) return '';
  const parts = getParts(dateStr);
  return `${parts.year}-${parts.month}-${parts.day}`;
}

function formatDateTime(dateStr) {
  if (!dateStr) return '';
  const parts = getParts(dateStr, { withTime: true });
  return `${parts.year}-${parts.month}-${parts.day} ${parts.hour}:${parts.minute}`;
}

function formatTime(dateStr) {
  if (!dateStr) return '';
  const parts = getParts(dateStr, { withTime: true });
  return `${parts.hour}:${parts.minute}`;
}

function formatMonthDay(dateStr) {
  if (!dateStr) return '';
  const parts = getParts(dateStr);
  return `${Number(parts.month)}月${Number(parts.day)}日`;
}

function formatDuration(seconds) {
  if (!seconds || seconds <= 0) return '已超期';
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const mins = Math.floor((seconds % 3600) / 60);
  if (days > 0) return `${days}天${hours}小时`;
  if (hours > 0) return `${hours}小时${mins}分钟`;
  return `${mins}分钟`;
}

function formatAmount(num) {
  if (num === undefined || num === null) return '0.00';
  return Number(num).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

function getWeekday(dateStr) {
  if (!dateStr) return '';
  const parts = getParts(dateStr, { withWeekday: true });
  return parts.weekday;
}

function isToday(dateStr) {
  if (!dateStr) return false;
  return formatDate(dateStr) === formatDate(new Date());
}

function currentMonthStart() {
  const now = shanghaiNowParts();
  return `${now.year}-${now.month}-01`;
}

function currentMonthEnd() {
  const now = shanghaiNowParts();
  const last = new Date(`${now.year}-${now.month}-01T00:00:00+08:00`);
  last.setMonth(last.getMonth() + 1, 0);
  return formatDate(last);
}

function addMonths(dateStr, n) {
  const d = new Date(dateStr + '-01');
  d.setMonth(d.getMonth() + n);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
}

module.exports = {
  formatDate,
  formatDateTime,
  formatTime,
  formatMonthDay,
  formatDuration,
  formatAmount,
  getWeekday,
  isToday,
  currentMonthStart,
  currentMonthEnd,
  addMonths,
  toShanghaiDateTimeInput,
  toShanghaiApiDateTime,
  toShanghaiApiDateStart,
};
