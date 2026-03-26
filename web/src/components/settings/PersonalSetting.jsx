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

import React, { useContext, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  API,
  copy,
  showError,
  showInfo,
  showSuccess,
  setStatusData,
  prepareCredentialCreationOptions,
  buildRegistrationResult,
  isPasskeySupported,
  setUserData,
} from '../../helpers';
import { UserContext } from '../../context/User';
import { Modal } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';

// 闂佽娴烽弫鎼佸储瑜斿畷锝夊幢濞嗗苯浜炬繛鎴炵懃婵绱掗鑲┬ｇ紒?import UserInfoHeader from './personal/components/UserInfoHeader';
import AccountManagement from './personal/cards/AccountManagement';
import NotificationSettings from './personal/cards/NotificationSettings';
import PreferencesSettings from './personal/cards/PreferencesSettings';
import CheckinCalendar from './personal/cards/CheckinCalendar';
import EmailBindModal from './personal/modals/EmailBindModal';
import WeChatBindModal from './personal/modals/WeChatBindModal';
import AccountDeleteModal from './personal/modals/AccountDeleteModal';
import ChangePasswordModal from './personal/modals/ChangePasswordModal';

const PersonalSetting = () => {
  const [userState, userDispatch] = useContext(UserContext);
  let navigate = useNavigate();
  const { t } = useTranslation();

  const [inputs, setInputs] = useState({
    wechat_verification_code: '',
    email_verification_code: '',
    email: '',
    self_account_deletion_confirmation: '',
    original_password: '',
    set_new_password: '',
    set_new_password_confirmation: '',
  });
  const [status, setStatus] = useState({});
  const [showChangePasswordModal, setShowChangePasswordModal] = useState(false);
  const [showWeChatBindModal, setShowWeChatBindModal] = useState(false);
  const [showEmailBindModal, setShowEmailBindModal] = useState(false);
  const [showAccountDeleteModal, setShowAccountDeleteModal] = useState(false);
  const [turnstileEnabled, setTurnstileEnabled] = useState(false);
  const [turnstileSiteKey, setTurnstileSiteKey] = useState('');
  const [turnstileToken, setTurnstileToken] = useState('');
  const [turnstileWidgetKey, setTurnstileWidgetKey] = useState(0);
  const [loading, setLoading] = useState(false);
  const [disableButton, setDisableButton] = useState(false);
  const [countdown, setCountdown] = useState(30);
  const [systemToken, setSystemToken] = useState('');
  const [passkeyStatus, setPasskeyStatus] = useState({ enabled: false });
  const [passkeyRegisterLoading, setPasskeyRegisterLoading] = useState(false);
  const [passkeyDeleteLoading, setPasskeyDeleteLoading] = useState(false);
  const [passkeySupported, setPasskeySupported] = useState(false);
  const [notificationSettings, setNotificationSettings] = useState({
    warningType: 'email',
    warningThreshold: 100000,
    webhookUrl: '',
    webhookSecret: '',
    notificationEmail: '',
    barkUrl: '',
    gotifyUrl: '',
    gotifyToken: '',
    gotifyPriority: 5,
    upstreamModelUpdateNotifyEnabled: false,
    acceptUnsetModelRatioModel: false,
    recordIpLog: false,
  });

  useEffect(() => {
    let saved = localStorage.getItem('status');
    if (saved) {
      const parsed = JSON.parse(saved);
      setStatus(parsed);
      if (parsed.turnstile_check) {
        setTurnstileEnabled(true);
        setTurnstileSiteKey(parsed.turnstile_site_key);
      } else {
        setTurnstileEnabled(false);
        setTurnstileSiteKey('');
      }
    }
    // Always refresh status from server to avoid stale flags (e.g., admin just enabled OAuth)
    (async () => {
      try {
        const res = await API.get('/api/status');
        const { success, data } = res.data;
        if (success && data) {
          setStatus(data);
          setStatusData(data);
          if (data.turnstile_check) {
            setTurnstileEnabled(true);
            setTurnstileSiteKey(data.turnstile_site_key);
          } else {
            setTurnstileEnabled(false);
            setTurnstileSiteKey('');
          }
        }
      } catch (e) {
        // ignore and keep local status
      }
    })();

    getUserData();

    isPasskeySupported()
      .then(setPasskeySupported)
      .catch(() => setPasskeySupported(false));
  }, []);

  useEffect(() => {
    let countdownInterval = null;
    if (disableButton && countdown > 0) {
      countdownInterval = setInterval(() => {
        setCountdown(countdown - 1);
      }, 1000);
    } else if (countdown === 0) {
      setDisableButton(false);
      setCountdown(30);
    }
    return () => clearInterval(countdownInterval); // Clean up on unmount
  }, [disableButton, countdown]);

  useEffect(() => {
    if (userState?.user?.setting) {
      const settings = JSON.parse(userState.user.setting);
      setNotificationSettings({
        warningType: settings.notify_type || 'email',
        warningThreshold: settings.quota_warning_threshold || 500000,
        webhookUrl: settings.webhook_url || '',
        webhookSecret: settings.webhook_secret || '',
        notificationEmail: settings.notification_email || '',
        barkUrl: settings.bark_url || '',
        gotifyUrl: settings.gotify_url || '',
        gotifyToken: settings.gotify_token || '',
        gotifyPriority:
          settings.gotify_priority !== undefined ? settings.gotify_priority : 5,
        upstreamModelUpdateNotifyEnabled:
          settings.upstream_model_update_notify_enabled === true,
        acceptUnsetModelRatioModel:
          settings.accept_unset_model_ratio_model || false,
        recordIpLog: settings.record_ip_log || false,
      });
    }
  }, [userState?.user?.setting]);

  const handleInputChange = (name, value) => {
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  };

  const resetTurnstileChallenge = () => {
    setTurnstileToken('');
    setTurnstileWidgetKey((prev) => prev + 1);
  };

  const generateAccessToken = async () => {
    const res = await API.get('/api/user/token');
    const { success, message, data } = res.data;
    if (success) {
      setSystemToken(data);
      await copy(data);
      showSuccess('Token reset and copied to clipboard.');
    } else {
      showError(message);
    }
  };

  const loadPasskeyStatus = async () => {
    try {
      const res = await API.get('/api/user/passkey');
      const { success, data, message } = res.data;
      if (success) {
        setPasskeyStatus({
          enabled: data?.enabled || false,
          last_used_at: data?.last_used_at || null,
          backup_eligible: data?.backup_eligible || false,
          backup_state: data?.backup_state || false,
        });
      } else {
        showError(message);
      }
    } catch (error) {
      // Ignore and keep the current status.
    }
  };

  const handleRegisterPasskey = async () => {
    if (!passkeySupported || !window.PublicKeyCredential) {
      showInfo('This device does not support Passkey.');
      return;
    }
    setPasskeyRegisterLoading(true);
    try {
      const beginRes = await API.post('/api/user/passkey/register/begin');
      const { success, message, data } = beginRes.data;
      if (!success) {
        showError(message || 'Unable to start Passkey registration.');
        return;
      }

      const publicKey = prepareCredentialCreationOptions(
        data?.options || data?.publicKey || data,
      );
      const credential = await navigator.credentials.create({ publicKey });
      const payload = buildRegistrationResult(credential);
      if (!payload) {
        showError('Unable to start Passkey registration.');
        return;
      }

      const finishRes = await API.post(
        '/api/user/passkey/register/finish',
        payload,
      );
      if (finishRes.data.success) {
        showSuccess('Passkey registered successfully.');
        await loadPasskeyStatus();
      } else {
        showError(finishRes.data.message || 'Passkey registration failed. Please try again.');
      }
    } catch (error) {
      if (error?.name === 'AbortError') {
        showInfo('Passkey registration was canceled.');
      } else {
        showError('Passkey registration failed. Please try again.');
      }
    } finally {
      setPasskeyRegisterLoading(false);
    }
  };

  const handleRemovePasskey = async () => {
    setPasskeyDeleteLoading(true);
    try {
      const res = await API.delete('/api/user/passkey');
      const { success, message } = res.data;
      if (success) {
        showSuccess('Passkey removed successfully.');
        await loadPasskeyStatus();
      } else {
        showError(message || 'Operation failed. Please try again.');
      }
    } catch (error) {
      showError('Operation failed. Please try again.');
    } finally {
      setPasskeyDeleteLoading(false);
    }
  };

  const getUserData = async () => {
    let res = await API.get(`/api/user/self`);
    const { success, message, data } = res.data;
    if (success) {
      userDispatch({ type: 'login', payload: data });
      setUserData(data);
      await loadPasskeyStatus();
    } else {
      showError(message);
    }
  };

  const handleSystemTokenClick = async (e) => {
    e.target.select();
    await copy(e.target.value);
    showSuccess('System token copied to clipboard.');
  };

  const deleteAccount = async () => {
    if (inputs.self_account_deletion_confirmation !== userState.user.username) {
      showError('Please enter your account name to confirm deletion.');
      return;
    }

    const res = await API.delete('/api/user/self');
    const { success, message } = res.data;

    if (success) {
      showSuccess('Account deleted successfully.');
      await API.get('/api/user/logout');
      userDispatch({ type: 'logout' });
      localStorage.removeItem('user');
      navigate('/login');
    } else {
      showError(message);
    }
  };

  const bindWeChat = async () => {
    if (inputs.wechat_verification_code === '') return;
    const res = await API.get(
      `/api/oauth/wechat/bind?code=${inputs.wechat_verification_code}`,
    );
    const { success, message } = res.data;
    if (success) {
      showSuccess('WeChat account bound successfully.');
      setShowWeChatBindModal(false);
    } else {
      showError(message);
    }
  };

  const changePassword = async () => {
    // if (inputs.original_password === '') {
    //   showError(t('闂佽崵濮村ú銊╁蓟婢跺本顐芥い鎾卞灩缁€鍌炴煏婢跺牆鍔氶柡鍌冨洦鍊甸梻鍫熺⊕椤ョ娀鏌ｉ弽銊х煉闁?));
    //   return;
    // }
    if (inputs.set_new_password === '') {
      showError('Please enter a new password.');
      return;
    }
    if (inputs.original_password === inputs.set_new_password) {
      showError('The new password must be different from the old password.');
      return;
    }
    if (inputs.set_new_password !== inputs.set_new_password_confirmation) {
      showError('The password confirmation does not match.');
      return;
    }
    const res = await API.put(`/api/user/self`, {
      original_password: inputs.original_password,
      password: inputs.set_new_password,
    });
    const { success, message } = res.data;
    if (success) {
      showSuccess('Password updated successfully.');
      setShowWeChatBindModal(false);
    } else {
      showError(message);
    }
    setShowChangePasswordModal(false);
  };

  const sendVerificationCode = async () => {
    if (inputs.email === '') {
      showError('Please enter your email.');
      return;
    }
    if (turnstileEnabled && turnstileToken === '') {
      showInfo('Please complete Turnstile verification and try again.');
      return;
    }
    setLoading(true);
    try {
      const res = await API.get(
        `/api/verification?email=${encodeURIComponent(inputs.email)}&turnstile=${turnstileToken}`,
      );
      const { success, message } = res.data;
      if (success) {
        setDisableButton(true);
        showSuccess('Verification code sent successfully.');
      } else {
        showError(message);
      }
    } finally {
      if (turnstileEnabled) {
        resetTurnstileChallenge();
      }
      setLoading(false);
    }
  };

  const bindEmail = async () => {
    if (inputs.email_verification_code === '') {
      showError('Please enter the email verification code.');
      return;
    }
    setLoading(true);
    try {
      const res = await API.get(
        `/api/oauth/email/bind?email=${encodeURIComponent(inputs.email)}&code=${encodeURIComponent(inputs.email_verification_code)}`,
      );
      const { success, message } = res.data;
      if (success) {
        showSuccess('Email bound successfully.');
        setShowEmailBindModal(false);
        userState.user.email = inputs.email;
      } else {
        showError(message);
      }
    } finally {
      if (turnstileEnabled) {
        resetTurnstileChallenge();
      }
      setLoading(false);
    }
  };
  const copyText = async (text) => {
    if (await copy(text)) {
      showSuccess('Copied: ' + text);
    } else {
      // setSearchKeyword(text);
      Modal.error({ title: 'Unable to copy to clipboard. Please copy it manually.', content: text });
    }
  };

  const handleNotificationSettingChange = (type, value) => {
    setNotificationSettings((prev) => ({
      ...prev,
      [type]: value.target
        ? value.target.value !== undefined
          ? value.target.value
          : value.target.checked
        : value, // handle checkbox properly
    }));
  };

  const saveNotificationSettings = async () => {
    try {
      const res = await API.put('/api/user/setting', {
        notify_type: notificationSettings.warningType,
        quota_warning_threshold: parseFloat(
          notificationSettings.warningThreshold,
        ),
        webhook_url: notificationSettings.webhookUrl,
        webhook_secret: notificationSettings.webhookSecret,
        notification_email: notificationSettings.notificationEmail,
        bark_url: notificationSettings.barkUrl,
        gotify_url: notificationSettings.gotifyUrl,
        gotify_token: notificationSettings.gotifyToken,
        gotify_priority: (() => {
          const parsed = parseInt(notificationSettings.gotifyPriority);
          return isNaN(parsed) ? 5 : parsed;
        })(),
        upstream_model_update_notify_enabled:
          notificationSettings.upstreamModelUpdateNotifyEnabled === true,
        accept_unset_model_ratio_model:
          notificationSettings.acceptUnsetModelRatioModel,
        record_ip_log: notificationSettings.recordIpLog,
      });

      if (res.data.success) {
        showSuccess('Settings saved successfully.');
        await getUserData();
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError('Failed to save settings.');
    }
  };

  return (
    <div className='mt-[60px]'>
      <div className='flex justify-center'>
        <div className='w-full max-w-7xl mx-auto px-2'>
          {/* 濠碉紕鍋戦崐鏇㈡偉婵傜纾块柟缁㈠枟閸嬨劑鏌曟繝蹇曠暠闁绘挻娲栬彁闁搞儻绲芥晶鎻捗归悡搴㈠殗鐎规洜濞€瀹曨偊宕熼鐐茬 */}
          <UserInfoHeader t={t} userState={userState} />

          {/* 缂傚倷鐒︾粙鎺楀磿閹惰棄鏄ョ€光偓閸曨偆鐫勯梺闈涱槶閸庨亶宕?- 濠电偛顕慨鎾箠鎼粹槄鑰挎い蹇撶墕鐟欙附銇勯弽銊х煁闁哄棗绻橀弻锟犲礃椤撶偟鍘┑鈽嗗灠椤﹁京鍒?*/}
          {status?.checkin_enabled && (
            <div className='mt-4 md:mt-6'>
              <CheckinCalendar
                t={t}
                status={status}
                turnstileEnabled={turnstileEnabled}
                turnstileSiteKey={turnstileSiteKey}
              />
            </div>
          )}

          {/* 闂佽崵濮甸崝褔姊介崟顖氭槬婵炴垶姘ㄦ稉宥夋煥濞戞ê顏柛濠勫仱閺屾稑顭ㄩ崘顓烆伃闂佸憡鐟ч崑鎾剁矉閹烘鍐€妞ゆ帒顦弲顓犵磽?*/}
          <div className='grid grid-cols-1 xl:grid-cols-2 items-start gap-4 md:gap-6 mt-4 md:mt-6'>
            {/* 闁诲骸缍婂鑽ょ磽濮樿泛鐤鹃柛顐ｆ礃閺咁剚鎱ㄥΟ鍝勬毐妞わ腹鏅犻弻鐔煎箻椤曞懏顥栧銈嗘尰閹倿骞冮崼鏇炲耿婵鍘ч弲顓犵磽?*/}
            <div className='flex flex-col gap-4 md:gap-6'>
              <AccountManagement
                t={t}
                userState={userState}
                status={status}
                systemToken={systemToken}
                setShowEmailBindModal={setShowEmailBindModal}
                setShowWeChatBindModal={setShowWeChatBindModal}
                generateAccessToken={generateAccessToken}
                handleSystemTokenClick={handleSystemTokenClick}
                setShowChangePasswordModal={setShowChangePasswordModal}
                setShowAccountDeleteModal={setShowAccountDeleteModal}
                passkeyStatus={passkeyStatus}
                passkeySupported={passkeySupported}
                passkeyRegisterLoading={passkeyRegisterLoading}
                passkeyDeleteLoading={passkeyDeleteLoading}
                onPasskeyRegister={handleRegisterPasskey}
                onPasskeyDelete={handleRemovePasskey}
              />

              {/* 闂備胶顭堥鍛崲閹版澘围闁伙絽鏈刊濂告煕閹炬鎳忛悗顓㈡⒑閹稿海鈽夐柣妤佸姍椤㈡岸濮€閵忊€虫疁闂侀€炲苯澧扮紒顔肩仛瀵板嫬鈽夊槌栨Т */}
              <PreferencesSettings t={t} />
            </div>

            {/* 闂備礁鎲￠悷銉╁储閺嶎厼鐤鹃柛顐ｆ礃閺咁剚鎱ㄥ鍡楀鐎电増鎸搁湁闁绘ê纾晶铏亜閺冣偓濞叉粎妲?*/}
            <NotificationSettings
              t={t}
              notificationSettings={notificationSettings}
              handleNotificationSettingChange={handleNotificationSettingChange}
              saveNotificationSettings={saveNotificationSettings}
            />
          </div>
        </div>
      </div>

      {/* 婵犵妲呴崹顏堝礈濠靛鐒垫い鎴ｆ硶閸斿秵銇勯姀鐙€鍎戠紒杈ㄥ浮瀹曠喖顢旈崪浣镐缓 */}
      <EmailBindModal
        t={t}
        showEmailBindModal={showEmailBindModal}
        setShowEmailBindModal={setShowEmailBindModal}
        inputs={inputs}
        handleInputChange={handleInputChange}
        sendVerificationCode={sendVerificationCode}
        bindEmail={bindEmail}
        disableButton={disableButton}
        loading={loading}
        countdown={countdown}
        turnstileEnabled={turnstileEnabled}
        turnstileSiteKey={turnstileSiteKey}
        turnstileWidgetKey={turnstileWidgetKey}
        setTurnstileToken={setTurnstileToken}
      />

      <WeChatBindModal
        t={t}
        showWeChatBindModal={showWeChatBindModal}
        setShowWeChatBindModal={setShowWeChatBindModal}
        inputs={inputs}
        handleInputChange={handleInputChange}
        bindWeChat={bindWeChat}
        status={status}
      />

      <AccountDeleteModal
        t={t}
        showAccountDeleteModal={showAccountDeleteModal}
        setShowAccountDeleteModal={setShowAccountDeleteModal}
        inputs={inputs}
        handleInputChange={handleInputChange}
        deleteAccount={deleteAccount}
        userState={userState}
        turnstileEnabled={turnstileEnabled}
        turnstileSiteKey={turnstileSiteKey}
        setTurnstileToken={setTurnstileToken}
      />

      <ChangePasswordModal
        t={t}
        showChangePasswordModal={showChangePasswordModal}
        setShowChangePasswordModal={setShowChangePasswordModal}
        inputs={inputs}
        handleInputChange={handleInputChange}
        changePassword={changePassword}
        turnstileEnabled={turnstileEnabled}
        turnstileSiteKey={turnstileSiteKey}
        setTurnstileToken={setTurnstileToken}
      />
    </div>
  );
};

export default PersonalSetting;
