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
import {
  Banner,
  Button,
  Col,
  Form,
  Row,
  Spin,
  Typography,
} from '@douyinfe/semi-ui';
import {
  API,
  removeTrailingSlash,
  showError,
  showSuccess,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';
import { BookOpen, TriangleAlert } from 'lucide-react';

const { Text } = Typography;

export default function SettingsPaymentGatewayAlipayF2F(props) {
  const { t } = useTranslation();
  const sectionTitle = props.hideSectionTitle
    ? undefined
    : t('支付宝当面付设置');
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    AlipayF2FEnabled: false,
    AlipayF2FSandboxEnabled: false,
    AlipayF2FDisplayName: '支付宝当面付',
    AlipayF2FAppId: '',
    AlipayF2FPrivateKey: '',
    AlipayF2FPublicKey: '',
    AlipayF2FGatewayUrl: '',
    AlipayF2FSellerId: '',
    AlipayF2FMinTopUp: 1,
    AlipayF2FTopUpNotifyUrl: '',
    AlipayF2FTopUpReturnUrl: '',
    AlipayF2FSubscriptionNotifyUrl: '',
    AlipayF2FSubscriptionReturnUrl: '',
  });
  const formApiRef = useRef(null);

  useEffect(() => {
    if (props.options && formApiRef.current) {
      const currentInputs = {
        AlipayF2FEnabled: Boolean(props.options.AlipayF2FEnabled),
        AlipayF2FSandboxEnabled: Boolean(props.options.AlipayF2FSandboxEnabled),
        AlipayF2FDisplayName:
          props.options.AlipayF2FDisplayName || '支付宝当面付',
        AlipayF2FAppId: props.options.AlipayF2FAppId || '',
        AlipayF2FPrivateKey: '',
        AlipayF2FPublicKey: props.options.AlipayF2FPublicKey || '',
        AlipayF2FGatewayUrl: props.options.AlipayF2FGatewayUrl || '',
        AlipayF2FSellerId: props.options.AlipayF2FSellerId || '',
        AlipayF2FMinTopUp:
          props.options.AlipayF2FMinTopUp !== undefined
            ? parseFloat(props.options.AlipayF2FMinTopUp)
            : 1,
        AlipayF2FTopUpNotifyUrl: props.options.AlipayF2FTopUpNotifyUrl || '',
        AlipayF2FTopUpReturnUrl: props.options.AlipayF2FTopUpReturnUrl || '',
        AlipayF2FSubscriptionNotifyUrl:
          props.options.AlipayF2FSubscriptionNotifyUrl || '',
        AlipayF2FSubscriptionReturnUrl:
          props.options.AlipayF2FSubscriptionReturnUrl || '',
      };
      setInputs(currentInputs);
      formApiRef.current.setValues(currentInputs);
    }
  }, [props.options]);

  const submitAlipayF2FSetting = async () => {
    setLoading(true);
    try {
      const options = [
        {
          key: 'AlipayF2FEnabled',
          value: inputs.AlipayF2FEnabled ? 'true' : 'false',
        },
        {
          key: 'AlipayF2FSandboxEnabled',
          value: inputs.AlipayF2FSandboxEnabled ? 'true' : 'false',
        },
        {
          key: 'AlipayF2FDisplayName',
          value: inputs.AlipayF2FDisplayName || '',
        },
        { key: 'AlipayF2FAppId', value: inputs.AlipayF2FAppId || '' },
        { key: 'AlipayF2FPublicKey', value: inputs.AlipayF2FPublicKey || '' },
        { key: 'AlipayF2FGatewayUrl', value: inputs.AlipayF2FGatewayUrl || '' },
        { key: 'AlipayF2FSellerId', value: inputs.AlipayF2FSellerId || '' },
        {
          key: 'AlipayF2FMinTopUp',
          value: String(inputs.AlipayF2FMinTopUp || 1),
        },
        {
          key: 'AlipayF2FTopUpNotifyUrl',
          value: inputs.AlipayF2FTopUpNotifyUrl || '',
        },
        {
          key: 'AlipayF2FTopUpReturnUrl',
          value: inputs.AlipayF2FTopUpReturnUrl || '',
        },
        {
          key: 'AlipayF2FSubscriptionNotifyUrl',
          value: inputs.AlipayF2FSubscriptionNotifyUrl || '',
        },
        {
          key: 'AlipayF2FSubscriptionReturnUrl',
          value: inputs.AlipayF2FSubscriptionReturnUrl || '',
        },
      ];

      if (inputs.AlipayF2FPrivateKey) {
        options.push({
          key: 'AlipayF2FPrivateKey',
          value: inputs.AlipayF2FPrivateKey,
        });
      }

      const results = await Promise.all(
        options.map((opt) => API.put('/api/option/', opt)),
      );
      const errorResults = results.filter((res) => !res.data.success);
      if (errorResults.length > 0) {
        errorResults.forEach((res) => showError(res.data.message));
      } else {
        showSuccess(t('更新成功'));
        props.refresh?.();
      }
    } catch (error) {
      showError(t('更新失败'));
    } finally {
      setLoading(false);
    }
  };

  const serverAddress = props.options.ServerAddress
    ? removeTrailingSlash(props.options.ServerAddress)
    : t('网站地址');

  return (
    <Spin spinning={loading}>
      <Form
        initValues={inputs}
        onValueChange={(values) => setInputs(values)}
        getFormApi={(api) => (formApiRef.current = api)}
      >
        <Form.Section text={sectionTitle}>
          <Banner
            type='info'
            icon={<BookOpen size={16} />}
            description={
              <>
                {t('支付宝当面付外部商品标题固定为')}
                <Text strong> 启宝扫码点餐订单 </Text>
                {t('，商品描述固定为')}
                <Text strong> 线下餐饮扫码点餐服务 </Text>
                {t('，避免支付侧风控信息和业务信息不一致。')}
                <br />
                {t('钱包回调地址')}：{serverAddress}
                /api/user/alipay-f2f/notify
                <br />
                {t('订阅回调地址')}：{serverAddress}
                /api/subscription/alipay-f2f/notify
              </>
            }
            style={{ marginBottom: 12 }}
          />
          <Banner
            type='warning'
            icon={<TriangleAlert size={16} />}
            description={t(
              'notify_url 必须是公网 HTTPS 地址；私钥保存后不会回显，留空表示保持当前私钥不变。',
            )}
            style={{ marginBottom: 16 }}
          />

          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Switch
                field='AlipayF2FEnabled'
                checkedText='｜'
                uncheckedText='〇'
                label={t('启用支付宝当面付')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Switch
                field='AlipayF2FSandboxEnabled'
                checkedText='｜'
                uncheckedText='〇'
                label={t('启用沙箱模式')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='AlipayF2FDisplayName'
                label={t('前台展示名称')}
                placeholder={t('支付宝当面付')}
              />
            </Col>
          </Row>

          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input field='AlipayF2FAppId' label='AppID' />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='AlipayF2FSellerId'
                label={t('卖家支付宝用户 ID')}
                placeholder={t('可选，用于回调校验')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.InputNumber
                field='AlipayF2FMinTopUp'
                label={t('最低充值数量')}
                min={1}
                precision={0}
              />
            </Col>
          </Row>

          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.TextArea
                field='AlipayF2FPrivateKey'
                label={t('应用私钥')}
                placeholder={t(
                  '留空表示保持当前不变，支持 PKCS#1/PKCS#8 PEM 或纯 Base64',
                )}
                rows={5}
              />
            </Col>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.TextArea
                field='AlipayF2FPublicKey'
                label={t('支付宝公钥')}
                placeholder={t('用于验签，支持 PEM 或纯 Base64')}
                rows={5}
              />
            </Col>
          </Row>

          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.Input
                field='AlipayF2FGatewayUrl'
                label={t('网关地址')}
                placeholder='https://openapi.alipay.com/gateway.do'
                extraText={t('沙箱模式开启时会自动使用支付宝沙箱网关')}
              />
            </Col>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.Input
                field='AlipayF2FTopUpNotifyUrl'
                label={t('钱包异步通知地址')}
                placeholder={`${serverAddress}/api/user/alipay-f2f/notify`}
              />
            </Col>
          </Row>

          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.Input
                field='AlipayF2FTopUpReturnUrl'
                label={t('钱包同步跳转地址')}
                placeholder={`${serverAddress}/topup?show_history=true`}
              />
            </Col>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.Input
                field='AlipayF2FSubscriptionNotifyUrl'
                label={t('订阅异步通知地址')}
                placeholder={`${serverAddress}/api/subscription/alipay-f2f/notify`}
              />
            </Col>
          </Row>

          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.Input
                field='AlipayF2FSubscriptionReturnUrl'
                label={t('订阅同步跳转地址')}
                placeholder={`${serverAddress}/topup?show_history=true`}
              />
            </Col>
          </Row>

          <Button onClick={submitAlipayF2FSetting}>
            {t('更新支付宝当面付设置')}
          </Button>
        </Form.Section>
      </Form>
    </Spin>
  );
}
