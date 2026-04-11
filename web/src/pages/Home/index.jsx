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

import React, { useContext, useEffect, useMemo, useState } from 'react';
import { Avatar, Button, Typography } from '@douyinfe/semi-ui';
import { IconCopy, IconFile, IconPlay } from '@douyinfe/semi-icons';
import { Link, useLocation } from 'react-router-dom';
import { marked } from 'marked';
import { useTranslation } from 'react-i18next';
import {
  API,
  copy,
  getLogo,
  getSystemName,
  showError,
  showSuccess,
  stringToColor,
} from '../../helpers';
import { API_ENDPOINTS } from '../../constants/common.constant';
import { StatusContext } from '../../context/Status';
import { UserContext } from '../../context/User';
import { useActualTheme } from '../../context/Theme';
import LanguageSelector from '../../components/layout/headerbar/LanguageSelector';
import { useLanguagePreference } from '../../hooks/common/useLanguagePreference';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { useNavigation } from '../../hooks/common/useNavigation';
import NoticeModal from '../../components/layout/NoticeModal';
import {
  OpenAI,
  Claude,
  Gemini,
  DeepSeek,
  Qwen,
  Grok,
  Midjourney,
  AzureAI,
} from '@lobehub/icons';
import './home.css';

const DEFAULT_DOCS_LINK = '/docs-home.html?v=20260325-220151';

const normalizeDocsLang = (lang) => {
  if (!lang) {
    return 'en';
  }
  return String(lang).toLowerCase() === 'en' ? 'en' : 'zh-CN';
};

const normalizeDocsLink = (docsLink, lang) => {
  const docsLang = normalizeDocsLang(lang);
  const appendLang = (rawLink) => {
    try {
      const url = new URL(rawLink, window.location.origin);
      url.searchParams.set('lang', docsLang);
      if (url.origin === window.location.origin) {
        return `${url.pathname}${url.search}${url.hash}`;
      }
      return url.toString();
    } catch (error) {
      return rawLink;
    }
  };

  if (!docsLink) {
    return appendLang(DEFAULT_DOCS_LINK);
  }
  if (docsLink.startsWith('/docs-home.html') && !docsLink.includes('?')) {
    return appendLang(`${docsLink}?v=20260325-220151`);
  }
  return appendLang(docsLink);
};

const { Text } = Typography;

const PROVIDERS = [
  {
    name: 'OpenAI',
    renderIcon: () => <OpenAI size={28} />,
  },
  {
    name: 'Claude',
    renderIcon: () => <Claude.Color size={28} />,
  },
  {
    name: 'Gemini',
    renderIcon: () => <Gemini.Color size={28} />,
  },
  {
    name: 'DeepSeek',
    renderIcon: () => <DeepSeek.Color size={28} />,
  },
  {
    name: 'Qwen',
    renderIcon: () => <Qwen.Color size={28} />,
  },
  {
    name: 'Grok',
    renderIcon: () => <Grok size={28} />,
  },
  {
    name: 'Midjourney',
    renderIcon: () => <Midjourney size={28} />,
  },
  {
    name: 'Azure AI',
    renderIcon: () => <AzureAI.Color size={28} />,
  },
];

