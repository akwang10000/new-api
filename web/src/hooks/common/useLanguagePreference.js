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

import { useCallback, useContext, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { UserContext } from '../../context/User';
import { API } from '../../helpers';
import { defaultLanguage, normalizeLanguage } from '../../i18n/language';

export const useLanguagePreference = () => {
  const { i18n } = useTranslation();
  const [userState, userDispatch] = useContext(UserContext);
  const [currentLang, setCurrentLang] = useState(
    normalizeLanguage(i18n.language) || defaultLanguage,
  );

  useEffect(() => {
    const handleLanguageChanged = (lng) => {
      const normalizedLang = normalizeLanguage(lng);
      setCurrentLang(normalizedLang);
      try {
        const iframe = document.querySelector('iframe');
        const cw = iframe && iframe.contentWindow;
        if (cw) {
          cw.postMessage({ lang: normalizedLang }, '*');
        }
      } catch (e) {
        // Silently ignore cross-origin or access errors.
      }
    };

    i18n.on('languageChanged', handleLanguageChanged);
    return () => {
      i18n.off('languageChanged', handleLanguageChanged);
    };
  }, [i18n]);

  const handleLanguageChange = useCallback(
    async (lang) => {
      const normalizedLang = normalizeLanguage(lang);
      const previousLang = normalizeLanguage(i18n.language) || defaultLanguage;

      i18n.changeLanguage(normalizedLang);
      localStorage.setItem('i18nextLng', normalizedLang);

      if (userState?.user?.id) {
        try {
          const res = await API.put('/api/user/self', {
            language: normalizedLang,
          });
          if (res.data.success) {
            let settings = {};
            if (userState?.user?.setting) {
              try {
                settings = JSON.parse(userState.user.setting) || {};
              } catch (e) {
                settings = {};
              }
            }

            settings.language = normalizedLang;
            const nextUser = {
              ...userState.user,
              setting: JSON.stringify(settings),
            };

            userDispatch({
              type: 'login',
              payload: nextUser,
            });
            localStorage.setItem('user', JSON.stringify(nextUser));
          }
        } catch (error) {
          i18n.changeLanguage(previousLang);
          localStorage.setItem('i18nextLng', previousLang);
          console.error('Failed to save language preference:', error);
        }
      }
    },
    [i18n, userDispatch, userState],
  );

  return {
    currentLang,
    handleLanguageChange,
  };
};
