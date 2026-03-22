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

export default function SettingsPaymentGatewayBTCPay(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    BTCPayServerURL: '',
    BTCPayStoreID: '',
    BTCPayApiToken: '',
    BTCPayWebhookSecret: '',
    BTCPayEnabled: false,
  });
  const [originInputs, setOriginInputs] = useState({});
  const formApiRef = useRef(null);

  useEffect(() => {
    if (props.options && formApiRef.current) {
      const currentInputs = {
        BTCPayServerURL: props.options.BTCPayServerURL || '',
        BTCPayStoreID: props.options.BTCPayStoreID || '',
        BTCPayApiToken: props.options.BTCPayApiToken || '',
        BTCPayWebhookSecret: props.options.BTCPayWebhookSecret || '',
        BTCPayEnabled: !!props.options.BTCPayEnabled,
      };
      setInputs(currentInputs);
      setOriginInputs({ ...currentInputs });
      formApiRef.current.setValues(currentInputs);
    }
  }, [props.options]);

  const handleFormChange = (values) => {
    setInputs(values);
  };

  const getWebhookURL = () => {
    const base =
      props.options.CustomCallbackAddress || props.options.ServerAddress || '';
    if (!base) return t('网站地址') + '/api/btcpay/webhook';
    return `${removeTrailingSlash(base)}/api/btcpay/webhook`;
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
        'BTCPayServerURL',
        'BTCPayStoreID',
        'BTCPayApiToken',
        'BTCPayWebhookSecret',
      ];

      stringFields.forEach((key) => {
        let value = inputs[key] || '';
        if (key === 'BTCPayServerURL') {
          value = removeTrailingSlash(value);
        }
        if (originInputs[key] !== value) {
          options.push({ key, value });
        }
      });

      if (originInputs.BTCPayEnabled !== inputs.BTCPayEnabled) {
        options.push({
          key: 'BTCPayEnabled',
          value: inputs.BTCPayEnabled ? 'true' : 'false',
        });
      }

      if (options.length === 0) {
        showSuccess(t('更新成功'));
        setLoading(false);
        return;
      }

      // Save sequentially so credentials land before the enable flag flips.
      for (const opt of options) {
        const res = await API.put('/api/option/', {
          key: opt.key,
          value: opt.value,
        });
        if (!res.data.success) {
          showError(res.data.message);
          setLoading(false);
          return;
        }
      }

      showSuccess(t('更新成功'));
      setOriginInputs({
        ...inputs,
        BTCPayServerURL: removeTrailingSlash(inputs.BTCPayServerURL || ''),
      });
      props.refresh?.();
    } catch (error) {
      showError(t('更新失败'));
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
        <Form.Section text={t('BTCPay 设置')}>
          <Text>
            {t(
              'BTCPay Server 使用 Greenfield API 创建发票，并依赖你在 BTCPay 商店内配置的 webhook 回调到账。',
            )}
          </Text>
          <Banner type='info' description={`Webhook 地址：${getWebhookURL()}`} />
          <Banner
            type='warning'
            description={t(
              '请在 BTCPay 商店中配置 API Token（至少可创建和读取发票）以及 webhook 密钥。',
            )}
          />
          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.Input
                field='BTCPayServerURL'
                label={t('BTCPay 服务地址')}
                placeholder='https://btcpay.example.com'
              />
            </Col>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.Input
                field='BTCPayStoreID'
                label={t('Store ID')}
                placeholder={t('BTCPay 商店 ID')}
              />
            </Col>
          </Row>
          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='BTCPayApiToken'
                label={t('API Token')}
                placeholder={t('BTCPay API Token，敏感信息不显示')}
                type='password'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='BTCPayWebhookSecret'
                label={t('Webhook 密钥')}
                placeholder={t(
                  '用于验证 BTCPay webhook 的密钥，敏感信息不显示',
                )}
                type='password'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Switch
                field='BTCPayEnabled'
                size='default'
                label={t('启用 BTCPay 充值')}
              />
            </Col>
          </Row>
          <Button onClick={submitSettings}>{t('更新 BTCPay 设置')}</Button>
        </Form.Section>
      </Form>
    </Spin>
  );
}