const Home = () => {
  const { t, i18n } = useTranslation();
  const location = useLocation();
  const [userState] = useContext(UserContext);
  const [statusState] = useContext(StatusContext);
  const actualTheme = useActualTheme();
  const isMobile = useIsMobile();
  const { currentLang, handleLanguageChange } = useLanguagePreference();
  const [homePageContentLoaded, setHomePageContentLoaded] = useState(false);
  const [homePageContent, setHomePageContent] = useState('');
  const [noticeVisible, setNoticeVisible] = useState(false);

  const logo = getLogo();
  const systemName = getSystemName();
  const docsLink = statusState?.status?.docs_link || '';
  const resolvedDocsLink = normalizeDocsLink(docsLink, currentLang);
  const serverAddress =
    statusState?.status?.server_address || `${window.location.origin}`;
  const isSelfUseMode = statusState?.status?.self_use_mode_enabled || false;

  const headerNavModules = useMemo(() => {
    const rawModules = statusState?.status?.HeaderNavModules;
    if (!rawModules) {
      return null;
    }

    try {
      const parsedModules = JSON.parse(rawModules);
      if (typeof parsedModules.pricing === 'boolean') {
        parsedModules.pricing = {
          enabled: parsedModules.pricing,
          requireAuth: false,
        };
      }
      return parsedModules;
    } catch (error) {
      console.error('Failed to parse header modules:', error);
      return null;
    }
  }, [statusState?.status?.HeaderNavModules]);

  const pricingRequireAuth = useMemo(() => {
    if (!headerNavModules?.pricing) {
      return false;
    }
    return typeof headerNavModules.pricing === 'object'
      ? headerNavModules.pricing.requireAuth
      : false;
  }, [headerNavModules]);

  const { mainNavLinks } = useNavigation(
    t,
    docsLink,
    headerNavModules,
    currentLang,
  );

  const endpointPath = API_ENDPOINTS[0];
  const normalizedServerAddress = serverAddress.endsWith('/')
    ? serverAddress.slice(0, -1)
    : serverAddress;
  const fullEndpoint = `${normalizedServerAddress}${endpointPath}`;
  const consoleTarget = userState?.user ? '/console' : '/login';
  const pricingTarget =
    pricingRequireAuth && !userState?.user ? '/login' : '/pricing';
  const profileSubtitle =
    userState?.user?.role >= 10 ? t('管理员') : t('控制台');
  const currentYear = new Date().getFullYear();
  const showLandingPage = homePageContentLoaded && homePageContent === '';
  const homePageContentCacheKey = `home_page_content:${i18n.language}`;
  const noticeCloseDateKey = `notice_close_date:${i18n.language}`;

  const displayHomePageContent = async () => {
    setHomePageContentLoaded(false);
    setHomePageContent(localStorage.getItem(homePageContentCacheKey) || '');

    try {
      const res = await API.get('/api/home_page_content');
      const { success, message, data } = res.data;

      if (!success) {
        showError(message);
        setHomePageContent(t('加载首页内容失败...'));
        return;
      }

      let content = data;
      if (!data.startsWith('https://')) {
        content = marked.parse(data);
      }

      setHomePageContent(content);
      localStorage.setItem(homePageContentCacheKey, content);

      if (data.startsWith('https://')) {
        const iframe = document.querySelector('iframe');
        if (iframe) {
          iframe.onload = () => {
            iframe.contentWindow.postMessage({ themeMode: actualTheme }, '*');
            iframe.contentWindow.postMessage({ lang: i18n.language }, '*');
          };
        }
      }
    } catch (error) {
      console.error('Failed to load home page content:', error);
      setHomePageContent(t('加载首页内容失败...'));
    } finally {
      setHomePageContentLoaded(true);
    }
  };

  const handleCopyEndpoint = async () => {
    const ok = await copy(fullEndpoint);
    if (ok) {
      showSuccess(t('已复制到剪切板'));
    }
  };

  useEffect(() => {
    const checkNoticeAndShow = async () => {
      const lastCloseDate = localStorage.getItem(noticeCloseDateKey);
      const today = new Date().toDateString();
      if (lastCloseDate === today) {
        return;
      }

      try {
        const res = await API.get('/api/notice');
        const { success, data } = res.data;
        if (success && data && data.trim() !== '') {
          setNoticeVisible(true);
        }
      } catch (error) {
        console.error('Failed to load notice:', error);
      }
    };

    checkNoticeAndShow();
  }, [noticeCloseDateKey]);

  useEffect(() => {
    displayHomePageContent().then();
  }, [homePageContentCacheKey]);

  useEffect(() => {
    document.body.classList.toggle('home-landing-body', showLandingPage);
    document.documentElement.classList.toggle(
      'home-landing-html',
      showLandingPage,
    );

    return () => {
      document.body.classList.remove('home-landing-body');
      document.documentElement.classList.remove('home-landing-html');
    };
  }, [showLandingPage]);

  const renderHeaderLink = (link) => {
    const isHomeLink =
      link.itemKey === 'home' && location.pathname === '/' && !link.isExternal;

    const className = `home-landing__nav-link${isHomeLink ? ' is-active' : ''}`;

    if (link.isExternal) {
      return (
        <a
          key={link.itemKey}
          href={link.externalLink}
          target='_blank'
          rel='noreferrer'
          className={className}
        >
          {link.text}
        </a>
      );
    }

    let targetPath = link.to;
    if (link.itemKey === 'console' && !userState?.user) {
      targetPath = '/login';
    }
    if (link.itemKey === 'pricing' && pricingRequireAuth && !userState?.user) {
      targetPath = '/login';
    }

    return (
      <Link key={link.itemKey} to={targetPath} className={className}>
        {link.text}
      </Link>
    );
  };

  const renderTopActions = () => {
    const languageSelector = (
      <LanguageSelector
        currentLang={currentLang}
        onLanguageChange={handleLanguageChange}
        t={t}
        menuClassName='home-landing__language-menu'
        buttonClassName='home-landing__language-button'
      />
    );

    if (userState?.user) {
      return (
        <div className='home-landing__action-group'>
          {languageSelector}
          <Link to='/console/personal' className='home-landing__profile'>
            <Avatar
              size='small'
              color={stringToColor(userState.user.username || 'U')}
            >
              {(userState.user.username || 'U')[0].toUpperCase()}
            </Avatar>
            <div className='home-landing__profile-meta'>
              <span className='home-landing__profile-name'>
                {userState.user.username}
              </span>
              <span className='home-landing__profile-role'>
                {profileSubtitle}
              </span>
            </div>
          </Link>
        </div>
      );
    }

    return (
      <div className='home-landing__action-group'>
        {languageSelector}
        <div className='home-landing__auth-actions'>
        <Link to='/login'>
          <Button
            theme='borderless'
            className='home-landing__ghost-button home-landing__header-button'
          >
            {t('登录')}
          </Button>
        </Link>
        {!isSelfUseMode && (
          <Link to='/register'>
            <Button
              theme='solid'
              type='primary'
              className='home-landing__header-button home-landing__gradient-button'
            >
              {t('注册')}
            </Button>
          </Link>
        )}
        </div>
      </div>
    );
  };

  const renderSecondaryAction = () => {
    const hasDocsNav = mainNavLinks.some((link) => link.itemKey === 'docs');
    const hasPricingNav = mainNavLinks.some(
      (link) => link.itemKey === 'pricing',
    );

    if (resolvedDocsLink && hasDocsNav) {
      return (
        <a href={resolvedDocsLink} target='_blank' rel='noreferrer'>
          <Button
            icon={<IconFile />}
            className='home-landing__secondary-action'
            size={isMobile ? 'default' : 'large'}
          >
            {t('开发者文档')}
          </Button>
        </a>
      );
    }

    if (!hasPricingNav) {
      return null;
    }

    return (
      <Link to={pricingTarget}>
        <Button
          icon={<IconFile />}
          className='home-landing__secondary-action'
          size={isMobile ? 'default' : 'large'}
        >
          {t('模型广场')}
        </Button>
      </Link>
    );
  };

  return (
    <div className='w-full overflow-x-hidden'>
      <NoticeModal
        visible={noticeVisible}
        onClose={() => setNoticeVisible(false)}
        isMobile={isMobile}
      />

      {!showLandingPage && (
        <div className='home-language-floating'>
          <LanguageSelector
            currentLang={currentLang}
            onLanguageChange={handleLanguageChange}
            t={t}
            menuClassName='home-landing__language-menu'
            buttonClassName='home-landing__language-button'
          />
        </div>
      )}

      {showLandingPage ? (
        <div className='home-landing'>
          <div className='home-landing__grid' />
          <div className='home-landing__glow home-landing__glow--top' />
          <div className='home-landing__glow home-landing__glow--center' />

          <header className='home-landing__header'>
            <div className='home-landing__header-inner'>
              <Link to='/' className='home-landing__brand'>
                <img
                  src={logo}
                  alt={systemName}
                  className='home-landing__brand-logo'
                />
                <span className='home-landing__brand-name'>{systemName}</span>
              </Link>

              <nav className='home-landing__nav'>
                {mainNavLinks.map(renderHeaderLink)}
              </nav>

              <div className='home-landing__header-actions'>
                {renderTopActions()}
              </div>
            </div>
          </header>

          <main className='home-landing__hero'>
            <div className='home-landing__badge'>
              {t('一体化 AI 网关与模型接入')}
            </div>

            <h1 className='home-landing__title'>
              <span>{t('连接你的')}</span>
              <span className='home-landing__title-gradient'>
                {t('AI 模型统一入口')}
              </span>
            </h1>

            <Text className='home-landing__description'>
              {t(
                '一个支持多模型、多供应商与统一计费的 AI 接口网关，帮助你把 OpenAI 兼容协议接入、模型路由和额度管理收敛到同一套系统中。',
              )}
            </Text>

            <div className='home-landing__endpoint-panel'>
              <div className='home-landing__endpoint-prefix'>
                {t('接口地址')}
              </div>
              <code className='home-landing__endpoint-value'>
                {fullEndpoint}
              </code>
              <Button
                theme='solid'
                type='primary'
                icon={<IconCopy />}
                onClick={handleCopyEndpoint}
                className='home-landing__copy-button home-landing__gradient-button'
                size={isMobile ? 'default' : 'large'}
              >
                {t('复制接口地址')}
              </Button>
            </div>

            <div className='home-landing__hero-actions'>
              <Link to={consoleTarget}>
                <Button
                  icon={<IconPlay />}
                  className='home-landing__primary-action'
                  theme='solid'
                  type='primary'
                  size={isMobile ? 'default' : 'large'}
                >
                  {t('进入控制台')}
                </Button>
              </Link>
              {renderSecondaryAction()}
            </div>

            <div className='home-landing__hero-meta'>
              <span>
                {t('适合统一接入 OpenAI 兼容协议、鉴权与额度管理')}
              </span>
              <span className='home-landing__meta-divider' />
              <span>
                {t(
                  '支持 OpenAI、Claude、Gemini、DeepSeek、Grok、Midjourney 等主流模型',
                )}
              </span>
            </div>
          </main>

          <section className='home-landing__providers'>
            {PROVIDERS.map((provider) => (
              <div key={provider.name} className='home-landing__provider-card'>
                <div className='home-landing__provider-icon'>
                  {provider.renderIcon()}
                </div>
                <span className='home-landing__provider-name'>
                  {provider.name}
                </span>
              </div>
            ))}
          </section>

          <footer className='home-landing__footer'>
            © {currentYear} {systemName}
          </footer>
        </div>
      ) : (
        <div className='overflow-x-hidden w-full'>
          {homePageContent.startsWith('https://') ? (
            <iframe
              src={homePageContent}
              className='w-full h-screen border-none'
            />
          ) : (
            <div
              className='mt-[60px]'
              dangerouslySetInnerHTML={{ __html: homePageContent }}
            />
          )}
        </div>
      )}
    </div>
  );
};

export default Home;
