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
*/

import React, { useCallback, useEffect, useState } from 'react';
import { Button, Card, Table, Tag, Typography } from '@douyinfe/semi-ui';
import { RefreshCw } from 'lucide-react';
import { API, renderQuota } from '../../../helpers';

const { Text } = Typography;

function getProductLabel(productType) {
  switch (productType) {
    case 'day_card':
      return '日卡';
    case 'week_card':
      return '周卡';
    case 'month_card':
      return '月卡';
    default:
      return productType || '-';
  }
}

const OneCardAdminOverview = ({ t }) => {
  const [stats, setStats] = useState([]);
  const [orders, setOrders] = useState([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [statsRes, ordersRes] = await Promise.all([
        API.get('/api/subscription/admin/onecard/stats'),
        API.get('/api/subscription/admin/orders', {
          params: { p: 1, page_size: 10 },
        }),
      ]);
      if (statsRes.data?.success) {
        setStats(statsRes.data.data?.items || []);
      }
      if (ordersRes.data?.success) {
        setOrders(ordersRes.data.data?.items || []);
        setTotal(ordersRes.data.data?.total || 0);
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const columns = [
    { title: 'ID', dataIndex: 'id', render: (_, item) => `#${item.order.id}` },
    {
      title: t('用户'),
      render: (_, item) =>
        item.user?.display_name ||
        item.user?.username ||
        `#${item.order.user_id}`,
    },
    {
      title: t('套餐'),
      render: (_, item) => item.plan?.title || `#${item.order.plan_id}`,
    },
    {
      title: t('金额'),
      render: (_, item) => `$${Number(item.order.money || 0).toFixed(2)}`,
    },
    {
      title: t('支付渠道'),
      render: (_, item) =>
        item.order.payment_provider || item.order.payment_method || '-',
    },
    {
      title: t('状态'),
      render: (_, item) => (
        <Tag color={item.order.status === 'success' ? 'green' : 'grey'}>
          {item.order.status || '-'}
        </Tag>
      ),
    },
    {
      title: t('创建时间'),
      render: (_, item) =>
        item.order.create_time
          ? new Date(item.order.create_time * 1000).toLocaleString()
          : '-',
    },
  ];

  return (
    <div className='grid grid-cols-1 xl:grid-cols-[1fr_1.4fr] gap-3 mb-3'>
      <Card
        className='!rounded-xl'
        title={t('一卡通销售概览')}
        headerExtraContent={
          <Button
            size='small'
            type='tertiary'
            theme='light'
            icon={
              <RefreshCw
                size={14}
                className={loading ? 'animate-spin' : ''}
              />
            }
            onClick={loadData}
            loading={loading}
          />
        }
      >
        <div className='grid grid-cols-1 md:grid-cols-3 gap-3'>
          {stats.map((item) => (
            <div key={item.product_type} className='border rounded-lg p-3'>
              <Text strong>{t(getProductLabel(item.product_type))}</Text>
              <div className='text-2xl font-semibold mt-2'>
                {item.order_count}
              </div>
              <div className='text-xs text-gray-500 mt-1'>
                {t('销售额')}: ${Number(item.order_revenue || 0).toFixed(2)}
              </div>
              <div className='text-xs text-gray-500 mt-1'>
                {t('生效卡')}: {item.active_card_count}
              </div>
              <div className='text-xs text-gray-500 mt-1'>
                {t('剩余额度')}: {renderQuota(item.active_remain || 0)}
              </div>
            </div>
          ))}
        </div>
      </Card>
      <Card
        className='!rounded-xl'
        title={t('最近订阅订单')}
        headerExtraContent={
          <Text type='tertiary' size='small'>
            {t('共')} {total} {t('条')}
          </Text>
        }
      >
        <Table
          size='small'
          rowKey={(item) => item.order.id}
          columns={columns}
          dataSource={orders}
          pagination={false}
          loading={loading}
        />
      </Card>
    </div>
  );
};

export default OneCardAdminOverview;
