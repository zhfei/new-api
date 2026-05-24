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

import React, { useState } from 'react';
import { Banner, Modal, Select, TextArea, Typography } from '@douyinfe/semi-ui';
import { API, showError, showSuccess, showWarning } from '../../../../helpers';

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
}`;

const OneCardImportModal = ({ visible, onCancel, onSuccess, t }) => {
  const [pool, setPool] = useState('free');
  const [provider, setProvider] = useState('codex');
  const [payload, setPayload] = useState(SAMPLE_ACCOUNTS);
  const [loading, setLoading] = useState(false);

  const submit = async () => {
    let parsed;
    try {
      parsed = JSON.parse(payload);
    } catch {
      showError(t('JSON 格式不正确'));
      return;
    }

    const accounts = Array.isArray(parsed?.accounts)
      ? parsed.accounts
      : Array.isArray(parsed?.items)
        ? parsed.items
        : [];
    if (accounts.length === 0) {
      showError(t('账号列表不能为空'));
      return;
    }

    setLoading(true);
    try {
      const res = await API.post('/api/channel/onecard/import', {
        pool,
        provider,
        accounts,
      });
      const { success, message, data } = res?.data || {};
      if (!success) {
        showError(message || t('导入失败'));
        return;
      }
      showSuccess(
        `OneCard ${t('导入完成')}: ${data?.created || 0} ${t('新增')}, ${data?.skipped || 0} ${t('跳过')}`,
      );
      if (data?.errors?.length) {
        showWarning(data.errors.slice(0, 3).join('\n'));
      }
      onSuccess?.();
      onCancel?.();
    } catch (error) {
      showError(error?.message || t('导入失败'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      title={`OneCard ${t('导入账号')}`}
      visible={visible}
      onOk={submit}
      onCancel={onCancel}
      confirmLoading={loading}
      maskClosable={false}
      centered
      size='large'
    >
      <Banner
        type='info'
        closeIcon={null}
        description={t(
          '粘贴 sub2api 风格 JSON，系统会按所选池组创建为独立渠道。',
        )}
        className='mb-4'
      />
      <div className='grid grid-cols-1 md:grid-cols-2 gap-3 mb-4'>
        <div>
          <Typography.Text strong>{t('池组')}</Typography.Text>
          <Select value={pool} onChange={setPool} className='w-full mt-2'>
            {['free', 'plus', 'pro'].map((item) => (
              <Select.Option key={item} value={item}>
                {item}
              </Select.Option>
            ))}
          </Select>
        </div>
        <div>
          <Typography.Text strong>{t('Provider')}</Typography.Text>
          <Select
            value={provider}
            onChange={setProvider}
            className='w-full mt-2'
          >
            {['codex', 'openai', 'claude', 'gemini'].map((item) => (
              <Select.Option key={item} value={item}>
                {item}
              </Select.Option>
            ))}
          </Select>
        </div>
      </div>
      <Typography.Text strong>JSON</Typography.Text>
      <TextArea
        value={payload}
        onChange={setPayload}
        rows={14}
        className='mt-2 font-mono text-xs'
      />
    </Modal>
  );
};

export default OneCardImportModal;
