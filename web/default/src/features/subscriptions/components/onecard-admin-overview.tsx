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
import { useCallback, useEffect, useState } from 'react'
import { RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { formatQuota } from '@/lib/format'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { StatusBadge } from '@/components/status-badge'
import {
  getAdminOneCardStats,
  getAdminSubscriptionOrders,
} from '../api'
import type {
  OneCardProductStats,
  SubscriptionOrderRecord,
} from '../types'

function getProductLabel(productType: string) {
  switch (productType) {
    case 'day_card':
      return '日卡'
    case 'week_card':
      return '周卡'
    case 'month_card':
      return '月卡'
    default:
      return productType || '-'
  }
}

function getStatusVariant(status: string) {
  return status === 'success'
    ? 'success'
    : status === 'pending'
      ? 'warning'
      : 'neutral'
}

export function OneCardAdminOverview() {
  const { t } = useTranslation()
  const [stats, setStats] = useState<OneCardProductStats[]>([])
  const [orders, setOrders] = useState<SubscriptionOrderRecord[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)

  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const [statsRes, ordersRes] = await Promise.all([
        getAdminOneCardStats(),
        getAdminSubscriptionOrders({ p: 1, page_size: 10 }),
      ])
      if (statsRes.success) {
        setStats(statsRes.data?.items || [])
      }
      if (ordersRes.success) {
        setOrders(ordersRes.data?.items || [])
        setTotal(ordersRes.data?.total || 0)
      }
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadData()
  }, [loadData])

  return (
    <div className='mb-4 grid gap-4 xl:grid-cols-[1fr_1.35fr]'>
      <Card>
        <CardHeader>
          <CardTitle>{t('OneCard Sales Overview')}</CardTitle>
          <CardDescription>
            {t('Day, week and month card sales and active quota')}
          </CardDescription>
          <CardAction>
            <Button
              variant='ghost'
              size='icon'
              onClick={loadData}
              disabled={loading}
            >
              <RefreshCw
                className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`}
              />
            </Button>
          </CardAction>
        </CardHeader>
        <CardContent>
          <div className='grid gap-3 sm:grid-cols-3'>
            {stats.map((item) => (
              <div key={item.product_type} className='rounded-lg border p-3'>
                <div className='text-sm font-medium'>
                  {t(getProductLabel(item.product_type))}
                </div>
                <div className='mt-2 text-2xl font-semibold'>
                  {item.order_count}
                </div>
                <div className='text-muted-foreground mt-1 text-xs'>
                  {t('Revenue')}: ${item.order_revenue.toFixed(2)}
                </div>
                <div className='text-muted-foreground mt-1 text-xs'>
                  {t('Active Cards')}: {item.active_card_count}
                </div>
                <div className='text-muted-foreground mt-1 text-xs'>
                  {t('Remaining')}: {formatQuota(item.active_remain)}
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t('Recent Subscription Orders')}</CardTitle>
          <CardDescription>
            {t('{{count}} orders in total', { count: total })}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ID</TableHead>
                <TableHead>{t('User')}</TableHead>
                <TableHead>{t('Plan')}</TableHead>
                <TableHead>{t('Amount')}</TableHead>
                <TableHead>{t('Provider')}</TableHead>
                <TableHead>{t('Status')}</TableHead>
                <TableHead>{t('Created At')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {orders.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className='py-8 text-center'>
                    {loading ? t('Loading...') : t('No Data')}
                  </TableCell>
                </TableRow>
              ) : (
                orders.map((item) => (
                  <TableRow key={item.order.id}>
                    <TableCell>#{item.order.id}</TableCell>
                    <TableCell>
                      {item.user?.display_name ||
                        item.user?.username ||
                        `#${item.order.user_id}`}
                    </TableCell>
                    <TableCell>
                      {item.plan?.title || `#${item.order.plan_id}`}
                    </TableCell>
                    <TableCell>${item.order.money.toFixed(2)}</TableCell>
                    <TableCell>
                      {item.order.payment_provider ||
                        item.order.payment_method ||
                        '-'}
                    </TableCell>
                    <TableCell>
                      <StatusBadge
                        label={item.order.status || '-'}
                        variant={getStatusVariant(item.order.status)}
                        copyable={false}
                      />
                    </TableCell>
                    <TableCell>
                      {item.order.create_time
                        ? new Date(
                            item.order.create_time * 1000
                          ).toLocaleString()
                        : '-'}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
