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
  Avatar,
  Banner,
  Button,
  Card,
  Col,
  Form,
  Row,
  Skeleton,
  Space,
  Spin,
  Tabs,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import { SiAlipay, SiStripe, SiWechat } from 'react-icons/si';
import {
  BarChart2,
  Coins,
  CreditCard,
  Receipt,
  Sparkles,
  TrendingUp,
  Wallet,
} from 'lucide-react';
import { IconGift } from '@douyinfe/semi-icons';
import { useMinimumLoadingTime } from '../../hooks/common/useMinimumLoadingTime';
import { getCurrencyConfig } from '../../helpers/render';
import SubscriptionPlansCard from './SubscriptionPlansCard';

const { Text } = Typography;
const { TabPane } = Tabs;

const RechargeCard = ({
  t,
  enableOnlineTopUp,
  enableStripeTopUp,
  enableBTCPayTopUp,
  enableBEpusdtTopUp,
  bepusdtNetworks,
  bepusdtTradeType,
  setBEpusdtTradeType,
  enableNOWPaymentsTopUp,
  nowPaymentsModes,
  nowPaymentsPricingMode,
  setNowPaymentsPricingMode,
  nowPaymentsNetworks,
  nowPaymentsPayCurrency,
  setNowPaymentsPayCurrency,
  nowPaymentsCryptoAmountOptions,
  nowPaymentsCryptoAmount,
  setNowPaymentsCryptoAmount,
  selectedNowPaymentsCryptoAmount,
  setSelectedNowPaymentsCryptoAmount,
  enableCreemTopUp,
  creemProducts,
  creemPreTopUp,
  presetAmounts,
  selectedPreset,
  selectPresetAmount,
  formatLargeNumber,
  priceRatio,
  topUpCount,
  minTopUp,
  renderQuotaWithAmount,
  getAmount,
  setTopUpCount,
  setSelectedPreset,
  renderAmount,
  amountLoading,
  payMethods,
  preTopUp,
  paymentLoading,
  payWay,
  isBEpusdtOnlyMode,
  redemptionCode,
  setRedemptionCode,
  topUp,
  isSubmitting,
  topUpLink,
  openTopUpLink,
  userState,
  renderQuota,
  statusLoading,
  topupInfo,
  onOpenHistory,
  subscriptionLoading = false,
  subscriptionPlans = [],
  billingPreference,
  onChangeBillingPreference,
  activeSubscriptions = [],
  allSubscriptions = [],
  reloadSubscriptionSelf,
}) => {
  const onlineFormApiRef = useRef(null);
  const redeemFormApiRef = useRef(null);
  const initialTabSetRef = useRef(false);
  const showAmountSkeleton = useMinimumLoadingTime(amountLoading);
  const [activeTab, setActiveTab] = useState('topup');

  const shouldShowSubscription =
    !subscriptionLoading && subscriptionPlans.length > 0;
  const isNOWPaymentsCryptoMode =
    enableNOWPaymentsTopUp && nowPaymentsPricingMode === 'crypto';
  const shouldShowAnyTopUpMethod =
    enableOnlineTopUp ||
    enableStripeTopUp ||
    enableBTCPayTopUp ||
    enableBEpusdtTopUp ||
    enableNOWPaymentsTopUp ||
    enableCreemTopUp;
  const shouldShowFiatAmountSection =
    enableOnlineTopUp ||
    enableStripeTopUp ||
    enableBTCPayTopUp ||
    enableBEpusdtTopUp ||
    (enableNOWPaymentsTopUp && nowPaymentsPricingMode === 'fiat');
  const availablePayMethods = (payMethods || []).filter((method) => {
    if (isNOWPaymentsCryptoMode) {
      return method.type === 'nowpayments';
    }
    return true;
  });

  useEffect(() => {
    if (initialTabSetRef.current) return;
    if (subscriptionLoading) return;
    setActiveTab(shouldShowSubscription ? 'subscription' : 'topup');
    initialTabSetRef.current = true;
  }, [shouldShowSubscription, subscriptionLoading]);

  useEffect(() => {
    if (!shouldShowSubscription && activeTab !== 'topup') {
      setActiveTab('topup');
    }
  }, [shouldShowSubscription, activeTab]);

  const topupContent = (
    <Space vertical style={{ width: '100%' }}>
      <Card
        className='!rounded-xl w-full'
        cover={
          <div
            className='relative h-30'
            style={{
              '--palette-primary-darkerChannel': '37 99 235',
              backgroundImage: `linear-gradient(0deg, rgba(var(--palette-primary-darkerChannel) / 80%), rgba(var(--palette-primary-darkerChannel) / 80%)), url('/cover-4.webp')`,
              backgroundSize: 'cover',
              backgroundPosition: 'center',
              backgroundRepeat: 'no-repeat',
            }}
          >
            <div className='relative z-10 h-full flex flex-col justify-between p-4'>
              <div className='flex justify-between items-center'>
                <Text strong style={{ color: 'white', fontSize: '16px' }}>
                  {t('账户统计')}
                </Text>
              </div>
              <div className='grid grid-cols-3 gap-6 mt-4'>
                <div className='text-center'>
                  <div
                    className='text-base sm:text-2xl font-bold mb-2'
                    style={{ color: 'white' }}
                  >
                    {renderQuota(userState?.user?.quota)}
                  </div>
                  <div className='flex items-center justify-center text-sm'>
                    <Wallet
                      size={14}
                      className='mr-1'
                      style={{ color: 'rgba(255,255,255,0.8)' }}
                    />
                    <Text
                      style={{
                        color: 'rgba(255,255,255,0.8)',
                        fontSize: '12px',
                      }}
                    >
                      {t('当前余额')}
                    </Text>
                  </div>
                </div>
                <div className='text-center'>
                  <div
                    className='text-base sm:text-2xl font-bold mb-2'
                    style={{ color: 'white' }}
                  >
                    {renderQuota(userState?.user?.used_quota)}
                  </div>
                  <div className='flex items-center justify-center text-sm'>
                    <TrendingUp
                      size={14}
                      className='mr-1'
                      style={{ color: 'rgba(255,255,255,0.8)' }}
                    />
                    <Text
                      style={{
                        color: 'rgba(255,255,255,0.8)',
                        fontSize: '12px',
                      }}
                    >
                      {t('历史消耗')}
                    </Text>
                  </div>
                </div>
                <div className='text-center'>
                  <div
                    className='text-base sm:text-2xl font-bold mb-2'
                    style={{ color: 'white' }}
                  >
                    {userState?.user?.request_count || 0}
                  </div>
                  <div className='flex items-center justify-center text-sm'>
                    <BarChart2
                      size={14}
                      className='mr-1'
                      style={{ color: 'rgba(255,255,255,0.8)' }}
                    />
                    <Text
                      style={{
                        color: 'rgba(255,255,255,0.8)',
                        fontSize: '12px',
                      }}
                    >
                      {t('请求次数')}
                    </Text>
                  </div>
                </div>
              </div>
            </div>
          </div>
        }
      >
        {statusLoading ? (
          <div className='py-8 flex justify-center'>
            <Spin size='large' />
          </div>
        ) : shouldShowAnyTopUpMethod ? (
          <Form
            getFormApi={(api) => (onlineFormApiRef.current = api)}
            initValues={{
              topUpCount,
              nowPaymentsCryptoAmount,
            }}
          >
            <div className='space-y-6'>
              {enableNOWPaymentsTopUp && (
                <Form.Slot label={t('NOWPayments')}>
                  <Space vertical style={{ width: '100%' }}>
                    <Text type='tertiary'>
                      请输入人民币金额，到账按 1 RMB = 1 USD 计算
                    </Text>
                    <div className='flex flex-wrap gap-2'>
                      {nowPaymentsModes?.fiat && (
                        <Button
                          theme={
                            nowPaymentsPricingMode === 'fiat'
                              ? 'solid'
                              : 'outline'
                          }
                          type='primary'
                          onClick={() => setNowPaymentsPricingMode('fiat')}
                        >
                          {t('法币定价模式')}
                        </Button>
                      )}
                      {nowPaymentsModes?.crypto && (
                        <Button
                          theme={
                            nowPaymentsPricingMode === 'crypto'
                              ? 'solid'
                              : 'outline'
                          }
                          type='primary'
                          onClick={() => setNowPaymentsPricingMode('crypto')}
                        >
                          {t('链上币定价模式')}
                        </Button>
                      )}
                    </div>
                    <div className='flex flex-wrap gap-2'>
                      {nowPaymentsNetworks.map((network) => (
                        <Button
                          key={network.code}
                          theme={
                            nowPaymentsPayCurrency === network.code
                              ? 'solid'
                              : 'outline'
                          }
                          type='tertiary'
                          onClick={() => setNowPaymentsPayCurrency(network.code)}
                        >
                          {network.name}
                        </Button>
                      ))}
                    </div>
                    {isNOWPaymentsCryptoMode && (
                      <Text type='tertiary'>
                        {t('到账按 1 USDT = 1 USD 等值计算')}
                      </Text>
                    )}
                  </Space>
                </Form.Slot>
              )}
              {enableBEpusdtTopUp && (
                <Form.Slot>
                  <Space vertical style={{ width: '100%' }}>
                    <Text type='tertiary'>
                      {t('请选择你希望用户支付的 USDT 网络')}
                    </Text>
                    <div className='flex flex-wrap gap-2'>
                      {bepusdtNetworks.map((network) => (
                        <Button
                          key={network.code}
                          theme={
                            bepusdtTradeType === network.code
                              ? 'solid'
                              : 'outline'
                          }
                          type='tertiary'
                          onClick={() => setBEpusdtTradeType(network.code)}
                        >
                          {network.name}
                        </Button>
                      ))}
                    </div>
                  </Space>
                </Form.Slot>
              )}

              {(shouldShowFiatAmountSection || isNOWPaymentsCryptoMode) && (
                <Row gutter={12}>
                  <Col xs={24} sm={24} md={24} lg={10} xl={10}>
                    {isNOWPaymentsCryptoMode ? (
                      <Form.InputNumber
                        field='nowPaymentsCryptoAmount'
                        label={t('支付数量')}
                        placeholder={t('请输入 USDT 数量')}
                        value={nowPaymentsCryptoAmount}
                        min={1}
                        max={999999999}
                        step={1}
                        precision={0}
                        onChange={(value) => {
                          if (value && value >= 1) {
                            setNowPaymentsCryptoAmount(value);
                            setSelectedNowPaymentsCryptoAmount(value);
                          }
                        }}
                        formatter={(value) => (value ? `${value}` : '')}
                        parser={(value) =>
                          value ? parseInt(value.replace(/[^\d]/g, '')) : 0
                        }
                        extraText={
                          <Text type='secondary'>
                            {t('到账按 1 USDT = 1 USD 等值计算')}
                          </Text>
                        }
                        style={{ width: '100%' }}
                      />
                    ) : (
                      <Form.InputNumber
                        field='topUpCount'
                        label={
                          isBEpusdtOnlyMode
                            ? t('支付人民币金额')
                            : t('充值数量')
                        }
                        disabled={!shouldShowFiatAmountSection}
                        placeholder={
                          isBEpusdtOnlyMode
                            ? `${t('支付人民币金额，最低 ')}${minTopUp} RMB`
                            : t('充值数量，最低 ') + renderQuotaWithAmount(minTopUp)
                        }
                        value={topUpCount}
                        min={minTopUp}
                        max={999999999}
                        step={1}
                        precision={0}
                        onChange={async (value) => {
                          if (value && value >= 1) {
                            setTopUpCount(value);
                            setSelectedPreset(null);
                            await getAmount(value);
                          }
                        }}
                        onBlur={(e) => {
                          const value = parseInt(e.target.value);
                          if (!value || value < minTopUp) {
                            setTopUpCount(minTopUp);
                            getAmount(minTopUp);
                          }
                        }}
                        formatter={(value) => (value ? `${value}` : '')}
                        parser={(value) =>
                          value ? parseInt(value.replace(/[^\d]/g, '')) : 0
                        }
                        extraText={
                          <Skeleton
                            loading={showAmountSkeleton}
                            active
                            placeholder={
                              <Skeleton.Title
                                style={{
                                  width: 120,
                                  height: 20,
                                  borderRadius: 6,
                                }}
                              />
                            }
                          >
                            <Text type='secondary' className='text-red-600'>
                              {isBEpusdtOnlyMode
                                ? `${t('到账额度：')}${Number(topUpCount || 0).toFixed(2)} USD · ${t('实付金额：')}`
                                : t('实付金额：')}
                              <span style={{ color: 'red' }}>
                                {renderAmount()}
                              </span>
                            </Text>
                          </Skeleton>
                        }
                        style={{ width: '100%' }}
                      />
                    )}
                  </Col>
                  <Col xs={24} sm={24} md={24} lg={14} xl={14}>
                    <Form.Slot label={t('选择支付方式')}>
                      {availablePayMethods.length > 0 ? (
                        <Space wrap>
                          {availablePayMethods.map((payMethod) => {
                            const minTopupVal = Number(payMethod.min_topup) || 0;
                            const isStripe = payMethod.type === 'stripe';
                            const isBTCPay = payMethod.type === 'btcpay';
                            const isBEpusdt = payMethod.type === 'bepusdt';
                            const isNOWPayments =
                              payMethod.type === 'nowpayments';
                            const insufficientMin =
                              !isNOWPaymentsCryptoMode &&
                              minTopupVal > Number(topUpCount || 0);
                            const disabled =
                              (!enableOnlineTopUp &&
                                !isStripe &&
                                !isBTCPay &&
                                !isBEpusdt &&
                                !isNOWPayments) ||
                              (!enableStripeTopUp && isStripe) ||
                              (!enableBTCPayTopUp && isBTCPay) ||
                              (!enableBEpusdtTopUp && isBEpusdt) ||
                              (!enableNOWPaymentsTopUp && isNOWPayments) ||
                              (isBEpusdt && !bepusdtTradeType) ||
                              (isNOWPayments && !nowPaymentsPayCurrency) ||
                              insufficientMin;

                            const buttonEl = (
                              <Button
                                key={payMethod.type}
                                theme='outline'
                                type='tertiary'
                                onClick={() => preTopUp(payMethod.type)}
                                disabled={disabled}
                                loading={
                                  paymentLoading && payWay === payMethod.type
                                }
                                icon={
                                  payMethod.type === 'alipay' ? (
                                    <SiAlipay size={18} color='#1677FF' />
                                  ) : payMethod.type === 'wxpay' ? (
                                    <SiWechat size={18} color='#07C160' />
                                  ) : payMethod.type === 'stripe' ? (
                                    <SiStripe size={18} color='#635BFF' />
                                  ) : (
                                    <CreditCard
                                      size={18}
                                      color={
                                        payMethod.color ||
                                        'var(--semi-color-text-2)'
                                      }
                                    />
                                  )
                                }
                                className='!rounded-lg !px-4 !py-2'
                              >
                                {payMethod.name}
                              </Button>
                            );

                            return insufficientMin ? (
                              <Tooltip
                                content={
                                  t('此支付方式最低充值金额为') +
                                  ' ' +
                                  minTopupVal
                                }
                                key={payMethod.type}
                              >
                                {buttonEl}
                              </Tooltip>
                            ) : (
                              <React.Fragment key={payMethod.type}>
                                {buttonEl}
                              </React.Fragment>
                            );
                          })}
                        </Space>
                      ) : (
                        <div className='text-gray-500 text-sm p-3 bg-gray-50 rounded-lg border border-dashed border-gray-300'>
                          {t('暂无可用的支付方式，请联系管理员配置')}
                        </div>
                      )}
                    </Form.Slot>
                  </Col>
                </Row>
              )}

              {shouldShowFiatAmountSection && (
                <Form.Slot
                  label={
                    <div className='flex items-center gap-2'>
                      <span>
                        {isBEpusdtOnlyMode
                          ? t('选择支付金额')
                          : t('选择充值额度')}
                      </span>
                      {(() => {
                        if (isBEpusdtOnlyMode) {
                          return (
                            <span
                              style={{
                                color: 'var(--semi-color-text-2)',
                                fontSize: '12px',
                                fontWeight: 'normal',
                              }}
                            >
                              (1 RMB = 1 USD)
                            </span>
                          );
                        }
                        const { symbol, rate, type } = getCurrencyConfig();
                        if (type === 'USD') return null;
                        return (
                          <span
                            style={{
                              color: 'var(--semi-color-text-2)',
                              fontSize: '12px',
                              fontWeight: 'normal',
                            }}
                          >
                            (1 $ = {rate.toFixed(2)} {symbol})
                          </span>
                        );
                      })()}
                    </div>
                  }
                >
                  <div className='grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-2'>
                    {presetAmounts.map((preset, index) => {
                      if (isBEpusdtOnlyMode) {
                        return (
                          <Card
                            key={index}
                            style={{
                              cursor: 'pointer',
                              border:
                                selectedPreset === preset.value
                                  ? '2px solid var(--semi-color-primary)'
                                  : '1px solid var(--semi-color-border)',
                              height: '100%',
                              width: '100%',
                            }}
                            bodyStyle={{ padding: '12px' }}
                            onClick={() => {
                              selectPresetAmount(preset);
                              onlineFormApiRef.current?.setValue(
                                'topUpCount',
                                preset.value,
                              );
                            }}
                          >
                            <div style={{ textAlign: 'center' }}>
                              <Typography.Title
                                heading={6}
                                style={{ margin: '0 0 8px 0' }}
                              >
                                <Coins size={18} />
                                {formatLargeNumber(preset.value)} RMB
                              </Typography.Title>
                              <div
                                style={{
                                  color: 'var(--semi-color-text-2)',
                                  fontSize: '12px',
                                  margin: '4px 0',
                                }}
                              >
                                {t('到账')} {preset.value} USD
                              </div>
                            </div>
                          </Card>
                        );
                      }
                      const discount =
                        preset.discount ||
                        topupInfo?.discount?.[preset.value] ||
                        1.0;
                      const originalPrice = preset.value * priceRatio;
                      const discountedPrice = originalPrice * discount;
                      const hasDiscount = discount < 1.0;
                      const actualPay = discountedPrice;
                      const save = originalPrice - discountedPrice;

                      const { symbol, rate, type } = getCurrencyConfig();
                      const statusStr = localStorage.getItem('status');
                      let usdRate = 7;
                      try {
                        if (statusStr) {
                          const status = JSON.parse(statusStr);
                          usdRate = status?.usd_exchange_rate || 7;
                        }
                      } catch (e) {
                        // ignore
                      }

                      let displayValue = preset.value;
                      let displayActualPay = actualPay;
                      let displaySave = save;

                      if (type === 'USD') {
                        displayActualPay = actualPay / usdRate;
                        displaySave = save / usdRate;
                      } else if (type === 'CNY') {
                        displayValue = preset.value * usdRate;
                      } else if (type === 'CUSTOM') {
                        displayValue = preset.value * rate;
                        displayActualPay = (actualPay / usdRate) * rate;
                        displaySave = (save / usdRate) * rate;
                      }

                      return (
                        <Card
                          key={index}
                          style={{
                            cursor: 'pointer',
                            border:
                              selectedPreset === preset.value
                                ? '2px solid var(--semi-color-primary)'
                                : '1px solid var(--semi-color-border)',
                            height: '100%',
                            width: '100%',
                          }}
                          bodyStyle={{ padding: '12px' }}
                          onClick={() => {
                            selectPresetAmount(preset);
                            onlineFormApiRef.current?.setValue(
                              'topUpCount',
                              preset.value,
                            );
                          }}
                        >
                          <div style={{ textAlign: 'center' }}>
                            <Typography.Title
                              heading={6}
                              style={{ margin: '0 0 8px 0' }}
                            >
                              <Coins size={18} />
                              {formatLargeNumber(displayValue)} {symbol}
                              {hasDiscount && (
                                <Tag style={{ marginLeft: 4 }} color='green'>
                                  {t('折').includes('off')
                                    ? ((1 - parseFloat(discount)) * 100).toFixed(1)
                                    : (discount * 10).toFixed(1)}
                                  {t('折')}
                                </Tag>
                              )}
                            </Typography.Title>
                            <div
                              style={{
                                color: 'var(--semi-color-text-2)',
                                fontSize: '12px',
                                margin: '4px 0',
                              }}
                            >
                              {t('实付')} {symbol}
                              {displayActualPay.toFixed(2)}，
                              {hasDiscount
                                ? `${t('节省')} ${symbol}${displaySave.toFixed(2)}`
                                : `${t('节省')} ${symbol}0.00`}
                            </div>
                          </div>
                        </Card>
                      );
                    })}
                  </div>
                </Form.Slot>
              )}

              {isNOWPaymentsCryptoMode && (
                <Form.Slot label={t('快捷选择 USDT 金额')}>
                  <div className='grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 gap-2'>
                    {nowPaymentsCryptoAmountOptions.map((option) => (
                      <Card
                        key={option}
                        style={{
                          cursor: 'pointer',
                          border:
                            selectedNowPaymentsCryptoAmount === option
                              ? '2px solid var(--semi-color-primary)'
                              : '1px solid var(--semi-color-border)',
                        }}
                        bodyStyle={{ padding: '12px', textAlign: 'center' }}
                        onClick={() => {
                          setNowPaymentsCryptoAmount(option);
                          setSelectedNowPaymentsCryptoAmount(option);
                          onlineFormApiRef.current?.setValue(
                            'nowPaymentsCryptoAmount',
                            option,
                          );
                        }}
                      >
                        <Typography.Title
                          heading={6}
                          style={{ margin: '0 0 6px 0' }}
                        >
                          {option} USDT
                        </Typography.Title>
                        <Text type='tertiary'>
                          {t('等值')} {option} USD
                        </Text>
                      </Card>
                    ))}
                  </div>
                </Form.Slot>
              )}

              {enableCreemTopUp && creemProducts.length > 0 && (
                <Form.Slot label={t('Creem 充值')}>
                  <div className='grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3'>
                    {creemProducts.map((product, index) => (
                      <Card
                        key={index}
                        onClick={() => creemPreTopUp(product)}
                        className='cursor-pointer !rounded-2xl transition-all hover:shadow-md border-gray-200 hover:border-gray-300'
                        bodyStyle={{ textAlign: 'center', padding: '16px' }}
                      >
                        <div className='font-medium text-lg mb-2'>
                          {product.name}
                        </div>
                        <div className='text-sm text-gray-600 mb-2'>
                          {t('充值额度')}: {product.quota}
                        </div>
                        <div className='text-lg font-semibold text-blue-600'>
                          {product.currency === 'EUR' ? '€' : '$'}
                          {product.price}
                        </div>
                      </Card>
                    ))}
                  </div>
                </Form.Slot>
              )}
            </div>
          </Form>
        ) : (
          <Banner
            type='info'
            description={t(
              '管理员未开启在线充值功能，请联系管理员开启或使用兑换码充值。',
            )}
            className='!rounded-xl'
            closeIcon={null}
          />
        )}
      </Card>

      <Card
        className='!rounded-xl w-full'
        title={
          <Text type='tertiary' strong>
            {t('兑换码充值')}
          </Text>
        }
      >
        <Form
          getFormApi={(api) => (redeemFormApiRef.current = api)}
          initValues={{ redemptionCode }}
        >
          <Form.Input
            field='redemptionCode'
            noLabel={true}
            placeholder={t('请输入兑换码')}
            value={redemptionCode}
            onChange={(value) => setRedemptionCode(value)}
            prefix={<IconGift />}
            suffix={
              <div className='flex items-center gap-2'>
                <Button
                  type='primary'
                  theme='solid'
                  onClick={topUp}
                  loading={isSubmitting}
                >
                  {t('兑换额度')}
                </Button>
              </div>
            }
            showClear
            style={{ width: '100%' }}
            extraText={
              topUpLink && (
                <Text type='tertiary'>
                  {t('在找兑换码？')}
                  <Text
                    type='secondary'
                    underline
                    className='cursor-pointer'
                    onClick={openTopUpLink}
                  >
                    {t('购买兑换码')}
                  </Text>
                </Text>
              )
            }
          />
        </Form>
      </Card>
    </Space>
  );

  return (
    <Card className='!rounded-2xl shadow-sm border-0'>
      <div className='flex items-center justify-between mb-4'>
        <div className='flex items-center'>
          <Avatar size='small' color='blue' className='mr-3 shadow-md'>
            <CreditCard size={16} />
          </Avatar>
          <div>
            <Typography.Text className='text-lg font-medium'>
              {t('账户充值')}
            </Typography.Text>
            <div className='text-xs'>{t('多种充值方式，安全便捷')}</div>
          </div>
        </div>
        <Button
          icon={<Receipt size={16} />}
          theme='solid'
          onClick={onOpenHistory}
        >
          {t('账单')}
        </Button>
      </div>

      {shouldShowSubscription ? (
        <Tabs type='card' activeKey={activeTab} onChange={setActiveTab}>
          <TabPane
            tab={
              <div className='flex items-center gap-2'>
                <Sparkles size={16} />
                {t('订阅套餐')}
              </div>
            }
            itemKey='subscription'
          >
            <div className='py-2'>
              <SubscriptionPlansCard
                t={t}
                loading={subscriptionLoading}
                plans={subscriptionPlans}
                payMethods={payMethods}
                enableOnlineTopUp={enableOnlineTopUp}
                enableStripeTopUp={enableStripeTopUp}
                enableCreemTopUp={enableCreemTopUp}
                billingPreference={billingPreference}
                onChangeBillingPreference={onChangeBillingPreference}
                activeSubscriptions={activeSubscriptions}
                allSubscriptions={allSubscriptions}
                reloadSubscriptionSelf={reloadSubscriptionSelf}
                withCard={false}
              />
            </div>
          </TabPane>
          <TabPane
            tab={
              <div className='flex items-center gap-2'>
                <Wallet size={16} />
                {t('额度充值')}
              </div>
            }
            itemKey='topup'
          >
            <div className='py-2'>{topupContent}</div>
          </TabPane>
        </Tabs>
      ) : (
        topupContent
      )}
    </Card>
  );
};

export default RechargeCard;
