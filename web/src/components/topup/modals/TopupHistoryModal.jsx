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

import React, { useEffect, useMemo, useState } from 'react';
import {
  Badge,
  Button,
  Empty,
  Input,
  Modal,
  Table,
  Tag,
  Toast,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { Coins } from 'lucide-react';
import { IconSearch } from '@douyinfe/semi-icons';
import { API, timestamp2string } from '../../../helpers';
import { isAdmin } from '../../../helpers/utils';
import { useIsMobile } from '../../../hooks/common/useIsMobile';

const { Text } = Typography;

const STATUS_CONFIG = {
  success: { type: 'success', key: '成功' },
  pending: { type: 'warning', key: '待支付' },
  expired: { type: 'danger', key: '已过期' },
};

const PAYMENT_METHOD_MAP = {
  stripe: 'Stripe',
  creem: 'Creem',
  alipay: '支付宝',
  wxpay: '微信',
  btcpay: 'BTCPay',
};

const BEPUSDT_NETWORK_MAP = {
  usdt_trc20: 'USDT on TRC20',
  usdt_bep20: 'USDT on BEP20',
  usdt_erc20: 'USDT on ERC20',
  usdt_polygon: 'USDT on Polygon',
  usdt_arbitrum: 'USDT on Arbitrum',
  usdt_solana: 'USDT on Solana',
};

const NOWPAYMENTS_NETWORK_MAP = {
  usdtton: 'USDT on TON',
  usdttrc20: 'USDT on TRC20',
  usdtbsc: 'USDT on BSC',
  usdtarb: 'USDT on Arbitrum',
};

const isBEpusdtPayment = (paymentMethod) =>
  String(paymentMethod || '').startsWith('bepusdt_');

const isNOWPaymentsPayment = (paymentMethod) =>
  String(paymentMethod || '').startsWith('nowpayments_');

const getNOWPaymentsLabel = (paymentMethod) => {
  const network = String(paymentMethod || '')
    .replace(/^nowpayments_/, '')
    .trim()
    .toLowerCase();
  if (!network) {
    return 'NOWPayments';
  }
  return `NOWPayments · ${NOWPAYMENTS_NETWORK_MAP[network] || network.toUpperCase()}`;
};

const getBEpusdtLabel = (paymentMethod) => {
  const network = String(paymentMethod || '')
    .replace(/^bepusdt_/, '')
    .trim()
    .toLowerCase();
  if (!network) {
    return '虚拟货币支付';
  }
  return `虚拟货币支付 · ${BEPUSDT_NETWORK_MAP[network] || network.toUpperCase()}`;
};

const TopupHistoryModal = ({ visible, onCancel, t }) => {
  const [loading, setLoading] = useState(false);
  const [topups, setTopups] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [keyword, setKeyword] = useState('');

  const isMobile = useIsMobile();

  const loadTopups = async (currentPage, currentPageSize) => {
    setLoading(true);
    try {
      const base = isAdmin() ? '/api/user/topup' : '/api/user/topup/self';
      const qs =
        `p=${currentPage}&page_size=${currentPageSize}` +
        (keyword ? `&keyword=${encodeURIComponent(keyword)}` : '');
      const endpoint = `${base}?${qs}`;
      const res = await API.get(endpoint);
      const { success, message, data } = res.data;
      if (success) {
        setTopups(data.items || []);
        setTotal(data.total || 0);
      } else {
        Toast.error({ content: message || t('加载失败') });
      }
    } catch (error) {
      Toast.error({ content: t('加载充值记录失败') });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (visible) {
      loadTopups(page, pageSize);
    }
  }, [visible, page, pageSize, keyword]);

  const handleAdminComplete = async (tradeNo) => {
    try {
      const res = await API.post('/api/user/topup/complete', {
        trade_no: tradeNo,
      });
      const { success, message } = res.data;
      if (success) {
        Toast.success({ content: t('补单成功') });
        await loadTopups(page, pageSize);
      } else {
        Toast.error({ content: message || t('补单失败') });
      }
    } catch (e) {
      Toast.error({ content: t('补单失败') });
    }
  };

  const confirmAdminComplete = (tradeNo) => {
    Modal.confirm({
      title: t('确认补单'),
      content: t('是否将该订单标记为成功并为用户入账？'),
      onOk: () => handleAdminComplete(tradeNo),
    });
  };

  const renderStatusBadge = (status) => {
    const config = STATUS_CONFIG[status] || { type: 'primary', key: status };
    return (
      <span className='flex items-center gap-2'>
        <Badge dot type={config.type} />
        <span>{t(config.key)}</span>
      </span>
    );
  };

  const renderPaymentMethod = (paymentMethod) => {
    if (isBEpusdtPayment(paymentMethod)) {
      return <Text>{getBEpusdtLabel(paymentMethod)}</Text>;
    }
    if (isNOWPaymentsPayment(paymentMethod)) {
      return <Text>{getNOWPaymentsLabel(paymentMethod)}</Text>;
    }
    const displayName = PAYMENT_METHOD_MAP[paymentMethod];
    return <Text>{displayName ? t(displayName) : paymentMethod || '-'}</Text>;
  };

  const isSubscriptionTopup = (record) => {
    const tradeNo = (record?.trade_no || '').toLowerCase();
    return Number(record?.amount || 0) === 0 && tradeNo.startsWith('sub');
  };

  const renderAmountCell = (topupAmount, record) => {
    if (isSubscriptionTopup(record)) {
      return (
        <Tag color='purple' shape='circle' size='small'>
          {t('订阅套餐')}
        </Tag>
      );
    }

    if (isBEpusdtPayment(record?.payment_method)) {
      return (
        <div className='flex flex-col'>
          <Text>{`¥${Number(topupAmount || 0).toFixed(2)}`}</Text>
          <Text type='tertiary' size='small'>
            {t('实付金额（人民币）')}
          </Text>
        </div>
      );
    }

    return (
      <span className='flex items-center gap-1'>
        <Coins size={16} />
        <Text>{topupAmount}</Text>
      </span>
    );
  };

  const renderSettlementCell = (money, record) => {
    if (isBEpusdtPayment(record?.payment_method)) {
      return (
        <div className='flex flex-col'>
          <Text type='danger'>${Number(money || 0).toFixed(2)}</Text>
          <Text type='tertiary' size='small'>
            {t('到账额度（美元）')}
          </Text>
        </div>
      );
    }

    return <Text type='danger'>${Number(money || 0).toFixed(2)}</Text>;
  };

  const userIsAdmin = useMemo(() => isAdmin(), []);

  const columns = useMemo(() => {
    const baseColumns = [
      {
        title: t('订单号'),
        dataIndex: 'trade_no',
        key: 'trade_no',
        render: (text) => <Text copyable>{text}</Text>,
      },
      {
        title: t('支付方式'),
        dataIndex: 'payment_method',
        key: 'payment_method',
        render: renderPaymentMethod,
      },
      {
        title: t('订单金额'),
        dataIndex: 'amount',
        key: 'amount',
        render: renderAmountCell,
      },
      {
        title: t('结算信息'),
        dataIndex: 'money',
        key: 'money',
        render: renderSettlementCell,
      },
      {
        title: t('状态'),
        dataIndex: 'status',
        key: 'status',
        render: renderStatusBadge,
      },
    ];

    if (userIsAdmin) {
      baseColumns.push({
        title: t('操作'),
        key: 'action',
        render: (_, record) => {
          if (record.status !== 'pending') return null;
          return (
            <Button
              size='small'
              type='primary'
              theme='outline'
              onClick={() => confirmAdminComplete(record.trade_no)}
            >
              {t('补单')}
            </Button>
          );
        },
      });
    }

    baseColumns.push({
      title: t('创建时间'),
      dataIndex: 'create_time',
      key: 'create_time',
      render: (time) => timestamp2string(time),
    });

    return baseColumns;
  }, [t, userIsAdmin]);

  return (
    <Modal
      title={t('充值记录')}
      visible={visible}
      onCancel={onCancel}
      footer={null}
      size={isMobile ? 'full-width' : 'large'}
    >
      <div className='mb-3'>
        <Input
          prefix={<IconSearch />}
          placeholder={t('订单号')}
          value={keyword}
          onChange={(value) => {
            setKeyword(value);
            setPage(1);
          }}
          showClear
        />
      </div>
      <Table
        columns={columns}
        dataSource={topups}
        loading={loading}
        rowKey='id'
        pagination={{
          currentPage: page,
          pageSize,
          total,
          showSizeChanger: true,
          pageSizeOpts: [10, 20, 50, 100],
          onPageChange: setPage,
          onPageSizeChange: (nextPageSize) => {
            setPageSize(nextPageSize);
            setPage(1);
          },
        }}
        size='small'
        empty={
          <Empty
            image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
            darkModeImage={
              <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
            }
            description={t('暂无充值记录')}
            style={{ padding: 30 }}
          />
        }
      />
    </Modal>
  );
};

export default TopupHistoryModal;
