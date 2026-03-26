(function () {
  var STORAGE_KEY = "docs-lang";
  var SITE_STORAGE_KEY = "i18nextLng";
  var SUPPORTED = ["zh-CN", "en"];

  function normalizeLang(value) {
    if (!value) return null;
    var lower = String(value).toLowerCase();
    if (lower === "zh" || lower === "zh-cn" || lower === "zh-hans") {
      return "zh-CN";
    }
    if (
      lower === "zh-tw" ||
      lower === "zh-hk" ||
      lower === "zh-mo" ||
      lower === "zh-hant"
    ) {
      return "zh-CN";
    }
    if (lower === "en" || lower === "en-us" || lower === "en-gb") {
      return "en";
    }
    return null;
  }

  function pickLang() {
    var url = new URL(window.location.href);
    var fromQuery = normalizeLang(url.searchParams.get("lang"));
    if (fromQuery) return fromQuery;

    var fromStorage = normalizeLang(window.localStorage.getItem(STORAGE_KEY));
    if (fromStorage) return fromStorage;

    var fromSiteStorage = normalizeLang(
      window.localStorage.getItem(SITE_STORAGE_KEY),
    );
    if (fromSiteStorage) return fromSiteStorage;

    var fromBrowser = normalizeLang(navigator.language || navigator.userLanguage);
    return fromBrowser || "zh-CN";
  }

  function setQueryLang(lang) {
    var url = new URL(window.location.href);
    url.searchParams.set("lang", lang);
    window.history.replaceState({}, "", url.toString());
  }

  function withLang(href, lang) {
    if (!href) return href;
    if (/^(mailto:|tel:|javascript:|#)/i.test(href)) return href;

    try {
      var url = new URL(href, window.location.origin);
      if (url.origin !== window.location.origin) return href;
      url.searchParams.set("lang", lang);
      return url.pathname + url.search + url.hash;
    } catch (error) {
      return href;
    }
  }

  function applyMap(selector, key, dict, setter) {
    var nodes = document.querySelectorAll(selector);
    nodes.forEach(function (node) {
      var token = node.getAttribute(key);
      if (!token || dict[token] == null) return;
      setter(node, dict[token]);
    });
  }

  function applyLanguage(lang) {
    var maps = window.DOCS_I18N || {};
    var dict = maps[lang] || maps["zh-CN"] || {};
    document.documentElement.lang = lang;

    applyMap("[data-i18n]", "data-i18n", dict, function (node, value) {
      node.textContent = value;
    });

    applyMap("[data-i18n-html]", "data-i18n-html", dict, function (node, value) {
      node.innerHTML = value;
    });

    var titleKey = document.documentElement.getAttribute("data-i18n-title");
    if (titleKey && dict[titleKey]) {
      document.title = dict[titleKey];
    }

    document.querySelectorAll("[data-lang-option]").forEach(function (node) {
      var active = node.getAttribute("data-lang-option") === lang;
      node.classList.toggle("is-active", active);
      node.setAttribute("aria-pressed", active ? "true" : "false");
    });

    document.querySelectorAll("a[data-keep-lang]").forEach(function (node) {
      var rawHref = node.getAttribute("data-href") || node.getAttribute("href");
      if (!node.getAttribute("data-href")) {
        node.setAttribute("data-href", rawHref);
      }
      node.setAttribute("href", withLang(rawHref, lang));
    });

    window.localStorage.setItem(STORAGE_KEY, lang);
    setQueryLang(lang);
  }

  function bindSwitcher() {
    document.querySelectorAll("[data-lang-option]").forEach(function (node) {
      node.addEventListener("click", function () {
        var lang = normalizeLang(node.getAttribute("data-lang-option"));
        if (!lang) return;
        applyLanguage(lang);
      });
    });
  }

  function init() {
    var lang = pickLang();
    if (SUPPORTED.indexOf(lang) === -1) {
      lang = "zh-CN";
    }
    bindSwitcher();
    applyLanguage(lang);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
