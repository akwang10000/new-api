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

const { Text } = Typography;

const defaultNetworksJSON = `[
  {
    "code": "USDTTON",
    "name": "USDT on TON",
    "enabled": true,
    "sort": 1
  }
]`;

const defaultCryptoAmountOptionsJSON = '[5, 10, 20, 50, 100]';

export default function SettingsPaymentGatewayNOWPayments(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    NOWPaymentsApiKey: '',
    NOWPaymentsIPNSecret: '',
    NOWPaymentsEnabled: false,
    NOWPaymentsFiatModeEnabled: true,
    NOWPaymentsCryptoModeEnabled: true,
    NOWPaymentsUSDTNetworks: defaultNetworksJSON,
    NOWPaymentsCryptoAmountOptions: defaultCryptoAmountOptionsJSON,
  });
  const [originInputs, setOriginInputs] = useState({});
  const formApiRef = useRef(null);

  useEffect(() => {
    if (!props.options || !formApiRef.current) {
      return;
    }
    const currentInputs = {
      NOWPaymentsApiKey: props.options.NOWPaymentsApiKey || '',
      NOWPaymentsIPNSecret: props.options.NOWPaymentsIPNSecret || '',
      NOWPaymentsEnabled: !!props.options.NOWPaymentsEnabled,
      NOWPaymentsFiatModeEnabled:
        props.options.NOWPaymentsFiatModeEnabled !== undefined
          ? !!props.options.NOWPaymentsFiatModeEnabled
          : true,
      NOWPaymentsCryptoModeEnabled:
        props.options.NOWPaymentsCryptoModeEnabled !== undefined
          ? !!props.options.NOWPaymentsCryptoModeEnabled
          : true,
      NOWPaymentsUSDTNetworks:
        props.options.NOWPaymentsUSDTNetworks || defaultNetworksJSON,
      NOWPaymentsCryptoAmountOptions:
        props.options.NOWPaymentsCryptoAmountOptions ||
        defaultCryptoAmountOptionsJSON,
    };
    setInputs(currentInputs);
    setOriginInputs({ ...currentInputs });
    formApiRef.current.setValues(currentInputs);
  }, [props.options]);

  const getWebhookURL = () => {
    const base =
      props.options.CustomCallbackAddress || props.options.ServerAddress || '';
    if (!base) {
      return '/api/nowpayments/webhook';
    }
    return `${removeTrailingSlash(base)}/api/nowpayments/webhook`;
  };

  const handleFormChange = (values) => {
    setInputs(values);
  };

  const submitSettings = async () => {
    if (props.options.ServerAddress === '') {
      showError(t('请先填写服务器地址'));
      return;
    }

    setLoading(true);
    try {
      const options = [];
      const stringFields = [
        'NOWPaymentsApiKey',
        'NOWPaymentsIPNSecret',
        'NOWPaymentsUSDTNetworks',
        'NOWPaymentsCryptoAmountOptions',
      ];

      stringFields.forEach((key) => {
        const value = inputs[key] || '';
        if (key.endsWith('Key') || key.endsWith('Secret')) {
          if (value && originInputs[key] !== value) {
            options.push({ key, value });
          }
          return;
        }
        if (originInputs[key] !== value) {
          options.push({ key, value });
        }
      });

      [
        'NOWPaymentsEnabled',
        'NOWPaymentsFiatModeEnabled',
        'NOWPaymentsCryptoModeEnabled',
      ].forEach((key) => {
        if (originInputs[key] !== inputs[key]) {
          options.push({
            key,
            value: inputs[key] ? 'true' : 'false',
          });
        }
      });

      if (options.length === 0) {
        showSuccess(t('保存成功'));
        setLoading(false);
        return;
      }

      for (const option of options) {
        const res = await API.put('/api/option/', {
          key: option.key,
          value: option.value,
        });
        if (!res.data.success) {
          showError(res.data.message);
          setLoading(false);
          return;
        }
      }

      showSuccess(t('保存成功'));
      setOriginInputs({ ...inputs });
      props.refresh?.();
    } catch (error) {
      showError(t('保存失败'));
    }
    setLoading(false);
  };

  return (
    <Spin spinning={loading}>
      <Form
        initValues={inputs}
        onValueChange={handleFormChange}
        getFormApi={(api) => (formApiRef.current = api)}
      >
        <Form.Section text={t('NOWPayments 设置')}>
          <Text>
            {t(
              'NOWPayments 会创建托管 invoice 页面，并通过 IPN webhook 回调完成到账。首期建议只开放 USDT on TON。',
            )}
          </Text>
          <Banner
            type='info'
            description={`Webhook 地址：${getWebhookURL()}`}
          />
          <Banner
            type='warning'
            description={t(
              '法币定价模式会受 NOWPayments 最小支付金额限制；小额 USDT 充值建议使用链上币定价模式。',
            )}
          />

          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='NOWPaymentsApiKey'
                label={t('API Key')}
                placeholder={t('NOWPayments API Key，敏感信息不显示')}
                type='password'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='NOWPaymentsIPNSecret'
                label={t('IPN Secret')}
                placeholder={t('用于验证 NOWPayments webhook 的签名密钥')}
                type='password'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Switch
                field='NOWPaymentsEnabled'
                label={t('启用 NOWPayments')}
              />
            </Col>
          </Row>

          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Switch
                field='NOWPaymentsFiatModeEnabled'
                label={t('启用法币定价模式')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Switch
                field='NOWPaymentsCryptoModeEnabled'
                label={t('启用链上币定价模式')}
              />
            </Col>
          </Row>

          <Form.TextArea
            field='NOWPaymentsUSDTNetworks'
            label={t('USDT 网络配置')}
            autosize
            placeholder={defaultNetworksJSON}
            style={{ marginTop: 16 }}
          />

          <Form.TextArea
            field='NOWPaymentsCryptoAmountOptions'
            label={t('链上币金额选项')}
            autosize
            placeholder={defaultCryptoAmountOptionsJSON}
            style={{ marginTop: 16 }}
          />

          <Button onClick={submitSettings}>{t('保存 NOWPayments 设置')}</Button>
        </Form.Section>
      </Form>
    </Spin>
  );
}
