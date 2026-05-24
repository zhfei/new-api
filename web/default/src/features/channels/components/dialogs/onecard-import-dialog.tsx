/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { importOneCardChannels } from '../../api'
import { channelsQueryKeys } from '../../lib'

const SAMPLE_ACCOUNTS = `{
  "accounts": [
    {
      "name": "user@example.com",
      "credentials": {
        "access_token": "access-token",
        "refresh_token": "refresh-token",
        "chatgpt_account_id": "chatgpt-account-id"
      },
      "extra": {
        "email": "user@example.com"
      }
    }
  ]
}`

type OneCardImportDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function OneCardImportDialog({
  open,
  onOpenChange,
}: OneCardImportDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [pool, setPool] = useState('free')
  const [provider, setProvider] = useState('codex')
  const [payload, setPayload] = useState(SAMPLE_ACCOUNTS)
  const [isSubmitting, setIsSubmitting] = useState(false)

  const handleSubmit = async () => {
    let parsed: { accounts?: unknown[]; items?: unknown[] }
    try {
      parsed = JSON.parse(payload)
    } catch {
      toast.error(t('Invalid JSON'))
      return
    }

    const accounts = Array.isArray(parsed.accounts)
      ? parsed.accounts
      : Array.isArray(parsed.items)
        ? parsed.items
        : []
    if (accounts.length === 0) {
      toast.error(t('Accounts cannot be empty'))
      return
    }

    setIsSubmitting(true)
    try {
      const res = await importOneCardChannels({
        pool,
        provider,
        accounts,
      })
      if (!res.success) {
        toast.error(res.message || t('Import failed'))
        return
      }
      const created = res.data?.created || 0
      const skipped = res.data?.skipped || 0
      toast.success(
        `OneCard ${t('Import completed')}: ${created} ${t('Created')}, ${skipped} ${t('Skipped')}`
      )
      if (res.data?.errors?.length) {
        toast.warning(res.data.errors.slice(0, 3).join('\n'))
      }
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: channelsQueryKeys.all }),
        queryClient.invalidateQueries({ queryKey: ['onecard-pool-health'] }),
      ])
      onOpenChange(false)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('Import failed'))
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>OneCard {t('Import Accounts')}</DialogTitle>
          <DialogDescription>
            {t(
              'Paste a sub2api-style JSON payload. Accounts will be created as channels in the selected pool.'
            )}
          </DialogDescription>
        </DialogHeader>

        <div className='grid gap-4'>
          <div className='grid grid-cols-1 gap-3 sm:grid-cols-2'>
            <div className='space-y-2'>
              <Label>{t('Pool')}</Label>
              <Select value={pool} onValueChange={setPool}>
                <SelectTrigger className='w-full'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent alignItemWithTrigger={false}>
                  <SelectGroup>
                    {['free', 'plus', 'pro'].map((item) => (
                      <SelectItem key={item} value={item}>
                        {item}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>

            <div className='space-y-2'>
              <Label>{t('Provider')}</Label>
              <Select value={provider} onValueChange={setProvider}>
                <SelectTrigger className='w-full'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent alignItemWithTrigger={false}>
                  <SelectGroup>
                    {['codex', 'openai', 'claude', 'gemini'].map((item) => (
                      <SelectItem key={item} value={item}>
                        {item}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className='space-y-2'>
            <Label>JSON</Label>
            <Textarea
              value={payload}
              onChange={(event) => setPayload(event.target.value)}
              rows={14}
              className='font-mono text-xs'
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant='outline' onClick={() => onOpenChange(false)}>
            {t('Cancel')}
          </Button>
          <Button onClick={handleSubmit} disabled={isSubmitting}>
            {isSubmitting && <Loader2 className='h-4 w-4 animate-spin' />}
            {t('Import')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
