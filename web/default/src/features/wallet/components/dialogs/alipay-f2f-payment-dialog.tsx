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
import { useEffect, useRef, useState } from 'react'
import { QRCodeSVG } from 'qrcode.react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import type { AlipayF2FPaymentData } from '../../types'

type AlipayF2FPaymentDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  payment: AlipayF2FPaymentData | null
  amountLabel?: string
  onPaid?: () => void | Promise<void>
}

export function AlipayF2FPaymentDialog(props: AlipayF2FPaymentDialogProps) {
  const { t } = useTranslation()
  const [checking, setChecking] = useState(false)
  const completedTradeNoRef = useRef('')
  const { open, onOpenChange, payment, onPaid } = props
  const statusUrl = payment?.status_url
  const tradeNo = payment?.out_trade_no

  useEffect(() => {
    if (!open || !statusUrl || !tradeNo) return

    const checkStatus = async () => {
      if (completedTradeNoRef.current === tradeNo) return
      try {
        setChecking(true)
        const res = await api.get(statusUrl)
        const status = res.data?.data?.status
        if (status === 'success') {
          completedTradeNoRef.current = tradeNo
          toast.success(t('Payment successful'))
          await onPaid?.()
          onOpenChange(false)
        }
      } finally {
        setChecking(false)
      }
    }

    const timer = window.setInterval(checkStatus, 3000)
    void checkStatus()
    return () => window.clearInterval(timer)
  }, [open, statusUrl, tradeNo, onPaid, onOpenChange, t])

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='sm:max-w-sm'>
        <DialogHeader>
          <DialogTitle>{t('Alipay Face-to-Face Payment')}</DialogTitle>
          <DialogDescription>
            {t(
              'Use Alipay to scan the QR code. The page will refresh after payment succeeds.'
            )}
          </DialogDescription>
        </DialogHeader>

        {props.payment ? (
          <div className='space-y-4 text-center'>
            <div className='bg-background mx-auto inline-flex rounded-2xl border p-4'>
              <QRCodeSVG value={props.payment.qr_code} size={220} />
            </div>
            <div className='space-y-1 text-sm'>
              {props.amountLabel && (
                <div className='font-medium'>{props.amountLabel}</div>
              )}
              <div className='text-muted-foreground break-all'>
                {props.payment.out_trade_no}
              </div>
            </div>
            <div className='flex gap-2'>
              {props.payment.payment_page_url && (
                <Button
                  type='button'
                  variant='outline'
                  className='flex-1'
                  onClick={() =>
                    window.open(props.payment?.payment_page_url, '_blank')
                  }
                >
                  {t('Open payment page')}
                </Button>
              )}
              <Button
                type='button'
                className='flex-1'
                disabled={checking}
                onClick={async () => {
                  const res = await api.get(props.payment!.status_url)
                  if (res.data?.data?.status === 'success') {
                    completedTradeNoRef.current = props.payment!.out_trade_no
                    await props.onPaid?.()
                    props.onOpenChange(false)
                  } else {
                    toast.info(t('Waiting for payment confirmation'))
                  }
                }}
              >
                {checking ? t('Checking...') : t('I have paid')}
              </Button>
            </div>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  )
}
