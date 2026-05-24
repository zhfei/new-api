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
import { useQuery } from '@tanstack/react-query'
import { Activity, GitBranch, Zap } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { StatusBadge } from '@/components/status-badge'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { Skeleton } from '@/components/ui/skeleton'
import { getOneCardPoolHealth } from '../api'

function formatLatency(value: number) {
  if (!value || value <= 0) return '-'
  return `${Math.round(value)}ms`
}

export function OneCardPoolHealthCard() {
  const { t } = useTranslation()
  const { data, isLoading } = useQuery({
    queryKey: ['onecard-pool-health'],
    queryFn: getOneCardPoolHealth,
    staleTime: 30 * 1000,
  })

  const items = data?.data?.items || []
  const autoOrder = data?.data?.auto_order || ['free', 'plus', 'pro']

  return (
    <Card size='sm'>
      <CardHeader>
        <CardTitle className='flex items-center gap-2'>
          <Activity className='h-4 w-4 text-emerald-500' />
          OneCard {t('Pool Health')}
        </CardTitle>
        <CardDescription>
          {t('Auto pool order')}: {autoOrder.join(' -> ')}
        </CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className='grid grid-cols-1 gap-3 lg:grid-cols-3'>
            {Array.from({ length: 3 }).map((_, index) => (
              <Skeleton key={index} className='h-28 rounded-lg' />
            ))}
          </div>
        ) : (
          <div className='grid grid-cols-1 gap-3 lg:grid-cols-3'>
            {items.map((item) => {
              const enabledPercent =
                item.total > 0
                  ? Math.round((Number(item.enabled || 0) / item.total) * 100)
                  : 0
              return (
                <div
                  key={item.group}
                  className='bg-muted/30 rounded-lg border p-3'
                >
                  <div className='flex items-start justify-between gap-2'>
                    <div>
                      <div className='flex items-center gap-2'>
                        <StatusBadge
                          label={item.group}
                          autoColor={item.group}
                          copyable={false}
                        />
                        <span className='text-muted-foreground text-xs'>
                          {item.health_score}/100
                        </span>
                      </div>
                      <div className='mt-2 flex flex-wrap gap-x-3 gap-y-1 text-xs'>
                        <span>
                          {t('Enabled')}: {item.enabled}/{item.total}
                        </span>
                        <span>
                          {t('Models')}: {item.model_count}
                        </span>
                        <span>
                          {t('Latency')}: {formatLatency(item.avg_response_time)}
                        </span>
                      </div>
                    </div>
                    <Zap className='text-muted-foreground h-4 w-4' />
                  </div>

                  <Progress value={item.health_score} className='mt-3 h-1.5' />

                  <div className='text-muted-foreground mt-2 flex items-center gap-1 text-xs'>
                    <GitBranch className='h-3 w-3' />
                    {t('Available rate')}: {enabledPercent}%
                    {item.auto_disabled > 0
                      ? ` · ${t('Auto disabled')}: ${item.auto_disabled}`
                      : ''}
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
