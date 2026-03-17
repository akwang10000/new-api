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

import React, { useState, useCallback, useMemo, useEffect } from 'react';
import {
  Button,
  Table,
  Tag,
  Empty,
  Checkbox,
  Form,
  Input,
  Tooltip,
  Select,
  Modal,
} from '@douyinfe/semi-ui';
import { IconSearch } from '@douyinfe/semi-icons';
import {
  RefreshCcw,
  CheckSquare,
  AlertTriangle,
  CheckCircle,
} from 'lucide-react';
import {
  API,
  showError,
  showSuccess,
  showWarning,
  stringToColor,
} from '../../../helpers';
import { useIsMobile } from '../../../hooks/common/useIsMobile';
import { DEFAULT_ENDPOINT } from '../../../constants';
import { useTranslation } from 'react-i18next';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import ChannelSelectorModal from '../../../components/settings/ChannelSelectorModal';

const OFFICIAL_RATIO_PRESET_ID = -100;
const OFFICIAL_RATIO_PRESET_NAME = '官方倍率预设';
const OFFICIAL_RATIO_PRESET_BASE_URL = 'https://basellm.github.io';
const OFFICIAL_RATIO_PRESET_ENDPOINT =
  '/llm-metadata/api/newapi/ratio_config-v1-base.json';
const MODELS_DEV_PRESET_ID = -101;
const MODELS_DEV_PRESET_NAME = 'models.dev 价格预设';
const MODELS_DEV_PRESET_BASE_URL = 'https://models.dev';
const MODELS_DEV_PRESET_ENDPOINT = 'https://models.dev/api.json';

function ConflictConfirmModal({ t, visible, items, onOk, onCancel }) {
  const isMobile = useIsMobile();
  const columns = [
    { title: t('渠道'), dataIndex: 'channel' },
    { title: t('模型'), dataIndex: 'model' },
    {
      title: t('当前计费'),
      dataIndex: 'current',
      render: (text) => <div style={{ whiteSpace: 'pre-wrap' }}>{text}</div>,
    },
    {
      title: t('修改为'),
      dataIndex: 'newVal',
      render: (text) => <div style={{ whiteSpace: 'pre-wrap' }}>{text}</div>,
    },
  ];

  return (
    <Modal
      title={t('确认冲突项修改')}
      visible={visible}
      onCancel={onCancel}
      onOk={onOk}
      size={isMobile ? 'full-width' : 'large'}
    >
      <Table
        columns={columns}
        dataSource={items}
        pagination={false}
        size='small'
      />
    </Modal>
  );
}

