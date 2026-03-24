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
import { Button, Dropdown } from '@douyinfe/semi-ui';
import { Languages } from 'lucide-react';
import { languageOptions } from '../../../i18n/language';

const DEFAULT_MENU_CLASS_NAME =
  '!bg-semi-color-bg-overlay !border-semi-color-border !shadow-lg !rounded-lg dark:!bg-gray-700 dark:!border-gray-600';
const DEFAULT_BUTTON_CLASS_NAME =
  '!p-1.5 !text-current focus:!bg-semi-color-fill-1 dark:focus:!bg-gray-700 !rounded-full !bg-semi-color-fill-0 dark:!bg-semi-color-fill-1 hover:!bg-semi-color-fill-1 dark:hover:!bg-semi-color-fill-2';

const LanguageSelector = ({
  currentLang,
  onLanguageChange,
  t,
  menuClassName = '',
  buttonClassName = '',
}) => {
  const mergedMenuClassName = [DEFAULT_MENU_CLASS_NAME, menuClassName]
    .filter(Boolean)
    .join(' ');
  const mergedButtonClassName = [DEFAULT_BUTTON_CLASS_NAME, buttonClassName]
    .filter(Boolean)
    .join(' ');

  return (
    <Dropdown
      position='bottomRight'
      render={
        <Dropdown.Menu className={mergedMenuClassName}>
          {languageOptions.map((language) => (
            <Dropdown.Item
              key={language.value}
              onClick={() => onLanguageChange(language.value)}
              className={`!px-3 !py-1.5 !text-sm !text-semi-color-text-0 dark:!text-gray-200 ${currentLang === language.value ? '!bg-semi-color-primary-light-default dark:!bg-blue-600 !font-semibold' : 'hover:!bg-semi-color-fill-1 dark:hover:!bg-gray-600'}`}
            >
              {language.label}
            </Dropdown.Item>
          ))}
        </Dropdown.Menu>
      }
    >
      <Button
        icon={<Languages size={18} />}
        aria-label={t('common.changeLanguage')}
        theme='borderless'
        type='tertiary'
        className={mergedButtonClassName}
      />
    </Dropdown>
  );
};

export default LanguageSelector;
