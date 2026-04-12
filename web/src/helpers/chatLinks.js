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

import { encodeToBase64 } from './base64';

const withSkPrefix = (key) => {
  if (!key) return '';
  return key.startsWith('sk-') ? key : `sk-${key}`;
};

const encodeJsonQueryParam = (url, paramName) => {
  const marker = `${paramName}=`;
  const markerIndex = url.indexOf(marker);
  if (markerIndex < 0) return url;

  const valueStart = markerIndex + marker.length;
  const valueEnd = url.indexOf('&', valueStart);
  const rawValue =
    valueEnd === -1 ? url.slice(valueStart) : url.slice(valueStart, valueEnd);

  if (!rawValue.trim().startsWith('{')) return url;

  const encodedValue = encodeURIComponent(rawValue);
  return valueEnd === -1
    ? `${url.slice(0, valueStart)}${encodedValue}`
    : `${url.slice(0, valueStart)}${encodedValue}${url.slice(valueEnd)}`;
};

export const isWebChatIntegrationLink = (url) => /^https?:\/\//i.test(url);

export const buildChatIntegrationLink = (template, key, serverAddress) => {
  if (!template || !key || !serverAddress) return '';

  const apiKey = withSkPrefix(key);
  let url = template;

  if (url.includes('{cherryConfig}')) {
    const cherryConfig = {
      id: 'new-api',
      baseUrl: serverAddress,
      apiKey,
    };
    return url.replaceAll(
      '{cherryConfig}',
      encodeURIComponent(encodeToBase64(JSON.stringify(cherryConfig))),
    );
  }

  if (url.includes('{aionuiConfig}')) {
    const aionuiConfig = {
      platform: 'new-api',
      baseUrl: serverAddress,
      apiKey,
    };
    return url.replaceAll(
      '{aionuiConfig}',
      encodeURIComponent(encodeToBase64(JSON.stringify(aionuiConfig))),
    );
  }

  url = url.replaceAll('{address}', serverAddress);
  url = url.replaceAll('{key}', apiKey);
  url = encodeJsonQueryParam(url, 'settings');
  url = encodeJsonQueryParam(url, 'provider');
  return url;
};
