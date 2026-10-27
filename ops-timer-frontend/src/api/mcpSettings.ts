import { get, put } from './client'

export interface MCPSettings {
  enabled: boolean
  path: string
  external_url: string
  server_name: string
  server_version: string
  timeout_seconds: number
  generic_tools_enabled: boolean
  sensitive_tools_enabled: boolean
  api_token_enabled: boolean
  oauth_enabled: boolean
  oauth_issuer: string
  oauth_dynamic_registration: boolean
  updated_by?: string
  updated_at?: string
}

export const mcpSettingsApi = {
  get: () => get<MCPSettings>('/admin/mcp-settings'),
  save: (settings: MCPSettings) => put<MCPSettings>('/admin/mcp-settings', settings),
}
