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

import { useMemo } from 'react';

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

export const useNavigation = (t, docsLink, headerNavModules, currentLang) => {
  const mainNavLinks = useMemo(() => {
    const resolvedDocsLink = normalizeDocsLink(docsLink, currentLang);
    const defaultModules = {
      home: true,
      console: true,
      pricing: true,
      docs: true,
      about: true,
    };

    const modules = {
      ...defaultModules,
      ...(headerNavModules || {}),
    };

    const allLinks = [
      {
        text: t('首页'),
        itemKey: 'home',
        to: '/',
      },
      {
        text: t('控制台'),
        itemKey: 'console',
        to: '/console',
      },
      {
        text: t('模型广场'),
        itemKey: 'pricing',
        to: '/pricing',
      },
      ...(resolvedDocsLink
        ? [
            {
              text: t('文档'),
              itemKey: 'docs',
              isExternal: true,
              externalLink: resolvedDocsLink,
            },
          ]
        : []),
      {
        text: t('关于'),
        itemKey: 'about',
        to: '/about',
      },
    ];

    return allLinks.filter((link) => {
      if (link.itemKey === 'docs') {
        return Boolean(resolvedDocsLink) && modules.docs;
      }
      if (link.itemKey === 'pricing') {
        return typeof modules.pricing === 'object'
          ? modules.pricing.enabled
          : modules.pricing;
      }
      return modules[link.itemKey] === true;
    });
  }, [t, docsLink, headerNavModules, currentLang]);

  return {
    mainNavLinks,
  };
};
