import client, { get, post, del } from './client'
import type { ApiResponse } from '@/types'

export interface Certificate {
  id: string
  attachment_id: string
  type: 'x509' | 'gpg'
  common_name: string
  subject: string
  issuer: string
  serial_number: string
  fingerprint: string
  algorithm: string
  dns_names: string[] | null
  emails: string[] | null
  user_ids: string[] | null
  not_before: string
  expires_at: string | null
  revoked: boolean
  countdown_unit_id: string | null
}

export interface CertificateAttachment {
  id: string
  filename: string
  content_type: string
  size: number
  created_at: string
  certificates: Certificate[]
}

export interface AttachmentOwner {
  owner_type: 'project' | 'unit' | 'todo' | 'note' | 'schedule'
  owner_id: string
}

export const certificateApi = {
  list: (params?: Record<string, unknown>) => get<CertificateAttachment[]>('/attachments', params),
  async upload(file: File, owner?: AttachmentOwner) {
    const form = new FormData()
    form.append('file', file)
    if (owner) {
      form.append('owner_type', owner.owner_type)
      form.append('owner_id', owner.owner_id)
    }
    const { data } = await client.post<ApiResponse<CertificateAttachment>>('/attachments', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    return data
  },
  async download(attachment: CertificateAttachment) {
    const { data } = await client.get<Blob>(`/attachments/${attachment.id}/download`, { responseType: 'blob' })
    const url = URL.createObjectURL(data)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = attachment.filename
    document.body.appendChild(anchor)
    anchor.click()
    anchor.remove()
    window.setTimeout(() => URL.revokeObjectURL(url), 1000)
  },
  delete: (id: string) => del(`/attachments/${id}`),
  link: (id: string, owner: AttachmentOwner) => post(`/attachments/${id}/links`, owner),
  unlink: (id: string, owner: AttachmentOwner) => client.delete(`/attachments/${id}/links`, { data: owner }),
  countdown: (id: string) => post<{ id: string }>(`/certificates/${id}/countdown`),
}
