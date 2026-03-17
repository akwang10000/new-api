const DOCS_PREFIX = '/docs/';

const ABSOLUTE_URL_PATTERN = /^(?:[a-z][a-z\d+\-.]*:|\/\/)/i;

export function normalizeLocalDocPath(value) {
  if (!value) {
    return null;
  }

  try {
    const url = new URL(value, window.location.origin);
    if (url.origin !== window.location.origin) {
      return null;
    }
    if (!url.pathname.startsWith(DOCS_PREFIX)) {
      return null;
    }
    return `${url.pathname}${url.search}${url.hash}`;
  } catch {
    return null;
  }
}

export function withRawParam(path) {
  const url = new URL(path, window.location.origin);
  url.searchParams.set('raw', '1');
  return `${url.pathname}${url.search}${url.hash}`;
}

export function getReadableDocTitle(path, fallback) {
  if (fallback) {
    return fallback;
  }

  try {
    const url = new URL(path, window.location.origin);
    const segments = url.pathname.split('/').filter(Boolean);
    const fileName = segments[segments.length - 1] || 'document';
    const baseName = fileName.replace(/\.[^/.]+$/, '');
    return baseName
      .split(/[-_]+/)
      .filter(Boolean)
      .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
      .join(' ');
  } catch {
    return fallback || 'Documentation';
  }
}

export function toMarkdownViewerHref(path, title) {
  const url = new URL(path, window.location.origin);
  const filePath = `${url.pathname}${url.search}`;
  const resolvedTitle = getReadableDocTitle(path, title);
  return `/docs-viewer/markdown?file=${encodeURIComponent(filePath)}&title=${encodeURIComponent(resolvedTitle)}${url.hash}`;
}

export function toOpenApiViewerHref(path, title) {
  const url = new URL(path, window.location.origin);
  const specPath = `${url.pathname}${url.search}`;
  const resolvedTitle = getReadableDocTitle(path, title);
  return `/docs-viewer/openapi?spec=${encodeURIComponent(specPath)}&title=${encodeURIComponent(resolvedTitle)}${url.hash}`;
}

export function resolveDocLink(basePath, value) {
  if (!value) {
    return null;
  }

  const trimmedValue = value.trim();
  if (!trimmedValue || trimmedValue.startsWith('#')) {
    return trimmedValue;
  }

  try {
    const baseUrl = new URL(basePath, window.location.origin);
    return new URL(trimmedValue, baseUrl).toString();
  } catch {
    if (ABSOLUTE_URL_PATTERN.test(trimmedValue)) {
      return trimmedValue;
    }
    return null;
  }
}

export function isMarkdownDocPath(path) {
  try {
    const url = new URL(path, window.location.origin);
    return url.pathname.startsWith(DOCS_PREFIX) && url.pathname.endsWith('.md');
  } catch {
    return false;
  }
}

export function isOpenApiDocPath(path) {
  try {
    const url = new URL(path, window.location.origin);
    return (
      url.pathname.startsWith('/docs/openapi/') &&
      url.pathname.endsWith('.json')
    );
  } catch {
    return false;
  }
}
