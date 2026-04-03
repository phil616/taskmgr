import { get, post, put, del } from './client'
import type {
  ApiResponse,
  Wallet,
  CreateWalletRequest,
  UpdateWalletRequest,
  BudgetCategory,
  CreateCategoryRequest,
  UpdateCategoryRequest,
  Transaction,
  CreateTransactionRequest,
  UpdateTransactionRequest,
  TransactionQueryParams,
  WalletStatResponse,
  PaginationMeta,
} from '@/types'

export const walletApi = {
  list: () => get<Wallet[]>('/wallets'),
  get: (id: string) => get<Wallet>(`/wallets/${id}`),
  create: (data: CreateWalletRequest) => post<Wallet>('/wallets', data),
  update: (id: string, data: UpdateWalletRequest) => put<Wallet>(`/wallets/${id}`, data),
  remove: (id: string) => del<null>(`/wallets/${id}`),
}

export const categoryApi = {
  list: (type?: string) => {
    const params = type ? { type } : undefined
    return get<BudgetCategory[]>('/budget/categories', params)
  },
  create: (data: CreateCategoryRequest) => post<BudgetCategory>('/budget/categories', data),
  update: (id: string, data: UpdateCategoryRequest) => put<BudgetCategory>(`/budget/categories/${id}`, data),
  remove: (id: string) => del<null>(`/budget/categories/${id}`),
}

export interface TransactionListResponse extends ApiResponse<Transaction[]> {
  meta?: PaginationMeta
}

export const transactionApi = {
  list: (params: TransactionQueryParams) => get<Transaction[]>('/transactions', params),
  get: (id: string) => get<Transaction>(`/transactions/${id}`),
  create: (data: CreateTransactionRequest) => post<Transaction>('/transactions', data),
  update: (id: string, data: UpdateTransactionRequest) => put<Transaction>(`/transactions/${id}`, data),
  remove: (id: string) => del<null>(`/transactions/${id}`),
}

export const budgetStatApi = {
  get: (params: { wallet_id?: string; start_date?: string; end_date?: string }) =>
    get<WalletStatResponse>('/budget/stats', params),
}
