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

import React from 'react';
import { describe, expect, it, beforeEach, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import ChatwootWidget from './ChatwootWidget';
import { UserContext } from '../../context/User';

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    i18n: {
      language: 'zh-CN',
    },
  }),
}));

vi.mock('../../helpers', () => ({
  API: {
    get: vi.fn(),
  },
}));

const { API } = await import('../../helpers');

const config = {
  base_url: 'https://chat.example.com',
  website_token: 'token-123',
};

const renderWidget = (user) =>
  render(
    <UserContext.Provider value={[{ user }, vi.fn()]}>
      <ChatwootWidget config={config} />
    </UserContext.Provider>,
  );

describe('ChatwootWidget', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.open = vi.fn();
    window.$chatwoot = {
      reset: vi.fn(),
      toggleBubbleVisibility: vi.fn(),
      toggle: vi.fn(),
      setUser: vi.fn(),
      setCustomAttributes: vi.fn(),
    };
    window.chatwootSDK = {
      run: vi.fn(),
    };
    window.__newApiChatwootConfigKey = undefined;
    window.chatwootSettings = undefined;
    document.body.innerHTML = '';
  });

  it('does not reset chatwoot before logged-in user finishes loading', async () => {
    API.get.mockResolvedValue({
      data: {
        success: true,
        data: {
          id: 1,
          username: 'alice',
          chatwoot_identifier: 'cw-1',
          chatwoot_identifier_hash: 'hash-1',
        },
      },
    });

    renderWidget({ id: 1, username: 'alice' });

    await waitFor(() => {
      expect(API.get).toHaveBeenCalledWith('/api/user/self', {
        disableDuplicate: true,
      });
    });

    expect(window.$chatwoot.reset).not.toHaveBeenCalled();
  });

  it('opens chatwoot when clicking the custom launcher', async () => {
    API.get.mockResolvedValue({
      data: {
        success: true,
        data: {
          id: 1,
          username: 'alice',
          chatwoot_identifier: 'cw-1',
          chatwoot_identifier_hash: 'hash-1',
        },
      },
    });

    renderWidget({ id: 1, username: 'alice' });

    const button = await screen.findByRole('button', { name: '在线客服' });
    fireEvent.click(button);

    expect(window.$chatwoot.toggle).toHaveBeenCalledWith('toggle');
    expect(window.open).not.toHaveBeenCalled();
  });
});
