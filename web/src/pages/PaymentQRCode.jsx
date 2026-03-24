import React from 'react';
import { Button, Card, Divider, Typography } from '@douyinfe/semi-ui';
import { useLocation } from 'react-router-dom';
import { QRCodeSVG } from 'qrcode.react';
import { useTranslation } from 'react-i18next';
import { copy, loadPaymentCheckout, showError, showSuccess } from '../helpers';

const { Title, Text } = Typography;

function getQRCodeTitle(paymentMethodLabel, content, t) {
  if (paymentMethodLabel) {
    return t('{{paymentMethodLabel}} 扫码支付', {
      paymentMethodLabel,
    });
  }
  if (content.includes('qr.alipay.com') || content.includes('alipays://')) {
    return t('支付宝扫码支付');
  }
  if (content.includes('weixin://')) {
    return t('微信扫码支付');
  }
  return t('请使用手机扫码支付');
}

function renderPayAmount(payAmount, payCurrency) {
  if (!payAmount) {
    return '-';
  }
  if (!payCurrency) {
    return payAmount;
  }
  if (payCurrency === 'CNY') {
    return `¥${payAmount}`;
  }
  return `${payAmount} ${payCurrency}`;
}

function InfoRow({ label, value, mono = false }) {
  return (
    <div className='flex items-start justify-between gap-4 text-left'>
      <Text type='tertiary'>{label}</Text>
      <Text
        strong
        className={`break-all text-right ${mono ? 'font-mono text-xs' : ''}`}
      >
        {value || '-'}
      </Text>
    </div>
  );
}

const PaymentQRCode = () => {
  const { t } = useTranslation();
  const location = useLocation();
  const search = new URLSearchParams(location.search);
  const checkoutId = String(search.get('checkout_id') || '').trim();
  const storedCheckout = loadPaymentCheckout(checkoutId) || {};

  const content = String(
    storedCheckout?.qr_content ||
      storedCheckout?.pay_link ||
      search.get('content') ||
      '',
  ).trim();
  const payLink = String(
    storedCheckout?.pay_link || search.get('link') || '',
  ).trim();
  const subject = String(
    storedCheckout?.subject || search.get('subject') || '',
  ).trim();
  const tradeNo = String(
    storedCheckout?.trade_no || search.get('trade_no') || '',
  ).trim();
  const payAmount = String(
    storedCheckout?.pay_amount || search.get('pay_amount') || '',
  ).trim();
  const payCurrency = String(
    storedCheckout?.pay_currency || search.get('pay_currency') || '',
  ).trim();
  const rechargeAmount = String(
    storedCheckout?.recharge_amount || search.get('recharge_amount') || '',
  ).trim();
  const paymentMethodLabel = String(
    storedCheckout?.payment_method_label ||
      search.get('payment_method_label') ||
      '',
  ).trim();
  const localizedPaymentMethodLabel = paymentMethodLabel
    ? t(paymentMethodLabel)
    : '';
  const title = getQRCodeTitle(
    localizedPaymentMethodLabel,
    content || payLink,
    t,
  );

  const copyContent = async () => {
    const ok = await copy(content || payLink);
    if (ok) {
      showSuccess(t('支付链接已复制'));
      return;
    }
    showError(t('复制支付链接失败'));
  };

  return (
    <div className='min-h-screen bg-slate-100 px-4 py-10'>
      <div className='mx-auto max-w-md'>
        <Card className='!rounded-2xl !shadow-sm'>
          <div className='flex flex-col items-center gap-4 text-center'>
            <Title heading={3} className='!mb-0'>
              {title}
            </Title>

            <div className='w-full rounded-xl bg-slate-50 p-4'>
              <div className='space-y-3'>
                <InfoRow
                  label={t('应付金额')}
                  value={renderPayAmount(payAmount, payCurrency)}
                />
                <InfoRow
                  label={t('支付方式')}
                  value={localizedPaymentMethodLabel || '-'}
                />
                <InfoRow label={t('用途')} value={subject || t('在线支付')} />
                {rechargeAmount ? (
                  <InfoRow label={t('充值面额')} value={rechargeAmount} />
                ) : null}
                <InfoRow label={t('订单号')} value={tradeNo} mono />
              </div>
            </div>

            <Divider margin='4px' />

            {content ? (
              <>
                <div className='rounded-2xl bg-white p-4 shadow-sm'>
                  <QRCodeSVG value={content} size={240} includeMargin />
                </div>
                <Text type='secondary'>
                  {t(
                    '请使用手机钱包扫码完成支付，支付成功后返回原页面刷新即可查看状态。',
                  )}
                </Text>
              </>
            ) : (
              <Text type='danger'>
                {t('缺少支付二维码内容，请重新发起订单。')}
              </Text>
            )}

            <div className='flex w-full gap-3'>
              <Button
                theme='solid'
                type='primary'
                block
                onClick={copyContent}
                disabled={!content && !payLink}
              >
                {t('复制支付链接')}
              </Button>
              <Button
                block
                onClick={() => {
                  if (payLink) {
                    window.open(payLink, '_blank', 'noopener,noreferrer');
                  }
                }}
                disabled={!payLink}
              >
                {t('打开原始链接')}
              </Button>
            </div>
          </div>
        </Card>
      </div>
    </div>
  );
};

export default PaymentQRCode;
