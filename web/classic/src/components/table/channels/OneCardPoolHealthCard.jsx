/*
Copyright (C) 2025 QuantumNous

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

import React, { useEffect, useState } from 'react';
import {
  Button,
  Card,
  Progress,
  Spin,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { API, showError } from '../../../helpers';

const formatLatency = (value) => {
  if (!value || value <= 0) return '-';
  return `${Math.round(value)}ms`;
};

const OneCardPoolHealthCard = ({ refreshKey = 0, t }) => {
  const [loading, setLoading] = useState(false);
  const [items, setItems] = useState([]);
  const [autoOrder, setAutoOrder] = useState(['free', 'plus', 'pro']);

  const loadHealth = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/channel/onecard/health');
      const { success, message, data } = res?.data || {};
      if (!success) {
        showError(message || t('获取 OneCard 池健康状态失败'));
        return;
      }
      setItems(data?.items || []);
      setAutoOrder(data?.auto_order || ['free', 'plus', 'pro']);
    } catch (error) {
      showError(error?.message || t('获取 OneCard 池健康状态失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadHealth();
  }, [refreshKey]);

  return (
    <Card
      className='!rounded-2xl shadow-sm border-0 mb-3'
      title={
        <div className='flex items-center justify-between gap-2'>
          <div>
            <Typography.Text strong>OneCard {t('池健康')}</Typography.Text>
            <div className='text-xs text-gray-500 mt-1'>
              {t('auto 访问顺序')}: {autoOrder.join(' -> ')}
            </div>
          </div>
          <Button size='small' theme='light' onClick={loadHealth}>
            {t('刷新')}
          </Button>
        </div>
      }
    >
      <Spin spinning={loading}>
        <div className='grid grid-cols-1 md:grid-cols-3 gap-3'>
          {items.map((item) => {
            const enabledPercent =
              item.total > 0
                ? Math.round((Number(item.enabled || 0) / item.total) * 100)
                : 0;
            return (
              <div
                key={item.group}
                className='rounded-xl border border-gray-100 p-3 bg-gray-50'
              >
                <div className='flex items-center justify-between gap-2'>
                  <Tag color='green'>{item.group}</Tag>
                  <Typography.Text type='secondary' size='small'>
                    {item.health_score}/100
                  </Typography.Text>
                </div>
                <Progress
                  percent={item.health_score}
                  showInfo={false}
                  size='small'
                  className='my-3'
                />
                <div className='grid grid-cols-2 gap-2 text-xs text-gray-600'>
                  <span>
                    {t('启用')}: {item.enabled}/{item.total}
                  </span>
                  <span>
                    {t('可用率')}: {enabledPercent}%
                  </span>
                  <span>
                    {t('模型')}: {item.model_count}
                  </span>
                  <span>
                    {t('延迟')}: {formatLatency(item.avg_response_time)}
                  </span>
                </div>
                {item.auto_disabled > 0 ? (
                  <div className='text-xs text-orange-500 mt-2'>
                    {t('自动禁用')}: {item.auto_disabled}
                  </div>
                ) : null}
              </div>
            );
          })}
          {!loading && items.length === 0 ? (
            <Typography.Text type='secondary'>
              {t('暂无 OneCard 池数据')}
            </Typography.Text>
          ) : null}
        </div>
      </Spin>
    </Card>
  );
};

export default OneCardPoolHealthCard;
