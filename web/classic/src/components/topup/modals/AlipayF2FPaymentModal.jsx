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

import React, { useEffect, useRef, useState } from 'react';
import { Button, Modal, Space, Typography } from '@douyinfe/semi-ui';
import { QRCodeSVG } from 'qrcode.react';
import { API, showError, showInfo, showSuccess } from '../../../helpers';

const { Text, Title } = Typography;

const AlipayF2FPaymentModal = ({
  t,
  visible,
  onCancel,
  payment,
  amountLabel,
  onPaid,
}) => {
  const [checking, setChecking] = useState(false);
  const completedTradeNoRef = useRef('');

  const checkStatus = async (silent = false) => {
    if (!payment?.status_url) return;
    if (completedTradeNoRef.current === payment.out_trade_no) return;
    try {
      setChecking(true);
      const res = await API.get(payment.status_url);
      const status = res.data?.data?.status;
      if (status === 'success') {
        completedTradeNoRef.current = payment.out_trade_no;
        showSuccess(t('支付成功'));
        await onPaid?.();
        onCancel?.();
      } else if (!silent) {
        showInfo(t('等待支付确认中'));
      }
    } catch (error) {
      if (!silent) {
        showError(t('查询支付状态失败'));
      }
    } finally {
      setChecking(false);
    }
  };

  useEffect(() => {
    if (!visible || !payment?.status_url) return;
    completedTradeNoRef.current = '';
    checkStatus(true);
    const timer = window.setInterval(() => checkStatus(true), 3000);
    return () => window.clearInterval(timer);
  }, [visible, payment?.status_url, payment?.out_trade_no]);

  return (
    <Modal
      title={t('支付宝当面付')}
      visible={visible}
      onCancel={onCancel}
      footer={null}
      maskClosable={false}
      size='small'
      centered
    >
      {payment?.qr_code ? (
        <Space vertical align='center' style={{ width: '100%' }}>
          <Title heading={6} style={{ margin: 0 }}>
            {t('请使用支付宝扫码支付')}
          </Title>
          <div
            style={{
              padding: 16,
              border: '1px solid var(--semi-color-border)',
              borderRadius: 16,
              background: 'var(--semi-color-bg-0)',
            }}
          >
            <QRCodeSVG value={payment.qr_code} size={220} />
          </div>
          {amountLabel ? <Text strong>{amountLabel}</Text> : null}
          <Text type='tertiary' style={{ wordBreak: 'break-all' }}>
            {payment.out_trade_no}
          </Text>
          <Space style={{ width: '100%', justifyContent: 'center' }}>
            {payment.payment_page_url ? (
              <Button
                onClick={() => window.open(payment.payment_page_url, '_blank')}
              >
                {t('打开支付页')}
              </Button>
            ) : null}
            <Button
              type='primary'
              theme='solid'
              loading={checking}
              onClick={() => checkStatus(false)}
            >
              {checking ? t('查询中') : t('我已支付')}
            </Button>
          </Space>
        </Space>
      ) : (
        <Text type='tertiary'>{t('暂无支付二维码')}</Text>
      )}
    </Modal>
  );
};

export default AlipayF2FPaymentModal;
