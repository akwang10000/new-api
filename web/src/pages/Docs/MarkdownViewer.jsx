import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Button, Empty, Spin } from '@douyinfe/semi-ui';
import { IconArrowLeft, IconDownload } from '@douyinfe/semi-icons';
import { useSearchParams } from 'react-router-dom';
import MarkdownRenderer from '../../components/common/markdown/MarkdownRenderer';
import {
  getReadableDocTitle,
  isMarkdownDocPath,
  isOpenApiDocPath,
  normalizeLocalDocPath,
  resolveDocLink,
  toMarkdownViewerHref,
  toOpenApiViewerHref,
} from './utils';
import './docs-viewer.css';

const DEFAULT_TITLE = 'Documentation';
const DEFAULT_SUBTITLE =
  'The local document is rendered below for direct reading in the browser.';

function rewriteRelativeDocumentLinks(container, filePath) {
  if (!container || !filePath) {
    return;
  }

  container.querySelectorAll('a[href]').forEach((anchor) => {
    const href = anchor.getAttribute('href');
    const resolved = resolveDocLink(filePath, href);

    if (!resolved || resolved.startsWith('#')) {
      return;
    }

    const resolvedUrl = new URL(resolved, window.location.origin);
    if (isMarkdownDocPath(resolvedUrl.toString())) {
      anchor.setAttribute(
        'href',
        toMarkdownViewerHref(
          `${resolvedUrl.pathname}${resolvedUrl.search}${resolvedUrl.hash}`,
          getReadableDocTitle(resolvedUrl.toString()),
        ),
      );
      return;
    }

    if (isOpenApiDocPath(resolvedUrl.toString())) {
      anchor.setAttribute(
        'href',
        toOpenApiViewerHref(
          `${resolvedUrl.pathname}${resolvedUrl.search}${resolvedUrl.hash}`,
          getReadableDocTitle(resolvedUrl.toString()),
        ),
      );
      return;
    }

    anchor.setAttribute('href', resolvedUrl.toString());
  });

  container
    .querySelectorAll('img[src], source[src], video[src], audio[src]')
    .forEach((node) => {
      const source = node.getAttribute('src');
      const resolved = resolveDocLink(filePath, source);
      if (!resolved || resolved.startsWith('#')) {
        return;
      }
      node.setAttribute('src', resolved);
    });
}

export default function MarkdownViewer() {
  const [searchParams] = useSearchParams();
  const [content, setContent] = useState('');
  const [loadError, setLoadError] = useState('');
  const [loading, setLoading] = useState(true);
  const contentRef = useRef(null);

  const title = searchParams.get('title') || DEFAULT_TITLE;
  const subtitle = searchParams.get('subtitle') || DEFAULT_SUBTITLE;
  const requestedFile = searchParams.get('file');

  const filePath = useMemo(
    () => normalizeLocalDocPath(requestedFile),
    [requestedFile],
  );

  useEffect(() => {
    document.title = `${title} | Nerve Centers`;
  }, [title]);

  useEffect(() => {
    let cancelled = false;

    if (!filePath) {
      setContent('');
      setLoadError('Missing or invalid file parameter.');
      setLoading(false);
      return undefined;
    }

    setLoading(true);
    setLoadError('');
    setContent('');

    fetch(filePath)
      .then(async (response) => {
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}`);
        }
        return response.text();
      })
      .then((payload) => {
        if (cancelled) {
          return;
        }
        setContent(payload);
      })
      .catch((error) => {
        if (cancelled) {
          return;
        }
        setLoadError(error.message || 'Failed to load document.');
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [filePath]);

  useEffect(() => {
    if (!content || !contentRef.current || !filePath) {
      return;
    }

    rewriteRelativeDocumentLinks(contentRef.current, filePath);
  }, [content, filePath]);

  return (
    <div className='docs-viewer'>
      <main className='docs-viewer__shell'>
        <section className='docs-viewer__hero'>
          <span className='docs-viewer__eyebrow'>Markdown Viewer</span>
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
            {filePath && (
              <a
                className='docs-viewer__action-link'
                href={filePath}
                target='_blank'
                rel='noreferrer'
              >
                <Button icon={<IconDownload />}>Open Raw File</Button>
              </a>
            )}
          </div>
        </section>

        <section className='docs-viewer__panel docs-viewer__panel--markdown'>
          {loading && (
            <div className='docs-viewer__loading'>
              <div className='docs-viewer__loading-body'>
                <Spin spinning size='large' />
                <span>Loading document...</span>
              </div>
            </div>
          )}

          {!loading && loadError && (
            <div className='docs-viewer__error'>
              <Empty
                description={`Unable to load this document: ${loadError}`}
              />
            </div>
          )}

          {!loading && !loadError && content && (
            <div ref={contentRef} className='docs-viewer__markdown'>
              <MarkdownRenderer content={content} />
            </div>
          )}
        </section>
      </main>
    </div>
  );
}
