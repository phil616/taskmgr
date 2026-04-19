import client from './client'
import type { ApiResponse } from '@/types'

export type BackupImportStrategy = 'merge' | 'overwrite'

export interface BackupImportResult {
  strategy: BackupImportStrategy
  stats: Record<string, number>
}

export const backupApi = {
  export: () =>
    client.get('/backup/export', {
      responseType: 'blob',
    }),

  import: (file: File, strategy: BackupImportStrategy) => {
    const formData = new FormData()
    formData.append('strategy', strategy)
    formData.append('file', file)
    return client.post<ApiResponse<BackupImportResult>>('/backup/import', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    })
  },
}
