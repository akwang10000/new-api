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

import React, { useMemo } from 'react';
import { useTokenKeys } from '../../hooks/chat/useTokenKeys';
import {
  Banner,
  Button,
  Card,
  Input,
  Space,
  Spin,
  Typography,
} from '@douyinfe/semi-ui';
import { useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  buildChatIntegrationLink,
  copy,
  isWebChatIntegrationLink,
  showError,
  showSuccess,
} from '../../helpers';

const getChatConfig = (id) => {
  const chats = localStorage.getItem('chats');
  if (!chats) return null;

  try {
    const parsed = JSON.parse(chats);
    const item = parsed?.[Number(id)];
    const name = item ? Object.keys(item)[0] : '';
    const template = name ? item[name] : '';
    if (!name || !template) return null;
    return { name, template };
  } catch (_) {
    return null;
  }
};

const ChatPage = () => {
  const { t } = useTranslation();
  const { id } = useParams();
  const { keys, serverAddress, isLoading } = useTokenKeys(id);
  const chatConfig = useMemo(() => getChatConfig(id), [id]);

  const link = useMemo(() => {
    if (!chatConfig || !keys[0] || !serverAddress) return '';
    return buildChatIntegrationLink(chatConfig.template, keys[0], serverAddress);
  }, [chatConfig, keys, serverAddress]);

  const openLink = () => {
    if (!link) return;
    if (isWebChatIntegrationLink(link)) {
      window.open(link, '_blank', 'noopener,noreferrer');
    } else {
      window.location.href = link;
    }
  };

  const copyLink = async () => {
    if (!link) return;
    if (await copy(link)) {
      showSuccess(t('已复制链接'));
    } else {
      showError(t('复制失败，请手动复制'));
    }
  };

  if (isLoading) {
    return (
      <div className='min-h-[calc(100vh-112px)] flex items-center justify-center'>
        <div className='flex flex-col items-center'>
          <Spin size='large' spinning={true} tip={null} />
          <span
            className='whitespace-nowrap mt-2 text-center'
            style={{ color: 'var(--semi-color-primary)' }}
          >
            {t('正在准备聊天客户端链接...')}
          </span>
        </div>
      </div>
    );
  }

  if (!chatConfig || !link) {
    return (
      <div className='mt-[96px] mx-auto px-4' style={{ maxWidth: 720 }}>
        <Card>
          <Typography.Title heading={4}>
            {t('聊天客户端配置不可用')}
          </Typography.Title>
          <Typography.Paragraph>
            {t('请到系统设置中的聊天设置检查该入口是否存在。')}
          </Typography.Paragraph>
        </Card>
      </div>
    );
  }

  return (
    <div className='min-h-[calc(100vh-112px)] flex items-center justify-center'>
      <Card style={{ width: 'min(720px, calc(100vw - 32px))' }}>
        <Space vertical align='start' style={{ width: '100%' }}>
          <Typography.Title heading={4} style={{ margin: 0 }}>
            {t('打开 {{name}}', { name: chatConfig.name })}
          </Typography.Title>
          <Banner
            type='info'
            description={t(
              '这些入口不是前端内置聊天功能，而是第三方客户端的导入配置链接。桌面客户端需要本机已安装对应应用；网页客户端会在新标签页打开。',
            )}
          />
          <Input value={link} readOnly />
          <Space>
            <Button type='primary' onClick={openLink}>
              {isWebChatIntegrationLink(link)
                ? t('打开网页客户端')
                : t('打开本机客户端')}
            </Button>
            <Button onClick={copyLink}>{t('复制链接')}</Button>
          </Space>
        </Space>
      </Card>
    </div>
  );
};

export default ChatPage;
