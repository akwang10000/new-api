import React, { useEffect, useMemo, useState } from 'react';
import { Button, Empty, Spin } from '@douyinfe/semi-ui';
import { IconArrowLeft, IconDownload } from '@douyinfe/semi-icons';
import { useSearchParams } from 'react-router-dom';
import { RedocStandalone } from 'redoc';
import { normalizeLocalDocPath, withRawParam } from './utils';
import './docs-viewer.css';

const DEFAULT_TITLE = 'API Documentation';
const DEFAULT_SUBTITLE =
  'The structured API reference is rendered below for direct reading in the browser.';

const redocOptions = {
  expandResponses: '200,201',
  hideDownloadButton: false,
  disableSearch: false,
  theme: {
    colors: {
      primary: {
        main: '#d96f32',
      },
      text: {
        primary: '#203036',
        secondary: '#5d6e73',
      },
    },
    typography: {
      fontFamily: '"Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif',
      headings: {
        fontFamily: '"Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif',
      },
    },
  },
};

export default function OpenApiViewer() {
  const [searchParams] = useSearchParams();
  const [specData, setSpecData] = useState(null);
  const [loadError, setLoadError] = useState('');
  const [loading, setLoading] = useState(true);

  const title = searchParams.get('title') || DEFAULT_TITLE;
  const subtitle = searchParams.get('subtitle') || DEFAULT_SUBTITLE;
  const requestedSpec = searchParams.get('spec');

  const specPath = useMemo(
    () => normalizeLocalDocPath(requestedSpec),
    [requestedSpec],
  );
  const rawSpecPath = useMemo(
    () => (specPath ? withRawParam(specPath) : null),
    [specPath],
  );

  useEffect(() => {
    document.title = `${title} | Nerve Centers`;
  }, [title]);

  useEffect(() => {
    let cancelled = false;

    if (!rawSpecPath) {
      setSpecData(null);
      setLoadError('Missing or invalid spec parameter.');
      setLoading(false);
      return undefined;
    }

    setLoading(true);
    setLoadError('');
    setSpecData(null);

    fetch(rawSpecPath, {
      headers: {
        Accept: 'application/json',
      },
    })
      .then(async (response) => {
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}`);
        }
        return response.json();
      })
      .then((payload) => {
        if (cancelled) {
          return;
        }
        setSpecData(payload);
      })
      .catch((error) => {
        if (cancelled) {
          return;
        }
        setLoadError(error.message || 'Failed to load specification.');
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [rawSpecPath]);

  return (
    <div className='docs-viewer'>
      <main className='docs-viewer__shell'>
        <section className='docs-viewer__hero'>
          <span className='docs-viewer__eyebrow'>OpenAPI Viewer</span>
          <h1 className='docs-viewer__title'>{title}</h1>
          <p className='docs-viewer__subtitle'>{subtitle}</p>
          <div className='docs-viewer__actions'>
            <a
              className='docs-viewer__action-link'
              href='/docs-home.html'
              target='_blank'
              rel='noreferrer'
            >
              <Button icon={<IconArrowLeft />}>Back to Docs Home</Button>
            </a>
            {rawSpecPath && (
              <a
                className='docs-viewer__action-link'
                href={rawSpecPath}
                target='_blank'
                rel='noreferrer'
              >
                <Button icon={<IconDownload />}>Open Raw JSON</Button>
              </a>
            )}
          </div>
        </section>

        <section className='docs-viewer__panel docs-viewer__panel--openapi'>
          {loading && (
            <div className='docs-viewer__loading'>
              <div className='docs-viewer__loading-body'>
                <Spin spinning size='large' />
                <span>Loading specification...</span>
              </div>
            </div>
          )}

          {!loading && loadError && (
            <div className='docs-viewer__error'>
              <Empty
                description={`Unable to load this specification: ${loadError}`}
              />
            </div>
          )}

          {!loading && !loadError && specData && (
            <RedocStandalone spec={specData} options={redocOptions} />
          )}
        </section>
      </main>
    </div>
  );
}
