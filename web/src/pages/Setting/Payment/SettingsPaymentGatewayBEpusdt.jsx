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
    "code": "usdt.trc20",
    "name": "USDT on TRC20",
    "enabled": true,
    "sort": 1
  }
]`;

export default function SettingsPaymentGatewayBEpusdt(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    BEpusdtBaseURL: '',
    BEpusdtToken: '',
    BEpusdtWebhookSecret: '',
    BEpusdtEnabled: false,
    BEpusdtUSDTNetworks: defaultNetworksJSON,
    BEpusdtOrderTimeout: 1800,
  });
  const [originInputs, setOriginInputs] = useState({});
  const formApiRef = useRef(null);

  useEffect(() => {
    if (!props.options || !formApiRef.current) {
      return;
    }
    const currentInputs = {
      BEpusdtBaseURL: props.options.BEpusdtBaseURL || '',
      BEpusdtToken: props.options.BEpusdtToken || '',
      BEpusdtWebhookSecret: props.options.BEpusdtWebhookSecret || '',
      BEpusdtEnabled: !!props.options.BEpusdtEnabled,
      BEpusdtUSDTNetworks:
        props.options.BEpusdtUSDTNetworks || defaultNetworksJSON,
      BEpusdtOrderTimeout:
        props.options.BEpusdtOrderTimeout !== undefined
          ? Number(props.options.BEpusdtOrderTimeout)
          : 1800,
    };
    setInputs(currentInputs);
    setOriginInputs({ ...currentInputs });
    formApiRef.current.setValues(currentInputs);
  }, [props.options]);

  const getWebhookURL = () => {
    const base =
      props.options.CustomCallbackAddress || props.options.ServerAddress || '';
    if (!base) {
      return '/api/bepusdt/webhook';
    }
    return `${removeTrailingSlash(base)}/api/bepusdt/webhook`;
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
        'BEpusdtBaseURL',
        'BEpusdtToken',
        'BEpusdtWebhookSecret',
        'BEpusdtUSDTNetworks',
      ];

      stringFields.forEach((key) => {
        const value = inputs[key] || '';
        if (key.endsWith('Token')) {
          if (value && originInputs[key] !== value) {
            options.push({ key, value });
          }
          return;
        }
        if (originInputs[key] !== value) {
          options.push({ key, value });
        }
      });

      if (
        Number(originInputs.BEpusdtOrderTimeout) !==
        Number(inputs.BEpusdtOrderTimeout)
      ) {
        options.push({
          key: 'BEpusdtOrderTimeout',
          value: String(inputs.BEpusdtOrderTimeout || 1800),
        });
      }

      if (originInputs.BEpusdtEnabled !== inputs.BEpusdtEnabled) {
        options.push({
          key: 'BEpusdtEnabled',
          value: inputs.BEpusdtEnabled ? 'true' : 'false',
        });
      }

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
        <Form.Section text={t('BEpusdt 设置')}>
          <Text>
            {t(
              'BEpusdt 将创建自托管收银台订单，并通过订单回调完成钱包充值到账。当前建议仅开放你已经准备好的 USDT 链。',
            )}
          </Text>
          <Banner
            type='info'
            description={`${t('Webhook 地址')}: ${getWebhookURL()}`}
          />

          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='BEpusdtBaseURL'
                label={t('服务地址')}
                placeholder='https://pay.example.com'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='BEpusdtToken'
                label='API Token'
                placeholder={t('BEpusdt API Token，敏感信息不显示')}
                type='password'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='BEpusdtWebhookSecret'
                label={t('Webhook 密钥')}
                placeholder={t('独立回调密钥，敏感信息不显示')}
                type='password'
              />
            </Col>
          </Row>

          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Switch field='BEpusdtEnabled' label={t('启用 BEpusdt')} />
            </Col>
          </Row>

          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.InputNumber
                field='BEpusdtOrderTimeout'
                label={t('订单超时秒数')}
                min={120}
                max={86400}
                step={60}
              />
            </Col>
          </Row>

          <Form.TextArea
            field='BEpusdtUSDTNetworks'
            label={t('USDT 网络配置')}
            autosize
            placeholder={defaultNetworksJSON}
            style={{ marginTop: 16 }}
          />

          <Button onClick={submitSettings}>{t('保存 BEpusdt 设置')}</Button>
        </Form.Section>
      </Form>
    </Spin>
  );
}
