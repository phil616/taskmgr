import dayjs from 'dayjs'
import customParseFormat from 'dayjs/plugin/customParseFormat'
import relativeTime from 'dayjs/plugin/relativeTime'
import duration from 'dayjs/plugin/duration'
import timezone from 'dayjs/plugin/timezone'
import utc from 'dayjs/plugin/utc'
import 'dayjs/locale/zh-cn'

export const APP_TIMEZONE = 'Asia/Shanghai'

dayjs.extend(utc)
dayjs.extend(timezone)
dayjs.extend(customParseFormat)
dayjs.extend(relativeTime)
dayjs.extend(duration)
dayjs.locale('zh-cn')
dayjs.tz.setDefault(APP_TIMEZONE)

const dateOnlyPattern = /^\d{4}-\d{2}-\d{2}$/
const localDateTimePattern = /^\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}(:\d{2})?$/

export function parseAppTime(date: string | undefined) {
  if (!date) return null
  if (dateOnlyPattern.test(date)) return dayjs.tz(date, 'YYYY-MM-DD', APP_TIMEZONE)
  if (localDateTimePattern.test(date)) {
    const normalized = date.replace(' ', 'T')
    const format = normalized.length === 16 ? 'YYYY-MM-DDTHH:mm' : 'YYYY-MM-DDTHH:mm:ss'
    return dayjs.tz(normalized, format, APP_TIMEZONE)
  }
  return dayjs(date).tz(APP_TIMEZONE)
}

export function formatDate(date: string | undefined): string {
  if (!date) return '-'
  return parseAppTime(date)?.format('YYYY-MM-DD') || '-'
}

export function formatDateTime(date: string | undefined): string {
  if (!date) return '-'
  return parseAppTime(date)?.format('YYYY-MM-DD HH:mm') || '-'
}

export function toDateInputValue(date: string | undefined): string {
  return parseAppTime(date)?.format('YYYY-MM-DD') || ''
}

export function toDateTimeInputValue(date: string | undefined): string {
  return parseAppTime(date)?.format('YYYY-MM-DDTHH:mm') || ''
}

export function toApiDateTime(dateTimeInput: string | undefined): string {
  if (!dateTimeInput) return ''
  return dayjs.tz(dateTimeInput, 'YYYY-MM-DDTHH:mm', APP_TIMEZONE).format('YYYY-MM-DDTHH:mm:ssZ')
}

export function toApiDateStart(dateInput: string | undefined): string {
  if (!dateInput) return ''
  return dayjs.tz(`${dateInput}T00:00:00`, 'YYYY-MM-DDTHH:mm:ss', APP_TIMEZONE).format('YYYY-MM-DDTHH:mm:ssZ')
}

export function formatDuration(seconds: number): string {
  const abs = Math.abs(seconds)
  const d = Math.floor(abs / 86400)
  const h = Math.floor((abs % 86400) / 3600)
  const m = Math.floor((abs % 3600) / 60)

  const parts: string[] = []
  if (d > 0) parts.push(`${d} 天`)
  if (h > 0) parts.push(`${h} 小时`)
  if (m > 0 || parts.length === 0) parts.push(`${m} 分钟`)

  return parts.join(' ')
}

export function getTimeColor(seconds: number | undefined, isCountdown: boolean): string {
  if (seconds === undefined) return 'primary'
  if (isCountdown) {
    if (seconds <= 0) return 'error'
    if (seconds <= 86400) return 'error'
    if (seconds <= 7 * 86400) return 'warning'
    return 'success'
  }
  return 'primary'
}

export function getPriorityColor(priority: string): string {
  const map: Record<string, string> = {
    critical: 'error',
    high: 'warning',
    normal: 'primary',
    low: 'grey',
  }
  return map[priority] || 'primary'
}

export function getPriorityLabel(priority: string): string {
  const map: Record<string, string> = {
    critical: '紧急',
    high: '高',
    normal: '普通',
    low: '低',
  }
  return map[priority] || priority
}

export function getStatusLabel(status: string): string {
  const map: Record<string, string> = {
    active: '激活',
    paused: '暂停',
    completed: '已完成',
    archived: '已归档',
    pending: '待办',
    in_progress: '进行中',
    done: '已完成',
    cancelled: '已取消',
  }
  return map[status] || status
}

export function getStatusColor(status: string): string {
  const map: Record<string, string> = {
    active: 'success',
    paused: 'warning',
    completed: 'info',
    archived: 'grey',
    pending: 'primary',
    in_progress: 'warning',
    done: 'success',
    cancelled: 'grey',
  }
  return map[status] || 'primary'
}

export function getUnitTypeLabel(type: string): string {
  const map: Record<string, string> = {
    time_countdown: '时间倒计时',
    time_countup: '时间正计时',
    count_countdown: '数值倒计时',
    count_countup: '数值正计时',
  }
  return map[type] || type
}

export function getUnitTypeIcon(type: string): string {
  const map: Record<string, string> = {
    time_countdown: 'mdi-timer-sand',
    time_countup: 'mdi-timer-outline',
    count_countdown: 'mdi-counter',
    count_countup: 'mdi-chart-line',
  }
  return map[type] || 'mdi-help-circle'
}

export { dayjs }
