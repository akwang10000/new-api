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

import React, { useContext, useEffect, useRef, useState } from 'react';
import {
  API,
  copy,
  getQuotaPerUnit,
  renderQuota,
  renderQuotaWithAmount,
  showError,
  showInfo,
  showSuccess,
} from '../../helpers';
import { Modal, Toast } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { UserContext } from '../../context/User';
import { StatusContext } from '../../context/Status';

import RechargeCard from './RechargeCard';
import InvitationCard from './InvitationCard';
import TransferModal from './modals/TransferModal';
import PaymentConfirmModal from './modals/PaymentConfirmModal';
import TopupHistoryModal from './modals/TopupHistoryModal';

const defaultNOWPaymentsModes = {
  fiat: false,
  crypto: false,
};

const defaultNOWPaymentsCryptoAmountOptions = [5, 10, 20, 50, 100];
const defaultBEpusdtNetworks = [];

const TopUp = () => {
  const { t } = useTranslation();
  const [userState, userDispatch] = useContext(UserContext);
  const [statusState] = useContext(StatusContext);

  const [redemptionCode, setRedemptionCode] = useState('');
  const [amount, setAmount] = useState(0.0);
  const [minTopUp, setMinTopUp] = useState(statusState?.status?.min_topup || 1);
  const [topUpCount, setTopUpCount] = useState(
    statusState?.status?.min_topup || 1,
  );
  const [topUpLink, setTopUpLink] = useState(
    statusState?.status?.top_up_link || '',
  );
  const [enableOnlineTopUp, setEnableOnlineTopUp] = useState(
    statusState?.status?.enable_online_topup || false,
  );
  const [priceRatio, setPriceRatio] = useState(statusState?.status?.price || 1);

  const [enableStripeTopUp, setEnableStripeTopUp] = useState(
    statusState?.status?.enable_stripe_topup || false,
  );
  const [enableBTCPayTopUp, setEnableBTCPayTopUp] = useState(false);
  const [enableBEpusdtTopUp, setEnableBEpusdtTopUp] = useState(false);
  const [bepusdtNetworks, setBEpusdtNetworks] = useState(
    defaultBEpusdtNetworks,
  );
  const [bepusdtTradeType, setBEpusdtTradeType] = useState('');
  const [enableNOWPaymentsTopUp, setEnableNOWPaymentsTopUp] = useState(false);
  const [nowPaymentsModes, setNowPaymentsModes] = useState(
    defaultNOWPaymentsModes,
  );
  const [nowPaymentsPricingMode, setNowPaymentsPricingMode] =
    useState('fiat');
  const [nowPaymentsNetworks, setNowPaymentsNetworks] = useState([]);
  const [nowPaymentsPayCurrency, setNowPaymentsPayCurrency] = useState('');
  const [nowPaymentsCryptoAmountOptions, setNowPaymentsCryptoAmountOptions] =
    useState(defaultNOWPaymentsCryptoAmountOptions);
  const [nowPaymentsCryptoAmount, setNowPaymentsCryptoAmount] = useState(
    defaultNOWPaymentsCryptoAmountOptions[0],
  );
  const [selectedNowPaymentsCryptoAmount, setSelectedNowPaymentsCryptoAmount] =
    useState(defaultNOWPaymentsCryptoAmountOptions[0]);
  const [nowPaymentsQuote, setNowPaymentsQuote] = useState(null);
  const [statusLoading, setStatusLoading] = useState(true);

  const [creemProducts, setCreemProducts] = useState([]);
  const [enableCreemTopUp, setEnableCreemTopUp] = useState(false);
  const [creemOpen, setCreemOpen] = useState(false);
  const [selectedCreemProduct, setSelectedCreemProduct] = useState(null);

  const [isSubmitting, setIsSubmitting] = useState(false);
  const [open, setOpen] = useState(false);
  const [payWay, setPayWay] = useState('');
  const [amountLoading, setAmountLoading] = useState(false);
  const [paymentLoading, setPaymentLoading] = useState(false);
  const [confirmLoading, setConfirmLoading] = useState(false);
  const [payMethods, setPayMethods] = useState([]);

  const affFetchedRef = useRef(false);

  const [affLink, setAffLink] = useState('');
  const [openTransfer, setOpenTransfer] = useState(false);
  const [transferAmount, setTransferAmount] = useState(0);
  const [openHistory, setOpenHistory] = useState(false);

  const [subscriptionPlans, setSubscriptionPlans] = useState([]);
  const [subscriptionLoading, setSubscriptionLoading] = useState(true);
  const [billingPreference, setBillingPreference] =
    useState('subscription_first');
  const [activeSubscriptions, setActiveSubscriptions] = useState([]);
  const [allSubscriptions, setAllSubscriptions] = useState([]);

  const [presetAmounts, setPresetAmounts] = useState([]);
  const [selectedPreset, setSelectedPreset] = useState(null);

  const [topupInfo, setTopupInfo] = useState({
    amount_options: [],
    discount: {},
  });

  const isBEpusdtOnlyMode = () => {
    return (
      enableBEpusdtTopUp &&
      !enableOnlineTopUp &&
      !enableStripeTopUp &&
      !enableBTCPayTopUp &&
      !enableNOWPaymentsTopUp
    );
  };

  const topUp = async () => {
    if (redemptionCode === '') {
      showInfo(t('请输入兑换码！'));
      return;
    }
    setIsSubmitting(true);
    try {
      const res = await API.post('/api/user/topup', {
        key: redemptionCode,
      });
      const { success, message, data } = res.data;
      if (success) {
        showSuccess(t('兑换成功！'));
        Modal.success({
          title: t('兑换成功！'),
          content: t('成功兑换额度：') + renderQuota(data),
          centered: true,
        });
        if (userState.user) {
          const updatedUser = {
            ...userState.user,
            quota: userState.user.quota + data,
          };
          userDispatch({ type: 'login', payload: updatedUser });
        }
        setRedemptionCode('');
      } else {
        showError(message);
      }
    } catch (err) {
      showError(t('请求失败'));
    } finally {
      setIsSubmitting(false);
    }
  };

  const openTopUpLink = () => {
    if (!topUpLink) {
      showError(t('超级管理员未设置充值链接！'));
      return;
    }
    window.open(topUpLink, '_blank');
  };

  const getCurrentNOWPaymentsAmount = () => {
    return nowPaymentsPricingMode === 'crypto'
      ? nowPaymentsCryptoAmount
      : topUpCount;
  };

  const getNOWPaymentsQuote = async (value) => {
    const amountValue = value ?? getCurrentNOWPaymentsAmount();
    const res = await API.post('/api/user/nowpayments/quote', {
      amount: parseInt(amountValue),
      pricing_mode: nowPaymentsPricingMode,
      pay_currency: nowPaymentsPayCurrency,
      payment_method: 'nowpayments',
    });
    const { message, data } = res.data;
    if (message === 'success') {
      setNowPaymentsQuote(data);
      return data;
    }
    setNowPaymentsQuote(null);
    throw new Error(typeof data === 'string' ? data : message);
  };

  const ensureNOWPaymentsReady = async () => {
    if (!enableNOWPaymentsTopUp) {
      showError(t('管理员未开启 NOWPayments 充值！'));
      return false;
    }
    if (!nowPaymentsPayCurrency) {
      showError(t('请选择 USDT 网络'));
      return false;
    }
    const quote = await getNOWPaymentsQuote();
    if (!quote?.meets_minimum) {
      showError(
        `${t('当前金额低于该网络最小支付金额')} (${quote?.minimum_amount || 0})`,
      );
      return false;
    }
    return true;
  };

  const preTopUp = async (payment) => {
    if (payment === 'stripe') {
      if (!enableStripeTopUp) {
        showError(t('管理员未开启 Stripe 充值！'));
        return;
      }
    } else if (payment === 'btcpay') {
      if (!enableBTCPayTopUp) {
        showError(t('管理员未开启 BTCPay 充值！'));
        return;
      }
    } else if (payment === 'bepusdt') {
      if (!enableBEpusdtTopUp) {
        showError(t('管理员未开启 BEpusdt 充值！'));
        return;
      }
      if (!bepusdtTradeType) {
        showError(t('请选择 USDT 网络'));
        return;
      }
    } else if (payment === 'nowpayments') {
      if (!enableNOWPaymentsTopUp) {
        showError(t('管理员未开启 NOWPayments 充值！'));
        return;
      }
    } else if (!enableOnlineTopUp) {
      showError(t('管理员未开启在线充值！'));
      return;
    }

    setPayWay(payment);
    setPaymentLoading(true);
    try {
      if (payment === 'stripe') {
        await getStripeAmount();
      } else if (payment === 'bepusdt') {
        setAmount(parseFloat(topUpCount || 0));
      } else if (payment === 'nowpayments') {
        const ready = await ensureNOWPaymentsReady();
        if (!ready) {
          return;
        }
      } else {
        await getAmount();
      }

      if (payment !== 'nowpayments' && topUpCount < minTopUp) {
        showError(t('充值数量不能小于') + minTopUp);
        return;
      }
      setOpen(true);
    } catch (error) {
      showError(error?.message || t('获取金额失败'));
    } finally {
      setPaymentLoading(false);
    }
  };

  const onlineTopUp = async () => {
    if (payWay === 'stripe') {
      if (amount === 0) {
        await getStripeAmount();
      }
    } else if (payWay === 'bepusdt') {
      setAmount(parseFloat(topUpCount || 0));
    } else if (payWay === 'nowpayments') {
      const ready = await ensureNOWPaymentsReady();
      if (!ready) {
        return;
      }
    } else if (amount === 0) {
      await getAmount();
    }

    if (payWay !== 'nowpayments' && topUpCount < minTopUp) {
      showError(t('充值数量不能小于') + minTopUp);
      return;
    }

    setConfirmLoading(true);
    try {
      let res;
      if (payWay === 'stripe') {
        res = await API.post('/api/user/stripe/pay', {
          amount: parseInt(topUpCount),
          payment_method: 'stripe',
        });
      } else if (payWay === 'btcpay') {
        res = await API.post('/api/user/btcpay/pay', {
          amount: parseInt(topUpCount),
          payment_method: 'btcpay',
        });
      } else if (payWay === 'bepusdt') {
        res = await API.post('/api/user/bepusdt/pay', {
          amount: parseInt(topUpCount),
          trade_type: bepusdtTradeType,
          payment_method: 'bepusdt',
        });
      } else if (payWay === 'nowpayments') {
        res = await API.post('/api/user/nowpayments/pay', {
          amount: parseInt(getCurrentNOWPaymentsAmount()),
          pricing_mode: nowPaymentsPricingMode,
          pay_currency: nowPaymentsPayCurrency,
          payment_method: 'nowpayments',
        });
      } else {
        res = await API.post('/api/user/pay', {
          amount: parseInt(topUpCount),
          payment_method: payWay,
        });
      }

      if (res !== undefined) {
        const { message, data } = res.data;
        if (message === 'success') {
          if (
            payWay === 'stripe' ||
            payWay === 'btcpay' ||
            payWay === 'bepusdt' ||
            payWay === 'nowpayments'
          ) {
            window.open(data.pay_link, '_blank');
          } else {
            const params = data;
            const url = res.data.url;
            const form = document.createElement('form');
            form.action = url;
            form.method = 'POST';
            const isSafari =
              navigator.userAgent.indexOf('Safari') > -1 &&
              navigator.userAgent.indexOf('Chrome') < 1;
            if (!isSafari) {
              form.target = '_blank';
            }
            for (const key in params) {
              const input = document.createElement('input');
              input.type = 'hidden';
              input.name = key;
              input.value = params[key];
              form.appendChild(input);
            }
            document.body.appendChild(form);
            form.submit();
            document.body.removeChild(form);
          }
        } else {
          const errorMsg =
            typeof data === 'string' ? data : message || t('支付失败');
          showError(errorMsg);
        }
      } else {
        showError(res);
      }
    } catch (err) {
      showError(t('支付请求失败'));
    } finally {
      setOpen(false);
      setConfirmLoading(false);
    }
  };

  const creemPreTopUp = async (product) => {
    if (!enableCreemTopUp) {
      showError(t('管理员未开启 Creem 充值！'));
      return;
    }
    setSelectedCreemProduct(product);
    setCreemOpen(true);
  };

  const onlineCreemTopUp = async () => {
    if (!selectedCreemProduct) {
      showError(t('请选择产品'));
      return;
    }
    if (!selectedCreemProduct.productId) {
      showError(t('产品配置错误，请联系管理员'));
      return;
    }
    setConfirmLoading(true);
    try {
      const res = await API.post('/api/user/creem/pay', {
        product_id: selectedCreemProduct.productId,
        payment_method: 'creem',
      });
      if (res !== undefined) {
        const { message, data } = res.data;
        if (message === 'success') {
          window.open(data.checkout_url, '_blank');
        } else {
          const errorMsg =
            typeof data === 'string' ? data : message || t('支付失败');
          showError(errorMsg);
        }
      } else {
        showError(res);
      }
    } catch (err) {
      showError(t('支付请求失败'));
    } finally {
      setCreemOpen(false);
      setConfirmLoading(false);
    }
  };

  const getUserQuota = async () => {
    const res = await API.get('/api/user/self');
    const { success, message, data } = res.data;
    if (success) {
      userDispatch({ type: 'login', payload: data });
    } else {
      showError(message);
    }
  };

  const getSubscriptionPlans = async () => {
    setSubscriptionLoading(true);
    try {
      const res = await API.get('/api/subscription/plans');
      if (res.data?.success) {
        setSubscriptionPlans(res.data.data || []);
      }
    } catch (e) {
      setSubscriptionPlans([]);
    } finally {
      setSubscriptionLoading(false);
    }
  };

  const getSubscriptionSelf = async () => {
    try {
      const res = await API.get('/api/subscription/self');
      if (res.data?.success) {
        setBillingPreference(
          res.data.data?.billing_preference || 'subscription_first',
        );
        setActiveSubscriptions(res.data.data?.subscriptions || []);
        setAllSubscriptions(res.data.data?.all_subscriptions || []);
      }
    } catch (e) {
      // ignore
    }
  };

  const updateBillingPreference = async (pref) => {
    const previousPref = billingPreference;
    setBillingPreference(pref);
    try {
      const res = await API.put('/api/subscription/self/preference', {
        billing_preference: pref,
      });
      if (res.data?.success) {
        showSuccess(t('更新成功'));
        const normalizedPref =
          res.data?.data?.billing_preference || pref || previousPref;
        setBillingPreference(normalizedPref);
      } else {
        showError(res.data?.message || t('更新失败'));
        setBillingPreference(previousPref);
      }
    } catch (e) {
      showError(t('请求失败'));
      setBillingPreference(previousPref);
    }
  };

  const normalizePayMethods = (rawPayMethods, stripeMinTopUp) => {
    let normalized = rawPayMethods || [];
    if (typeof normalized === 'string') {
      normalized = JSON.parse(normalized);
    }
    if (!Array.isArray(normalized)) {
      return [];
    }
    return normalized
      .filter((method) => method?.name && method?.type)
      .map((method) => {
        const minTopup = Number(method.min_topup);
        const nextMethod = {
          ...method,
          min_topup: Number.isFinite(minTopup) ? minTopup : 0,
        };
        if (
          nextMethod.type === 'stripe' &&
          (!nextMethod.min_topup || nextMethod.min_topup <= 0)
        ) {
          const stripeMin = Number(stripeMinTopUp);
          if (Number.isFinite(stripeMin)) {
            nextMethod.min_topup = stripeMin;
          }
        }
        if (!nextMethod.color) {
          if (nextMethod.type === 'alipay') {
            nextMethod.color = 'rgba(var(--semi-blue-5), 1)';
          } else if (nextMethod.type === 'wxpay') {
            nextMethod.color = 'rgba(var(--semi-green-5), 1)';
          } else if (nextMethod.type === 'stripe') {
            nextMethod.color = 'rgba(var(--semi-purple-5), 1)';
          } else if (nextMethod.type === 'btcpay') {
            nextMethod.color = 'rgba(var(--semi-orange-5), 1)';
          } else if (nextMethod.type === 'bepusdt') {
            nextMethod.color = 'rgba(var(--semi-cyan-5), 1)';
          } else if (nextMethod.type === 'nowpayments') {
            nextMethod.color = 'rgba(var(--semi-teal-5), 1)';
          } else {
            nextMethod.color = 'rgba(var(--semi-primary-5), 1)';
          }
        }
        return nextMethod;
      });
  };

  const getTopupInfo = async () => {
    try {
      const res = await API.get('/api/user/topup/info');
      const { data, success } = res.data;
      if (!success) {
        return;
      }

      setTopupInfo({
        amount_options: data.amount_options || [],
        discount: data.discount || {},
      });

      try {
        const normalizedPayMethods = normalizePayMethods(
          data.pay_methods,
          data.stripe_min_topup,
        );
        setPayMethods(normalizedPayMethods);
      } catch (e) {
        console.log('解析支付方式失败:', e);
        setPayMethods([]);
      }

      const nextEnableOnlineTopUp = data.enable_online_topup || false;
      const nextEnableStripeTopUp = data.enable_stripe_topup || false;
      const nextEnableCreemTopUp = data.enable_creem_topup || false;
      const nextEnableBTCPayTopUp = data.enable_btcpay_topup || false;
      const nextEnableBEpusdtTopUp = data.enable_bepusdt_topup || false;
      const nextEnableNOWPaymentsTopUp =
        data.enable_nowpayments_topup || false;
      const incomingBEpusdtNetworks = Array.isArray(data.bepusdt_usdt_networks)
        ? data.bepusdt_usdt_networks
        : [];
      const incomingNOWPaymentsModes =
        data.nowpayments_modes || defaultNOWPaymentsModes;
      const incomingNOWPaymentsNetworks = Array.isArray(
        data.nowpayments_usdt_networks,
      )
        ? data.nowpayments_usdt_networks
        : [];
      const incomingNOWPaymentsCryptoAmountOptions =
        Array.isArray(data.nowpayments_crypto_amount_options) &&
        data.nowpayments_crypto_amount_options.length > 0
          ? data.nowpayments_crypto_amount_options
          : defaultNOWPaymentsCryptoAmountOptions;

      const defaultPricingMode =
        incomingNOWPaymentsModes.crypto && !incomingNOWPaymentsModes.fiat
          ? 'crypto'
          : 'fiat';
      const defaultBEpusdtTradeType = incomingBEpusdtNetworks[0]?.code || '';
      const defaultPayCurrency = incomingNOWPaymentsNetworks[0]?.code || '';
      const defaultCryptoAmount = incomingNOWPaymentsCryptoAmountOptions[0] || 5;
      const nextMinTopUp =
        nextEnableOnlineTopUp ||
        nextEnableBTCPayTopUp ||
        nextEnableBEpusdtTopUp ||
        nextEnableNOWPaymentsTopUp
          ? data.min_topup
          : nextEnableStripeTopUp
            ? data.stripe_min_topup
            : 1;

      setEnableOnlineTopUp(nextEnableOnlineTopUp);
      setEnableStripeTopUp(nextEnableStripeTopUp);
      setEnableCreemTopUp(nextEnableCreemTopUp);
      setEnableBTCPayTopUp(nextEnableBTCPayTopUp);
      setEnableBEpusdtTopUp(nextEnableBEpusdtTopUp);
      setBEpusdtNetworks(incomingBEpusdtNetworks);
      setBEpusdtTradeType(defaultBEpusdtTradeType);
      setEnableNOWPaymentsTopUp(nextEnableNOWPaymentsTopUp);
      setNowPaymentsModes(incomingNOWPaymentsModes);
      setNowPaymentsPricingMode(defaultPricingMode);
      setNowPaymentsNetworks(incomingNOWPaymentsNetworks);
      setNowPaymentsPayCurrency(defaultPayCurrency);
      setNowPaymentsCryptoAmountOptions(incomingNOWPaymentsCryptoAmountOptions);
      setNowPaymentsCryptoAmount(defaultCryptoAmount);
      setSelectedNowPaymentsCryptoAmount(defaultCryptoAmount);
      setNowPaymentsQuote(null);
      setMinTopUp(nextMinTopUp);
      setTopUpCount(nextMinTopUp);

      try {
        const products = JSON.parse(data.creem_products || '[]');
        setCreemProducts(products);
      } catch (e) {
        setCreemProducts([]);
      }

      if (Array.isArray(data.amount_options) && data.amount_options.length > 0) {
        setPresetAmounts(
          data.amount_options.map((item) => ({
            value: item,
            discount: data.discount?.[item] || 1.0,
          })),
        );
      } else {
        setPresetAmounts(generatePresetAmounts(nextMinTopUp));
      }

      if (nextEnableBEpusdtTopUp && !nextEnableOnlineTopUp && !nextEnableStripeTopUp && !nextEnableBTCPayTopUp && !nextEnableNOWPaymentsTopUp) {
        setAmount(parseFloat(nextMinTopUp || 0));
      } else {
        getAmount(nextMinTopUp);
      }
    } catch (error) {
      console.error('获取充值配置异常:', error);
    }
  };

  const getAffLink = async () => {
    const res = await API.get('/api/user/aff');
    const { success, message, data } = res.data;
    if (success) {
      setAffLink(`${window.location.origin}/register?aff=${data}`);
    } else {
      showError(message);
    }
  };

  const transfer = async () => {
    if (transferAmount < getQuotaPerUnit()) {
      showError(t('划转金额最低为') + ' ' + renderQuota(getQuotaPerUnit()));
      return;
    }
    const res = await API.post('/api/user/aff_transfer', {
      quota: transferAmount,
    });
    const { success, message } = res.data;
    if (success) {
      showSuccess(message);
      setOpenTransfer(false);
      getUserQuota().then();
    } else {
      showError(message);
    }
  };

  const handleAffLinkClick = async () => {
    await copy(affLink);
    showSuccess(t('邀请链接已复制到剪切板'));
  };

  useEffect(() => {
    getUserQuota().then();
    setTransferAmount(getQuotaPerUnit());
  }, []);

  useEffect(() => {
    if (affFetchedRef.current) return;
    affFetchedRef.current = true;
    getAffLink().then();
  }, []);

  useEffect(() => {
    getTopupInfo().then();
    getSubscriptionPlans().then();
    getSubscriptionSelf().then();
  }, []);

  useEffect(() => {
    if (statusState?.status) {
      setTopUpLink(statusState.status.top_up_link || '');
      setPriceRatio(statusState.status.price || 1);
      setStatusLoading(false);
    }
  }, [statusState?.status]);

  useEffect(() => {
    setNowPaymentsQuote(null);
  }, [
    nowPaymentsPricingMode,
    nowPaymentsPayCurrency,
    nowPaymentsCryptoAmount,
    topUpCount,
  ]);

  const getNOWPaymentsNetworkName = () => {
    const selectedNetwork = nowPaymentsNetworks.find(
      (network) => network.code === nowPaymentsPayCurrency,
    );
    return selectedNetwork?.name || nowPaymentsPayCurrency?.toUpperCase() || '-';
  };

  const getNOWPaymentsDisplayCurrency = () => {
    if (!nowPaymentsQuote) {
      return 'USD';
    }
    const priceCurrency = String(nowPaymentsQuote.price_currency || '')
      .trim()
      .toLowerCase();
    if (priceCurrency === 'usd') {
      return 'USD';
    }
    if (priceCurrency === nowPaymentsPayCurrency) {
      return getNOWPaymentsNetworkName();
    }
    return priceCurrency.toUpperCase();
  };

  const renderAmount = () => {
    if (payWay === 'nowpayments' && nowPaymentsQuote) {
      return `${Number(nowPaymentsQuote.price_amount || 0).toFixed(2)} ${getNOWPaymentsDisplayCurrency()}`;
    }
    if (payWay === 'bepusdt' || isBEpusdtOnlyMode()) {
      return `${Number(topUpCount || 0).toFixed(2)} RMB`;
    }
    return amount + ' ' + t('元');
  };

  const getBEpusdtNetworkName = () => {
    const selectedNetwork = bepusdtNetworks.find(
      (network) => network.code === bepusdtTradeType,
    );
    return selectedNetwork?.name || bepusdtTradeType || '-';
  };

  const getPaymentConfirmSummary = () => {
    if (payWay === 'bepusdt') {
      return {
        countLabel: t('到账额度'),
        countValue: `${Number(topUpCount || 0).toFixed(2)} USD`,
        amountLabel: t('实付金额'),
        amountValue: renderAmount(),
        extraRows: [
          {
            label: t('支付网络'),
            value: getBEpusdtNetworkName(),
          },
          {
            label: t('到账规则'),
            value: '1 RMB = 1 USD',
          },
        ],
      };
    }
    if (payWay !== 'nowpayments' || !nowPaymentsQuote) {
      return null;
    }
    const creditAmount = Number(
      nowPaymentsQuote.credit_amount || nowPaymentsQuote.price_amount || 0,
    );
    const summary = {
      countLabel: t('到账额度'),
      countValue: renderQuotaWithAmount(creditAmount),
      amountLabel: t('实付金额'),
      amountValue: renderAmount(),
      extraRows: [
        {
          label: t('定价模式'),
          value:
            nowPaymentsPricingMode === 'crypto'
              ? t('链上币定价')
              : t('法币定价'),
        },
        {
          label: t('支付网络'),
          value: getNOWPaymentsNetworkName(),
        },
      ],
    };
    if (nowPaymentsPricingMode === 'crypto') {
      summary.extraRows.push({
        label: t('到账规则'),
        value: t('按 1 USDT = 1 USD 等值到账'),
      });
    }
    return summary;
  };

  const getAmount = async (value) => {
    const nextValue = value === undefined ? topUpCount : value;
    setAmountLoading(true);
    try {
      const res = await API.post('/api/user/amount', {
        amount: parseFloat(nextValue),
      });
      if (res !== undefined) {
        const { message, data } = res.data;
        if (message === 'success') {
          setAmount(parseFloat(data));
        } else {
          setAmount(0);
          Toast.error({ content: '错误：' + data, id: 'getAmount' });
        }
      } else {
        showError(res);
      }
    } catch (err) {
      console.log(err);
    }
    setAmountLoading(false);
  };

  const getStripeAmount = async (value) => {
    const nextValue = value === undefined ? topUpCount : value;
    setAmountLoading(true);
    try {
      const res = await API.post('/api/user/stripe/amount', {
        amount: parseFloat(nextValue),
      });
      if (res !== undefined) {
        const { message, data } = res.data;
        if (message === 'success') {
          setAmount(parseFloat(data));
        } else {
          setAmount(0);
          Toast.error({ content: '错误：' + data, id: 'getAmount' });
        }
      } else {
        showError(res);
      }
    } catch (err) {
      console.log(err);
    } finally {
      setAmountLoading(false);
    }
  };

  const handleCancel = () => {
    setOpen(false);
  };

  const handleTransferCancel = () => {
    setOpenTransfer(false);
  };

  const handleOpenHistory = () => {
    setOpenHistory(true);
  };

  const handleHistoryCancel = () => {
    setOpenHistory(false);
  };

  const handleCreemCancel = () => {
    setCreemOpen(false);
    setSelectedCreemProduct(null);
  };

  const selectPresetAmount = (preset) => {
    setTopUpCount(preset.value);
    setSelectedPreset(preset.value);
    if (isBEpusdtOnlyMode()) {
      setAmount(parseFloat(preset.value || 0));
      return;
    }
    const discount = preset.discount || topupInfo.discount[preset.value] || 1.0;
    const discountedAmount = preset.value * priceRatio * discount;
    setAmount(discountedAmount);
  };

  const formatLargeNumber = (num) => {
    return num.toString();
  };

  const generatePresetAmounts = (minimumAmount) => {
    const multipliers = [1, 5, 10, 30, 50, 100, 300, 500];
    return multipliers.map((multiplier) => ({
      value: minimumAmount * multiplier,
    }));
  };

  return (
    <div className='w-full max-w-7xl mx-auto relative min-h-screen lg:min-h-0 mt-[60px] px-2'>
      <TransferModal
        t={t}
        openTransfer={openTransfer}
        transfer={transfer}
        handleTransferCancel={handleTransferCancel}
        userState={userState}
        renderQuota={renderQuota}
        getQuotaPerUnit={getQuotaPerUnit}
        transferAmount={transferAmount}
        setTransferAmount={setTransferAmount}
      />

      <PaymentConfirmModal
        t={t}
        open={open}
        onlineTopUp={onlineTopUp}
        handleCancel={handleCancel}
        confirmLoading={confirmLoading}
        topUpCount={topUpCount}
        renderQuotaWithAmount={renderQuotaWithAmount}
        amountLoading={amountLoading}
        renderAmount={renderAmount}
        payWay={payWay}
        payMethods={payMethods}
        amountNumber={amount}
        discountRate={topupInfo?.discount?.[topUpCount] || 1.0}
        summary={getPaymentConfirmSummary()}
      />

      <TopupHistoryModal
        visible={openHistory}
        onCancel={handleHistoryCancel}
        t={t}
      />

      <Modal
        title={t('确定要充值 $')}
        visible={creemOpen}
        onOk={onlineCreemTopUp}
        onCancel={handleCreemCancel}
        maskClosable={false}
        size='small'
        centered
        confirmLoading={confirmLoading}
      >
        {selectedCreemProduct && (
          <>
            <p>
              {t('产品名称')}：{selectedCreemProduct.name}
            </p>
            <p>
              {t('价格')}：
              {selectedCreemProduct.currency === 'EUR' ? '€' : '$'}
              {selectedCreemProduct.price}
            </p>
            <p>
              {t('充值额度')}：{selectedCreemProduct.quota}
            </p>
            <p>{t('是否确认充值？')}</p>
          </>
        )}
      </Modal>

      <div className='grid grid-cols-1 lg:grid-cols-2 gap-6'>
        <RechargeCard
          t={t}
          enableOnlineTopUp={enableOnlineTopUp}
          enableStripeTopUp={enableStripeTopUp}
          enableBTCPayTopUp={enableBTCPayTopUp}
          enableBEpusdtTopUp={enableBEpusdtTopUp}
          bepusdtNetworks={bepusdtNetworks}
          bepusdtTradeType={bepusdtTradeType}
          setBEpusdtTradeType={setBEpusdtTradeType}
          enableNOWPaymentsTopUp={enableNOWPaymentsTopUp}
          nowPaymentsModes={nowPaymentsModes}
          nowPaymentsPricingMode={nowPaymentsPricingMode}
          setNowPaymentsPricingMode={setNowPaymentsPricingMode}
          nowPaymentsNetworks={nowPaymentsNetworks}
          nowPaymentsPayCurrency={nowPaymentsPayCurrency}
          setNowPaymentsPayCurrency={setNowPaymentsPayCurrency}
          nowPaymentsCryptoAmountOptions={nowPaymentsCryptoAmountOptions}
          nowPaymentsCryptoAmount={nowPaymentsCryptoAmount}
          setNowPaymentsCryptoAmount={setNowPaymentsCryptoAmount}
          selectedNowPaymentsCryptoAmount={selectedNowPaymentsCryptoAmount}
          setSelectedNowPaymentsCryptoAmount={
            setSelectedNowPaymentsCryptoAmount
          }
          enableCreemTopUp={enableCreemTopUp}
          creemProducts={creemProducts}
          creemPreTopUp={creemPreTopUp}
          presetAmounts={presetAmounts}
          selectedPreset={selectedPreset}
          selectPresetAmount={selectPresetAmount}
          formatLargeNumber={formatLargeNumber}
          priceRatio={priceRatio}
          topUpCount={topUpCount}
          minTopUp={minTopUp}
          renderQuotaWithAmount={renderQuotaWithAmount}
          getAmount={getAmount}
          setTopUpCount={setTopUpCount}
          setSelectedPreset={setSelectedPreset}
          renderAmount={renderAmount}
          amountLoading={amountLoading}
          payMethods={payMethods}
          preTopUp={preTopUp}
          paymentLoading={paymentLoading}
          payWay={payWay}
          isBEpusdtOnlyMode={isBEpusdtOnlyMode()}
          redemptionCode={redemptionCode}
          setRedemptionCode={setRedemptionCode}
          topUp={topUp}
          isSubmitting={isSubmitting}
          topUpLink={topUpLink}
          openTopUpLink={openTopUpLink}
          userState={userState}
          renderQuota={renderQuota}
          statusLoading={statusLoading}
          topupInfo={topupInfo}
          onOpenHistory={handleOpenHistory}
          subscriptionLoading={subscriptionLoading}
          subscriptionPlans={subscriptionPlans}
          billingPreference={billingPreference}
          onChangeBillingPreference={updateBillingPreference}
          activeSubscriptions={activeSubscriptions}
          allSubscriptions={allSubscriptions}
          reloadSubscriptionSelf={getSubscriptionSelf}
        />
        <InvitationCard
          t={t}
          userState={userState}
          renderQuota={renderQuota}
          setOpenTransfer={setOpenTransfer}
          affLink={affLink}
          handleAffLinkClick={handleAffLinkClick}
        />
      </div>
    </div>
  );
};

export default TopUp;
