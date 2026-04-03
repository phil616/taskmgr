export interface ApiResponse<T = any> {
  code: number
  message: string
  data: T
  meta?: PaginationMeta
  errors?: FieldError[]
}

export interface PaginationMeta {
  page: number
  page_size: number
  total: number
  total_pages: number
}

export interface FieldError {
  field: string
  message: string
}

export interface User {
  id: string
  username: string
  display_name: string
  email: string
  created_at: string
  updated_at: string
}

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  user: User
}

export interface TokenResponse {
  api_token: string
}

export interface Unit {
  id: string
  project_id: string | null
  title: string
  description: string
  type: UnitType
  status: UnitStatus
  priority: Priority
  tags: string[]
  color: string

  target_time?: string
  start_time?: string
  display_unit?: string
  remind_before_days?: number[]
  remind_after_days?: number[]

  current_value?: number
  target_value?: number
  step?: number
  unit_label?: string
  allow_exceed?: boolean
  remind_on_values?: number[]

  remaining_seconds?: number
  elapsed_seconds?: number
  progress?: number

  created_at: string
  updated_at: string
}

export type UnitType = 'time_countdown' | 'time_countup' | 'count_countdown' | 'count_countup'
export type UnitStatus = 'active' | 'paused' | 'completed' | 'archived'
export type Priority = 'low' | 'normal' | 'high' | 'critical'

export interface UnitSummary {
  total_active: number
  total_paused: number
  total_completed: number
  total_archived: number
  expiring_count: number
  expired_count: number
}

export interface UnitLog {
  id: string
  unit_id: string
  delta: number
  value_before: number
  value_after: number
  note: string
  operated_at: string
}

export interface Project {
  id: string
  title: string
  description: string
  status: string
  color: string
  icon: string
  sort_order: number
  created_at: string
  updated_at: string
  unit_stats?: ProjectUnitStats
}

export interface ProjectUnitStats {
  active_count: number
  expiring_count: number
  completed_count: number
  total_count: number
}

export interface Todo {
  id: string
  group_id: string | null
  title: string
  description: string
  status: TodoStatus
  priority: Priority
  due_date: string | null
  sort_order: number
  completed_at: string | null
  created_at: string
  updated_at: string
}

export type TodoStatus = 'pending' | 'in_progress' | 'done' | 'cancelled'

export interface TodoGroup {
  id: string
  name: string
  color: string
  sort_order: number
  todo_count: number
  created_at: string
  updated_at: string
}

export interface Notification {
  id: string
  unit_id: string
  level: 'info' | 'warning' | 'critical'
  message: string
  is_read: boolean
  triggered_at: string
  read_at: string | null
  unit_title?: string
}

export interface UnreadCount {
  count: number
}

// ==================== Schedule 日程管理 ====================

export type ScheduleStatus = 'planned' | 'in_progress' | 'completed' | 'cancelled'
export type RecurrenceType = 'none' | 'daily' | 'weekly' | 'monthly' | 'yearly'
export type ResourceType = 'project' | 'todo' | 'unit'

export interface ScheduleResource {
  id: string
  schedule_id: string
  resource_type: ResourceType
  resource_id: string
  note: string
  resource_title?: string
  resource_color?: string
  resource_status?: string
  created_at: string
}

export interface Schedule {
  id: string
  title: string
  description: string
  start_time: string
  end_time: string
  all_day: boolean
  color: string
  location: string
  status: ScheduleStatus
  recurrence_type: RecurrenceType
  recurrence_end: string | null
  tags: string[]
  resources: ScheduleResource[]
  created_at: string
  updated_at: string
}

export interface CreateScheduleRequest {
  title: string
  description?: string
  start_time: string
  end_time: string
  all_day?: boolean
  color?: string
  location?: string
  status?: ScheduleStatus
  recurrence_type?: RecurrenceType
  recurrence_end?: string
  tags?: string[]
}

export interface UpdateScheduleRequest {
  title?: string
  description?: string
  start_time?: string
  end_time?: string
  all_day?: boolean
  color?: string
  location?: string
  status?: ScheduleStatus
  recurrence_type?: RecurrenceType
  recurrence_end?: string
  tags?: string[]
}

export interface AddScheduleResourceRequest {
  resource_type: ResourceType
  resource_id: string
  note?: string
}

export interface ScheduleQueryParams {
  start_date?: string
  end_date?: string
  status?: ScheduleStatus
  page?: number
  page_size?: number
}

// ==================== Budget 预算管理 ====================

export type WalletType = 'bank' | 'cash' | 'credit' | 'alipay' | 'wechat' | 'other'
export type TransactionType = 'income' | 'expense' | 'transfer'
export type CategoryType = 'income' | 'expense' | 'both'

export interface Wallet {
  id: string
  name: string
  type: WalletType
  balance: number
  currency: string
  color: string
  icon: string
  description: string
  is_default: boolean
  sort_order: number
  total_income: number
  total_expense: number
  created_at: string
  updated_at: string
}

export interface CreateWalletRequest {
  name: string
  type?: WalletType
  balance?: number
  currency?: string
  color?: string
  icon?: string
  description?: string
  is_default?: boolean
  sort_order?: number
}

export interface UpdateWalletRequest {
  name?: string
  type?: WalletType
  color?: string
  icon?: string
  description?: string
  is_default?: boolean
  sort_order?: number
}

export interface BudgetCategory {
  id: string
  name: string
  type: CategoryType
  color: string
  icon: string
  sort_order: number
  is_system: boolean
  created_at: string
  updated_at: string
}

export interface CreateCategoryRequest {
  name: string
  type: CategoryType
  color?: string
  icon?: string
  sort_order?: number
}

export interface UpdateCategoryRequest {
  name?: string
  type?: CategoryType
  color?: string
  icon?: string
  sort_order?: number
}

export interface Transaction {
  id: string
  wallet_id: string
  wallet_name: string
  category_id?: string
  category_name: string
  category_icon: string
  category_color: string
  type: TransactionType
  amount: number
  note: string
  tags: string[]
  transaction_at: string
  to_wallet_id?: string
  to_wallet_name?: string
  created_at: string
}

export interface CreateTransactionRequest {
  wallet_id: string
  category_id?: string
  type: TransactionType
  amount: number
  note?: string
  tags?: string[]
  transaction_at: string
  to_wallet_id?: string
}

export interface UpdateTransactionRequest {
  category_id?: string | null  // null 或 '' 均表示清除分类
  amount?: number
  note?: string
  tags?: string[]
  transaction_at?: string
}

export interface TransactionQueryParams {
  wallet_id?: string
  category_id?: string
  type?: TransactionType
  start_date?: string
  end_date?: string
  min_amount?: number
  max_amount?: number
  keyword?: string
  page?: number
  page_size?: number
}

export interface CategoryStatItem {
  category_id: string
  category_name: string
  category_icon: string
  category_color: string
  type: TransactionType
  total: number
  count: number
}

export interface WalletStatResponse {
  total_income: number
  total_expense: number
  net_amount: number
  transaction_count: number
  category_stats: CategoryStatItem[]
}