export default function UpstreamRatioSync(props) {
  const { t } = useTranslation();
  const [modalVisible, setModalVisible] = useState(false);
  const [loading, setLoading] = useState(false);
  const [syncLoading, setSyncLoading] = useState(false);

  const [allChannels, setAllChannels] = useState([]);
  const [selectedChannelIds, setSelectedChannelIds] = useState([]);
  const [channelEndpoints, setChannelEndpoints] = useState({});

  const [differences, setDifferences] = useState({});
  const [resolutions, setResolutions] = useState({});
  const [hasSynced, setHasSynced] = useState(false);

  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [ratioTypeFilter, setRatioTypeFilter] = useState('');

  const [confirmVisible, setConfirmVisible] = useState(false);
  const [conflictItems, setConflictItems] = useState([]);

  const channelSelectorRef = React.useRef(null);

  useEffect(() => {
    setCurrentPage(1);
  }, [ratioTypeFilter, searchKeyword]);

  const getCurrentRatios = useCallback(
    () => ({
      ModelRatio: JSON.parse(props.options.ModelRatio || '{}'),
      CompletionRatio: JSON.parse(props.options.CompletionRatio || '{}'),
      CacheRatio: JSON.parse(props.options.CacheRatio || '{}'),
      ModelPrice: JSON.parse(props.options.ModelPrice || '{}'),
    }),
    [
      props.options.CacheRatio,
      props.options.CompletionRatio,
      props.options.ModelPrice,
      props.options.ModelRatio,
    ],
  );

  const getBillingCategory = useCallback((ratioType) => {
    return ratioType === 'model_price' ? 'price' : 'ratio';
  }, []);

  const fetchAllChannels = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/ratio_sync/channels');

      if (!res.data.success) {
        showError(res.data.message);
        return;
      }

      const channels = (res.data.data || []).map((channel) => ({
        key: channel.id,
        label: channel.name,
        value: channel.id,
        disabled: false,
        _originalData: channel,
      }));

      setAllChannels(channels);
      setChannelEndpoints((prev) => {
        const merged = { ...prev };

        channels.forEach((channel) => {
          const id = channel.key;
          const base = channel._originalData?.base_url || '';
          const name = channel.label || '';
          const channelType = channel._originalData?.type;
          const isOfficialRatioPreset =
            id === OFFICIAL_RATIO_PRESET_ID ||
            base === OFFICIAL_RATIO_PRESET_BASE_URL ||
            name === OFFICIAL_RATIO_PRESET_NAME;
          const isModelsDevPreset =
            id === MODELS_DEV_PRESET_ID ||
            base === MODELS_DEV_PRESET_BASE_URL ||
            name === MODELS_DEV_PRESET_NAME;
          const isOpenRouter = channelType === 20;

          if (merged[id]) {
            return;
          }

          if (isModelsDevPreset) {
            merged[id] = MODELS_DEV_PRESET_ENDPOINT;
          } else if (isOfficialRatioPreset) {
            merged[id] = OFFICIAL_RATIO_PRESET_ENDPOINT;
          } else if (isOpenRouter) {
            merged[id] = 'openrouter';
          } else {
            merged[id] = DEFAULT_ENDPOINT;
          }
        });

        return merged;
      });
    } catch (error) {
      showError(t('获取渠道失败：') + error.message);
    } finally {
      setLoading(false);
    }
  };

  const requestDifferencesFromChannels = useCallback(
    async (channelList) => {
      const upstreams = channelList.map((ch) => ({
        id: ch.id,
        name: ch.name,
        base_url: ch.base_url,
        endpoint: channelEndpoints[ch.id] || DEFAULT_ENDPOINT,
      }));

      const res = await API.post('/api/ratio_sync/fetch', {
        upstreams,
        timeout: 10,
      });

      if (!res.data.success) {
        throw new Error(res.data.message || t('后端请求失败'));
      }

      const { differences = {}, test_results = [] } = res.data.data;
      const errorResults = test_results.filter((item) => item.status === 'error');

      if (errorResults.length > 0) {
        showWarning(
          t('部分渠道测试失败：') +
            errorResults.map((item) => `${item.name}: ${item.error}`).join(', '),
        );
      }

      return differences;
    },
    [channelEndpoints, t],
  );

  const fetchRatiosFromChannels = useCallback(
    async (channelList) => {
      setSyncLoading(true);
      try {
        const sourceDifferences = await requestDifferencesFromChannels(channelList);
        setDifferences(sourceDifferences);
        setResolutions({});
        setHasSynced(true);

        if (Object.keys(sourceDifferences).length === 0) {
          showSuccess(t('未找到差异化倍率，无需同步'));
        }
      } catch (error) {
        showError(t('请求后端接口失败：') + error.message);
      } finally {
        setSyncLoading(false);
      }
    },
    [requestDifferencesFromChannels, t],
  );

  const confirmChannelSelection = () => {
    const selected = allChannels
      .filter((channel) => selectedChannelIds.includes(channel.value))
      .map((channel) => channel._originalData);

    if (selected.length === 0) {
      showWarning(t('请至少选择一个渠道'));
      return;
    }

    setModalVisible(false);
    fetchRatiosFromChannels(selected);
  };

  const selectValue = useCallback(
    (model, ratioType, value) => {
      const category = getBillingCategory(ratioType);

      setResolutions((prev) => {
        const nextModelRes = { ...(prev[model] || {}) };

        Object.keys(nextModelRes).forEach((existingType) => {
          if (getBillingCategory(existingType) !== category) {
            delete nextModelRes[existingType];
          }
        });

        nextModelRes[ratioType] = value;

        return {
          ...prev,
          [model]: nextModelRes,
        };
      });
    },
    [getBillingCategory],
  );

  const buildAutoResolutions = useCallback(
    (sourceDifferences) => {
      const nextResolutions = {};

      Object.entries(sourceDifferences).forEach(([model, ratioTypes]) => {
        Object.entries(ratioTypes).forEach(([ratioType, diff]) => {
          const upstreamEntries = Object.entries(diff.upstreams || {});
          const confidentEntry =
            upstreamEntries.find(([name, value]) => {
              if (value === null || value === undefined || value === 'same') {
                return false;
              }
              return diff.confidence?.[name] !== false;
            }) || [];
          const fallbackEntry =
            upstreamEntries.find(
              ([, value]) =>
                value !== null && value !== undefined && value !== 'same',
            ) || [];
          const selectedValue =
            confidentEntry.length > 0 ? confidentEntry[1] : fallbackEntry[1];

          if (selectedValue === null || selectedValue === undefined) {
            return;
          }

          const category = getBillingCategory(ratioType);
          const nextModelRes = { ...(nextResolutions[model] || {}) };

          Object.keys(nextModelRes).forEach((existingType) => {
            if (getBillingCategory(existingType) !== category) {
              delete nextModelRes[existingType];
            }
          });

          nextModelRes[ratioType] = selectedValue;
          nextResolutions[model] = nextModelRes;
        });
      });

      return nextResolutions;
    },
    [getBillingCategory],
  );

  const performSync = useCallback(
    async (currentRatios, targetResolutions = resolutions) => {
      const finalRatios = {
        ModelRatio: { ...currentRatios.ModelRatio },
        CompletionRatio: { ...currentRatios.CompletionRatio },
        CacheRatio: { ...currentRatios.CacheRatio },
        ModelPrice: { ...currentRatios.ModelPrice },
      };

      Object.entries(targetResolutions).forEach(([model, ratios]) => {
        const selectedTypes = Object.keys(ratios);
        const hasPrice = selectedTypes.includes('model_price');
        const hasRatio = selectedTypes.some((ratioType) => ratioType !== 'model_price');

        if (hasPrice) {
          delete finalRatios.ModelRatio[model];
          delete finalRatios.CompletionRatio[model];
          delete finalRatios.CacheRatio[model];
        }
        if (hasRatio) {
          delete finalRatios.ModelPrice[model];
        }

        Object.entries(ratios).forEach(([ratioType, value]) => {
          const optionKey = ratioType
            .split('_')
            .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
            .join('');
          finalRatios[optionKey][model] = parseFloat(value);
        });
      });

      setLoading(true);
      try {
        const updates = Object.entries(finalRatios).map(([key, value]) =>
          API.put('/api/option/', {
            key,
            value: JSON.stringify(value, null, 2),
          }),
        );

        const results = await Promise.all(updates);

        if (!results.every((res) => res.data.success)) {
          showError(t('部分保存失败'));
          return;
        }

        showSuccess(t('同步成功'));
        props.refresh();

        setDifferences((prevDifferences) => {
          const nextDifferences = { ...prevDifferences };

          Object.entries(targetResolutions).forEach(([model, ratios]) => {
            Object.keys(ratios).forEach((ratioType) => {
              if (nextDifferences[model] && nextDifferences[model][ratioType]) {
                delete nextDifferences[model][ratioType];
                if (Object.keys(nextDifferences[model]).length === 0) {
                  delete nextDifferences[model];
                }
              }
            });
          });

          return nextDifferences;
        });

        setResolutions({});
      } catch (error) {
        showError(t('保存失败'));
      } finally {
        setLoading(false);
      }
    },
    [props.refresh, resolutions, t],
  );

  const applySync = useCallback(
    async (targetResolutions = resolutions, sourceDifferences = differences) => {
      const currentRatios = getCurrentRatios();
      const conflicts = [];

      const getLocalBillingCategory = (model) => {
        if (currentRatios.ModelPrice[model] !== undefined) {
          return 'price';
        }
        if (
          currentRatios.ModelRatio[model] !== undefined ||
          currentRatios.CompletionRatio[model] !== undefined ||
          currentRatios.CacheRatio[model] !== undefined
        ) {
          return 'ratio';
        }
        return null;
      };

      const findSourceChannel = (model, ratioType, value) => {
        if (sourceDifferences[model] && sourceDifferences[model][ratioType]) {
          const upstreamMap = sourceDifferences[model][ratioType].upstreams || {};
          const entry = Object.entries(upstreamMap).find(([, current]) => {
            return current === value;
          });
          if (entry) {
            return entry[0];
          }
        }
        return t('未知');
      };

      Object.entries(targetResolutions).forEach(([model, ratios]) => {
        const localCategory = getLocalBillingCategory(model);
        const nextCategory = 'model_price' in ratios ? 'price' : 'ratio';

        if (!localCategory || localCategory === nextCategory) {
          return;
        }

        const currentDesc =
          localCategory === 'price'
            ? `${t('固定价格')} : ${currentRatios.ModelPrice[model]}`
            : `${t('模型倍率')} : ${currentRatios.ModelRatio[model] ?? '-'}\n${t('补全倍率')} : ${currentRatios.CompletionRatio[model] ?? '-'}`;

        let nextDesc = '';
        if (nextCategory === 'price') {
          nextDesc = `${t('固定价格')} : ${ratios.model_price}`;
        } else {
          nextDesc = `${t('模型倍率')} : ${ratios.model_ratio ?? '-'}\n${t('补全倍率')} : ${ratios.completion_ratio ?? '-'}`;
        }

        const channels = Object.entries(ratios)
          .map(([ratioType, value]) => findSourceChannel(model, ratioType, value))
          .filter((name, index, list) => list.indexOf(name) === index)
          .join(', ');

        conflicts.push({
          channel: channels,
          model,
          current: currentDesc,
          newVal: nextDesc,
        });
      });

      setResolutions(targetResolutions);

      if (conflicts.length > 0) {
        setConflictItems(conflicts);
        setConfirmVisible(true);
        return;
      }

      await performSync(currentRatios, targetResolutions);
    },
    [differences, getCurrentRatios, performSync, resolutions, t],
  );

  const quickSyncPreset = useCallback(
    async (presetType) => {
      const presetChannel =
        presetType === 'official'
          ? {
              id: OFFICIAL_RATIO_PRESET_ID,
              name: OFFICIAL_RATIO_PRESET_NAME,
              base_url: OFFICIAL_RATIO_PRESET_BASE_URL,
            }
          : {
              id: MODELS_DEV_PRESET_ID,
              name: MODELS_DEV_PRESET_NAME,
              base_url: MODELS_DEV_PRESET_BASE_URL,
            };

      setSyncLoading(true);
      try {
        const sourceDifferences = await requestDifferencesFromChannels([
          presetChannel,
        ]);
        const autoResolutions = buildAutoResolutions(sourceDifferences);

        setDifferences(sourceDifferences);
        setHasSynced(true);

        if (Object.keys(autoResolutions).length === 0) {
          setResolutions({});
          showSuccess(t('未找到差异化倍率，无需同步'));
          return;
        }

        await applySync(autoResolutions, sourceDifferences);
      } catch (error) {
        showError(t('请求后端接口失败：') + error.message);
      } finally {
        setSyncLoading(false);
      }
    },
    [applySync, buildAutoResolutions, requestDifferencesFromChannels, t],
  );

  const dataSource = useMemo(() => {
    const rows = [];

    Object.entries(differences).forEach(([model, ratioTypes]) => {
      const hasPrice = 'model_price' in ratioTypes;
      const hasOtherRatio = ['model_ratio', 'completion_ratio', 'cache_ratio'].some(
        (ratioType) => ratioType in ratioTypes,
      );
      const billingConflict = hasPrice && hasOtherRatio;

      Object.entries(ratioTypes).forEach(([ratioType, diff]) => {
        rows.push({
          key: `${model}_${ratioType}`,
          model,
          ratioType,
          current: diff.current,
          upstreams: diff.upstreams,
          confidence: diff.confidence || {},
          billingConflict,
        });
      });
    });

    return rows;
  }, [differences]);

  const filteredDataSource = useMemo(() => {
    if (!searchKeyword.trim() && !ratioTypeFilter) {
      return dataSource;
    }

    return dataSource.filter((item) => {
      const matchesKeyword =
        !searchKeyword.trim() ||
        item.model.toLowerCase().includes(searchKeyword.toLowerCase().trim());
      const matchesRatioType =
        !ratioTypeFilter || item.ratioType === ratioTypeFilter;
      return matchesKeyword && matchesRatioType;
    });
  }, [dataSource, ratioTypeFilter, searchKeyword]);

  const upstreamNames = useMemo(() => {
    const set = new Set();
    filteredDataSource.forEach((row) => {
      Object.keys(row.upstreams || {}).forEach((name) => set.add(name));
    });
    return Array.from(set);
  }, [filteredDataSource]);

  const getCurrentPageData = useCallback(
    (rows) => {
      const startIndex = (currentPage - 1) * pageSize;
      const endIndex = startIndex + pageSize;
      return rows.slice(startIndex, endIndex);
    },
    [currentPage, pageSize],
  );

  const columns = useMemo(() => {
    return [
      {
        title: t('模型'),
        dataIndex: 'model',
        fixed: 'left',
      },
      {
        title: t('倍率类型'),
        dataIndex: 'ratioType',
        render: (text, record) => {
          const typeMap = {
            model_ratio: t('模型倍率'),
            completion_ratio: t('补全倍率'),
            cache_ratio: t('缓存倍率'),
            model_price: t('固定价格'),
          };
          const baseTag = (
            <Tag color={stringToColor(text)} shape='circle'>
              {typeMap[text] || text}
            </Tag>
          );

          if (!record?.billingConflict) {
            return baseTag;
          }

          return (
            <div className='flex items-center gap-1'>
              {baseTag}
              <Tooltip
                position='top'
                content={t(
                  '该模型同时存在固定价格与倍率计费方式冲突，请确认选择',
                )}
              >
                <AlertTriangle size={14} className='text-yellow-500' />
              </Tooltip>
            </div>
          );
        },
      },
      {
        title: t('置信度'),
        dataIndex: 'confidence',
        render: (_, record) => {
          const allConfident = Object.values(record.confidence || {}).every(
            (value) => value !== false,
          );

          if (allConfident) {
            return (
              <Tooltip content={t('所有上游数据均可信')}>
                <Tag
                  color='green'
                  shape='circle'
                  type='light'
                  prefixIcon={<CheckCircle size={14} />}
                >
                  {t('可信')}
                </Tag>
              </Tooltip>
            );
          }

          const untrustedSources = Object.entries(record.confidence || {})
            .filter(([, confident]) => confident === false)
            .map(([name]) => name)
            .join(', ');

          return (
            <Tooltip content={t('以下上游数据可能不可信：') + untrustedSources}>
              <Tag
                color='yellow'
                shape='circle'
                type='light'
                prefixIcon={<AlertTriangle size={14} />}
              >
                {t('谨慎')}
              </Tag>
            </Tooltip>
          );
        },
      },
      {
        title: t('当前值'),
        dataIndex: 'current',
        render: (text) => (
          <Tag
            color={text !== null && text !== undefined ? 'blue' : 'default'}
            shape='circle'
          >
            {text !== null && text !== undefined ? String(text) : t('未设置')}
          </Tag>
        ),
      },
      ...upstreamNames.map((upstreamName) => {
        const channelStats = (() => {
          let selectableCount = 0;
          let selectedCount = 0;

          filteredDataSource.forEach((row) => {
            const upstreamValue = row.upstreams?.[upstreamName];
            if (
              upstreamValue !== null &&
              upstreamValue !== undefined &&
              upstreamValue !== 'same'
            ) {
              selectableCount++;
              if (resolutions[row.model]?.[row.ratioType] === upstreamValue) {
                selectedCount++;
              }
            }
          });

          return {
            selectableCount,
            selectedCount,
            allSelected:
              selectableCount > 0 && selectedCount === selectableCount,
            partiallySelected:
              selectedCount > 0 && selectedCount < selectableCount,
            hasSelectableItems: selectableCount > 0,
          };
        })();

        const handleBulkSelect = (checked) => {
          if (checked) {
            filteredDataSource.forEach((row) => {
              const upstreamValue = row.upstreams?.[upstreamName];
              if (
                upstreamValue !== null &&
                upstreamValue !== undefined &&
                upstreamValue !== 'same'
              ) {
                selectValue(row.model, row.ratioType, upstreamValue);
              }
            });
            return;
          }

          setResolutions((prev) => {
            const nextResolutions = { ...prev };
            filteredDataSource.forEach((row) => {
              if (nextResolutions[row.model]) {
                delete nextResolutions[row.model][row.ratioType];
                if (Object.keys(nextResolutions[row.model]).length === 0) {
                  delete nextResolutions[row.model];
                }
              }
            });
            return nextResolutions;
          });
        };

        return {
          title: channelStats.hasSelectableItems ? (
            <Checkbox
              checked={channelStats.allSelected}
              indeterminate={channelStats.partiallySelected}
              onChange={(event) => handleBulkSelect(event.target.checked)}
            >
              {upstreamName}
            </Checkbox>
          ) : (
            <span>{upstreamName}</span>
          ),
          dataIndex: upstreamName,
          render: (_, record) => {
            const upstreamValue = record.upstreams?.[upstreamName];
            const isConfident = record.confidence?.[upstreamName] !== false;

            if (upstreamValue === null || upstreamValue === undefined) {
              return (
                <Tag color='default' shape='circle'>
                  {t('未设置')}
                </Tag>
              );
            }

            if (upstreamValue === 'same') {
              return (
                <Tag color='blue' shape='circle'>
                  {t('与本地相同')}
                </Tag>
              );
            }

            const isSelected =
              resolutions[record.model]?.[record.ratioType] === upstreamValue;

            return (
              <div className='flex items-center gap-2'>
                <Checkbox
                  checked={isSelected}
                  onChange={(event) => {
                    const checked = event.target.checked;
                    if (checked) {
                      selectValue(record.model, record.ratioType, upstreamValue);
                      return;
                    }

                    setResolutions((prev) => {
                      const nextResolutions = { ...prev };
                      if (nextResolutions[record.model]) {
                        delete nextResolutions[record.model][record.ratioType];
                        if (Object.keys(nextResolutions[record.model]).length === 0) {
                          delete nextResolutions[record.model];
                        }
                      }
                      return nextResolutions;
                    });
                  }}
                >
                  {String(upstreamValue)}
                </Checkbox>
                {!isConfident && (
                  <Tooltip position='left' content={t('该数据可能不可信，请谨慎使用')}>
                    <AlertTriangle size={16} className='text-yellow-500' />
                  </Tooltip>
                )}
              </div>
            );
          },
        };
      }),
    ];
  }, [filteredDataSource, resolutions, selectValue, t, upstreamNames]);

  const handleModalClose = () => {
    setModalVisible(false);
    if (channelSelectorRef.current) {
      channelSelectorRef.current.resetPagination();
    }
  };

  const renderHeader = (
    <div className='flex flex-col w-full gap-3'>
      <div className='flex flex-col md:flex-row justify-between items-center gap-4 w-full'>
        <div className='flex flex-col md:flex-row gap-2 w-full md:w-auto order-2 md:order-1'>
          <Button
            icon={<RefreshCcw size={14} />}
            className='w-full md:w-auto mt-2'
            onClick={() => {
              setModalVisible(true);
              if (allChannels.length === 0) {
                fetchAllChannels();
              }
            }}
          >
            {t('选择同步渠道')}
          </Button>

          <Button
            type='tertiary'
            className='w-full md:w-auto mt-2'
            loading={syncLoading}
            disabled={loading}
            onClick={() => quickSyncPreset('official')}
          >
            {t('一键同步官方计费预设')}
          </Button>

          <Button
            type='tertiary'
            className='w-full md:w-auto mt-2'
            loading={syncLoading}
            disabled={loading}
            onClick={() => quickSyncPreset('models_dev')}
          >
            {t('一键同步 models.dev 价格预设')}
          </Button>

          <Button
            icon={<CheckSquare size={14} />}
            type='secondary'
            onClick={() => applySync()}
            disabled={Object.keys(resolutions).length === 0}
            className='w-full md:w-auto mt-2'
          >
            {t('应用同步')}
          </Button>
        </div>
      </div>

      <div className='flex flex-col sm:flex-row gap-2 w-full md:w-auto'>
        <Input
          prefix={<IconSearch size={14} />}
          placeholder={t('搜索模型名称')}
          value={searchKeyword}
          onChange={setSearchKeyword}
          className='w-full sm:w-64'
          showClear
        />

        <Select
          placeholder={t('按倍率类型筛选')}
          value={ratioTypeFilter}
          onChange={setRatioTypeFilter}
          className='w-full sm:w-48'
          showClear
          onClear={() => setRatioTypeFilter('')}
        >
          <Select.Option value='model_ratio'>{t('模型倍率')}</Select.Option>
          <Select.Option value='completion_ratio'>
            {t('补全倍率')}
          </Select.Option>
          <Select.Option value='cache_ratio'>{t('缓存倍率')}</Select.Option>
          <Select.Option value='model_price'>{t('固定价格')}</Select.Option>
        </Select>
      </div>
    </div>
  );

  const renderDifferenceTable = () => {
    if (filteredDataSource.length === 0) {
      return (
        <Empty
          image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
          darkModeImage={
            <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
          }
          description={
            searchKeyword.trim()
              ? t('未找到匹配的模型')
              : Object.keys(differences).length === 0
                ? hasSynced
                  ? t('暂无差异化倍率显示')
                  : t('请先选择同步渠道')
                : t('请先选择同步渠道')
          }
          style={{ padding: 30 }}
        />
      );
    }

    return (
      <Table
        columns={columns}
        dataSource={getCurrentPageData(filteredDataSource)}
        pagination={{
          currentPage,
          pageSize,
          total: filteredDataSource.length,
          showSizeChanger: true,
          showQuickJumper: true,
          pageSizeOptions: ['5', '10', '20', '50'],
          onChange: (page, size) => {
            setCurrentPage(page);
            setPageSize(size);
          },
          onShowSizeChange: (_, size) => {
            setCurrentPage(1);
            setPageSize(size);
          },
        }}
        scroll={{ x: 'max-content' }}
        size='middle'
        loading={loading || syncLoading}
      />
    );
  };

  const updateChannelEndpoint = useCallback((channelId, endpoint) => {
    setChannelEndpoints((prev) => ({ ...prev, [channelId]: endpoint }));
  }, []);

  return (
    <>
      <Form.Section text={renderHeader}>{renderDifferenceTable()}</Form.Section>

      <ChannelSelectorModal
        ref={channelSelectorRef}
        t={t}
        visible={modalVisible}
        onCancel={handleModalClose}
        onOk={confirmChannelSelection}
        allChannels={allChannels}
        selectedChannelIds={selectedChannelIds}
        setSelectedChannelIds={setSelectedChannelIds}
        channelEndpoints={channelEndpoints}
        updateChannelEndpoint={updateChannelEndpoint}
      />

      <ConflictConfirmModal
        t={t}
        visible={confirmVisible}
        items={conflictItems}
        onOk={async () => {
          setConfirmVisible(false);
          await performSync(getCurrentRatios());
        }}
        onCancel={() => setConfirmVisible(false)}
      />
    </>
  );
}
