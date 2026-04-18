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

import { useEffect } from 'react';

const CHATWOOT_SCRIPT_ID = 'chatwoot-sdk-script';

const normalizeBaseURL = (baseURL) => (baseURL || '').replace(/\/+$/, '');

export default function ChatwootWidget({ config }) {
  useEffect(() => {
    const baseUrl = normalizeBaseURL(config?.base_url);
    const websiteToken = config?.website_token;

    if (!baseUrl || !websiteToken) {
      window.$chatwoot?.toggleBubbleVisibility?.('hide');
      return;
    }

    const configKey = `${baseUrl}|${websiteToken}`;
    if (window.__newApiChatwootConfigKey === configKey) {
      window.$chatwoot?.toggleBubbleVisibility?.('show');
      return;
    }

    window.__newApiChatwootConfigKey = configKey;
    window.chatwootSettings = {
      hideMessageBubble: false,
      position: 'right',
      locale: 'zh_CN',
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
  }, [config?.base_url, config?.website_token]);

  return null;
}
