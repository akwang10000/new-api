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

import { useContext, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { UserContext } from '../../context/User';
import { API } from '../../helpers';
import { normalizeLanguage } from '../../i18n/language';

const CHATWOOT_SCRIPT_ID = 'chatwoot-sdk-script';

const normalizeBaseURL = (baseURL) => (baseURL || '').replace(/\/+$/, '');

const getChatwootLocale = (language) => {
  const raw = (language || '').toLowerCase();
  if (raw.startsWith('fr')) return 'fr';
  if (raw.startsWith('ru')) return 'ru';
  if (raw.startsWith('ja')) return 'ja';
  if (raw.startsWith('vi')) return 'vi';

  const normalized = normalizeLanguage(language);
  if (normalized === 'zh-CN') return 'zh_CN';
  if (normalized === 'zh-TW') return 'zh_TW';
  return 'en';
};

const getLauncherTitle = (language) => {
  const raw = (language || '').toLowerCase();
  if (raw.startsWith('fr')) return 'Support en ligne';
  if (raw.startsWith('ru')) return 'Онлайн-поддержка';
  if (raw.startsWith('ja')) return 'オンラインサポート';
  if (raw.startsWith('vi')) return 'Ho tro truc tuyen';

  const normalized = normalizeLanguage(language);
  if (normalized === 'zh-TW') return '線上客服';
  if (normalized === 'zh-CN') return '在线客服';
  return 'Live chat';
};

const getChatwootName = (user) => user?.username || user?.display_name || '';

const normalizeChatwootUser = (user) => {
  if (!user?.id || !user?.chatwoot_identifier_hash) return null;

  const name = getChatwootName(user);
  return {
    identifier: user.chatwoot_identifier || String(user.id),
    identifierHash: user.chatwoot_identifier_hash,
    email: user.email || undefined,
    name: name || undefined,
    customAttributes: {
      routeropenai_user_id: user.id,
      routeropenai_username: user.username || '',
      routeropenai_display_name: user.display_name || '',
    },
  };
};

export default function ChatwootWidget({ config }) {
  const { i18n } = useTranslation();
  const [userState] = useContext(UserContext);
  const [chatwootUser, setChatwootUser] = useState(null);
  const language = normalizeLanguage(i18n.language);
  const locale = useMemo(() => getChatwootLocale(language), [language]);
  const launcherTitle = useMemo(() => getLauncherTitle(language), [language]);

  useEffect(() => {
    const loadChatwootUser = async () => {
      if (!config?.base_url || !config?.website_token || !userState?.user) {
        setChatwootUser(null);
        return;
      }

      try {
        const response = await API.get('/api/user/self', {
          disableDuplicate: true,
        });
        if (response.data?.success) {
          setChatwootUser(normalizeChatwootUser(response.data.data));
        }
      } catch (_) {
        setChatwootUser(null);
      }
    };

    loadChatwootUser();
  }, [config?.base_url, config?.website_token, userState?.user?.id]);

  useEffect(() => {
    const baseUrl = normalizeBaseURL(config?.base_url);
    const websiteToken = config?.website_token;

    if (!baseUrl || !websiteToken) {
      window.$chatwoot?.toggleBubbleVisibility?.('hide');
      return;
    }

    const configKey = `${baseUrl}|${websiteToken}|${locale}|${launcherTitle}`;
    if (window.__newApiChatwootConfigKey === configKey) {
      window.$chatwoot?.toggleBubbleVisibility?.('show');
      return;
    }

    window.__newApiChatwootConfigKey = configKey;
    window.chatwootSettings = {
      hideMessageBubble: false,
      position: 'right',
      type: 'expanded_bubble',
      launcherTitle,
      locale,
      useBrowserLanguage: false,
    };

    const runChatwoot = () => {
      if (!window.chatwootSDK?.run) return;
      window.chatwootSDK.run({
        websiteToken,
        baseUrl,
      });
    };

    const existingScript = document.getElementById(CHATWOOT_SCRIPT_ID);
    if (existingScript) {
      runChatwoot();
      return;
    }

    const script = document.createElement('script');
    script.id = CHATWOOT_SCRIPT_ID;
    script.src = `${baseUrl}/packs/js/sdk.js`;
    script.defer = true;
    script.async = true;
    script.onload = runChatwoot;
    document.body.appendChild(script);
  }, [config?.base_url, config?.website_token, launcherTitle, locale]);

  useEffect(() => {
    if (!chatwootUser) {
      window.$chatwoot?.reset?.();
      return;
    }

    const applyUser = () => {
      if (!window.$chatwoot?.setUser) return;
      window.$chatwoot.setUser(chatwootUser.identifier, {
        email: chatwootUser.email,
        name: chatwootUser.name,
        identifier_hash: chatwootUser.identifierHash,
      });
      window.$chatwoot?.setCustomAttributes?.(chatwootUser.customAttributes);
    };

    applyUser();
    window.addEventListener('chatwoot:ready', applyUser);
    return () => {
      window.removeEventListener('chatwoot:ready', applyUser);
    };
  }, [chatwootUser]);

  return null;
}
