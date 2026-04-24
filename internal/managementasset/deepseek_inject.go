package managementasset

import (
	"strings"
)

const deepseekProviderInjection = `<script id="deepseek-provider-injection">
(function () {
  var MANAGEMENT_BASE = "/v0/management";
  var ROOT_ID = "provider-deepseek";
  var STYLE_ID = "deepseek-provider-style";
  var DEFAULT_BASE_URL = "https://api.deepseek.com/anthropic";
  var _dsAuthToken = "";
  function _dsTryStorage() {
    try { var k = localStorage.getItem("managementKey"); if (k) _dsAuthToken = "Bearer " + k; } catch(e) {}
    try { for (var i = 0; i < localStorage.length; i++) { var sk = localStorage.key(i); if (sk && sk.indexOf("enc-") === 0 && sk.indexOf("managementKey") !== -1) { var v = localStorage.getItem(sk); if (v) { try { v = atob(v); } catch(e2) {} if (v) _dsAuthToken = "Bearer " + v; } } } } catch(e) {}
  }
  _dsTryStorage();
  var _origXHROpen = XMLHttpRequest.prototype.open;
  var _origXHRSetHdr = XMLHttpRequest.prototype.setRequestHeader;
  XMLHttpRequest.prototype.open = function () { this._dsReqArgs = arguments; return _origXHROpen.apply(this, arguments); };
  XMLHttpRequest.prototype.setRequestHeader = function (n, v) {
    if (n && n.toLowerCase() === "authorization" && v && v.indexOf("Bearer ") === 0) _dsAuthToken = v;
    return _origXHRSetHdr.apply(this, arguments);
  };
  var _origFetch = window.fetch;
  window.fetch = function () {
    var a = arguments;
    if (a.length >= 2 && a[1] && a[1].headers) {
      try {
        var h = a[1].headers;
        if (typeof h === "object" && !Array.isArray(h)) {
          var auth = h.Authorization || h.authorization;
          if (auth && auth.indexOf("Bearer ") === 0) _dsAuthToken = auth;
        }
      } catch (e) {}
    }
    return _origFetch.apply(this, a);
  };
  var state = { configs: [], staticModels: [], loading: false, status: "", error: false, loaded: false };

  var DEEPSEEK_SVG = '<svg xmlns="http://www.w3.org/2000/svg" fill="none" height="1em" width="1em" viewBox="3.771 6.973 23.993 17.652" style="flex:none;line-height:1"><title>DeepSeek</title><path d="m27.501 8.469c-.252-.123-.36.111-.508.23-.05.04-.093.09-.135.135-.368.395-.797.652-1.358.621-.821-.045-1.521.213-2.14.842-.132-.776-.57-1.238-1.235-1.535-.349-.155-.701-.309-.944-.645-.171-.238-.217-.504-.303-.765-.054-.159-.108-.32-.29-.348-.197-.031-.274.135-.352.273-.31.567-.43 1.192-.419 1.825.028 1.421.628 2.554 1.82 3.36.136.093.17.186.128.321-.081.278-.178.547-.264.824-.054.178-.135.217-.324.14a5.448 5.448 0 0 1 -1.719-1.169c-.848-.82-1.614-1.726-2.57-2.435-.225-.166-.449-.32-.681-.467-.976-.95.128-1.729.383-1.82.267-.096.093-.428-.77-.424s-1.653.293-2.659.677a2.782 2.782 0 0 1 -.46.135 9.554 9.554 0 0 0 -2.853-.1c-1.866.21-3.356 1.092-4.452 2.6-1.315 1.81-1.625 3.87-1.246 6.018.399 2.261 1.552 4.136 3.326 5.601 1.837 1.518 3.955 2.262 6.37 2.12 1.466-.085 3.1-.282 4.942-1.842.465.23.952.322 1.762.392.623.059 1.223-.031 1.687-.127.728-.154.677-.828.414-.953-2.132-.994-1.665-.59-2.09-.916 1.084-1.285 2.717-2.619 3.356-6.94.05-.343.007-.558 0-.837-.004-.168.034-.235.228-.254a4.084 4.084 0 0 0 1.529-.47c1.382-.757 1.938-1.997 2.07-3.485.02-.227-.004-.463-.243-.582zm-12.041 13.391c-2.067-1.627-3.07-2.162-3.483-2.138-.387.021-.318.465-.233.754.089.285.205.482.368.732.113.166.19.414-.112.598-.666.414-1.823-.139-1.878-.166-1.347-.793-2.473-1.842-3.267-3.276-.765-1.38-1.21-2.861-1.284-4.441-.02-.383.093-.518.472-.586a4.692 4.692 0 0 1 1.514-.04c2.109.31 3.905 1.255 5.41 2.749.86.853 1.51 1.871 2.18 2.865.711 1.057 1.478 2.063 2.454 2.887.343.289.619.51.881.672-.792.088-2.117.107-3.022-.61zm.99-6.38a.304.304 0 1 1 .609 0c0 .17-.136.304-.306.304a.3.3 0 0 1 -.303-.305zm3.077 1.581c-.197.08-.394.15-.584.159a1.246 1.246 0 0 1 -.79-.252c-.27-.227-.463-.354-.546-.752a1.752 1.752 0 0 1 .016-.582c.07-.324-.008-.531-.235-.72-.187-.155-.422-.196-.682-.196a.551.551 0 0 1 -.252-.078c-.108-.055-.197-.19-.112-.356.027-.053.159-.183.19-.207.352-.201.758-.135 1.134.016.349.142.611.404.99.773.388.448.457.573.678.906.174.264.333.534.441.842.066.192-.02.35-.248.448z" fill="#4d6bfe"/></svg>';
  var DEEPSEEK_ICON_DATAURI = "data:image/svg+xml;utf8," + encodeURIComponent(DEEPSEEK_SVG);

  function request(path, options) {
    var opts = options || {};
    var headers = opts.headers || {};
    if (_dsAuthToken) headers["Authorization"] = _dsAuthToken;
    if (!headers["Content-Type"] && opts.body) headers["Content-Type"] = "application/json";
    return _origFetch(MANAGEMENT_BASE + path, {
      method: opts.method || "GET",
      headers: headers,
      body: opts.body
    }).then(function (r) {
      return r.text().then(function (t) {
        var d = t ? JSON.parse(t) : {};
        if (!r.ok) throw new Error((d && d.error) ? d.error : t || r.statusText || "Request failed");
        return d;
      });
    });
  }

  function esc(v) { return String(v == null ? "" : v).replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;"); }
  function mask(v) { var r = String(v||"").trim(); return !r ? "-" : r.length <= 10 ? r : r.slice(0,6) + "..." + r.slice(-4); }

  function normalizeConfig(item) {
    item = item && typeof item === "object" ? item : {};
    return {
      apiKey: String(item["api-key"] || item.apiKey || "").trim(),
      priority: item.priority == null || item.priority === "" ? "" : String(item.priority).trim(),
      prefix: String(item.prefix || "").trim(),
      baseUrl: String(item["base-url"] || item.baseUrl || "").trim(),
      proxyUrl: String(item["proxy-url"] || item.proxyUrl || "").trim(),
      models: Array.isArray(item.models) ? item.models : [],
      headers: item.headers && typeof item.headers === "object" && !Array.isArray(item.headers) ? item.headers : {},
      excludedModels: Array.isArray(item["excluded-models"]) ? item["excluded-models"] : []
    };
  }

  function toPayload(c) {
    var p = { "api-key": c.apiKey };
    if (c.priority !== "") p.priority = Number(c.priority);
    if (c.prefix) p.prefix = c.prefix;
    if (c.baseUrl) p["base-url"] = c.baseUrl;
    if (c.proxyUrl) p["proxy-url"] = c.proxyUrl;
    var h = c.headers || {};
    if (Object.keys(h).length) p.headers = h;
    if (c.models.length) p.models = c.models;
    if (c.excludedModels.length) p["excluded-models"] = c.excludedModels;
    return p;
  }

  var DS_CSS = [
    "#" + ROOT_ID + " { margin-top: 16px; }",
    ".ds-card { border: 1px solid var(--border-color, rgba(148,163,184,.35)); border-radius: 18px; padding: 18px; background: var(--card-bg, rgba(255,255,255,.96)); box-shadow: 0 18px 50px rgba(15,23,42,.08); }",
    ".ds-head { display: flex; justify-content: space-between; gap: 12px; flex-wrap: wrap; align-items: flex-start; margin-bottom: 14px; }",
    ".ds-title-row { display: flex; align-items: center; gap: 8px; }",
    ".ds-title-icon { width: 22px; height: 22px; flex: none; display: inline-block; }",
    ".ds-title { margin: 0; font-size: 20px; font-weight: 700; color: var(--text-primary, #0f172a); }",
    ".ds-sub { margin: 4px 0 0; color: var(--text-secondary, #475569); font-size: 13px; line-height: 1.5; }",
    ".ds-actions { display: flex; gap: 8px; flex-wrap: wrap; }",
    ".ds-btn { border: 0; border-radius: 999px; padding: 8px 14px; font-size: 13px; font-weight: 600; cursor: pointer; transition: opacity .15s; }",
    ".ds-btn:hover { opacity: .85; }",
    ".ds-btn.primary { background: #4D6BFE; color: #fff; }",
    ".ds-btn.secondary { background: var(--btn-secondary-bg, rgba(15,23,42,.07)); color: var(--text-primary, #0f172a); }",
    ".ds-btn.danger { background: rgba(220,38,38,.09); color: #b91c1c; }",
    ".ds-btn:disabled { opacity: .5; cursor: not-allowed; }",
    ".ds-status { margin: 0 0 12px; padding: 10px 12px; border-radius: 12px; font-size: 13px; line-height: 1.5; background: rgba(77,107,254,.08); color: #3451d1; }",
    ".ds-status.error { background: rgba(220,38,38,.09); color: #b91c1c; }",
    ".ds-meta { display: flex; gap: 8px 12px; flex-wrap: wrap; margin-bottom: 14px; }",
    ".ds-pill { display: inline-flex; gap: 6px; align-items: center; padding: 5px 10px; border-radius: 999px; background: rgba(77,107,254,.08); color: #3451d1; font-size: 12px; font-weight: 600; }",
    ".ds-empty { padding: 18px; border: 1px dashed rgba(148,163,184,.5); border-radius: 14px; color: #64748b; text-align: center; }",
    ".ds-list { display: grid; gap: 12px; }",
    ".ds-item { border: 1px solid rgba(148,163,184,.25); border-radius: 16px; padding: 16px; background: var(--item-bg, rgba(248,250,252,.8)); }",
    ".ds-item-head, .ds-item-foot { display: flex; justify-content: space-between; gap: 12px; flex-wrap: wrap; align-items: center; }",
    ".ds-item-head { margin-bottom: 10px; }",
    ".ds-item-title { font-size: 15px; font-weight: 700; }",
    ".ds-badges { display: flex; gap: 6px; flex-wrap: wrap; }",
    ".ds-badge { display: inline-flex; padding: 4px 8px; border-radius: 999px; background: rgba(148,163,184,.15); color: #334155; font-size: 11px; font-weight: 700; }",
    ".ds-badge.warn { background: rgba(245,158,11,.14); color: #b45309; }",
    ".ds-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px,1fr)); gap: 10px; margin-bottom: 10px; }",
    ".ds-field { display: flex; flex-direction: column; gap: 3px; }",
    ".ds-label { font-size: 11px; font-weight: 700; letter-spacing: .04em; text-transform: uppercase; color: #64748b; }",
    ".ds-value { font-size: 13px; line-height: 1.4; word-break: break-all; }",
    ".ds-note { font-size: 12px; color: #64748b; }"
  ].join("\n");

  function ensureStyle() {
    if (document.getElementById(STYLE_ID)) return;
    var s = document.createElement("style");
    s.id = STYLE_ID;
    s.textContent = DS_CSS;
    document.head.appendChild(s);
  }

  function ensureRoot() {
    var anchor = document.getElementById("provider-zhipu") || document.getElementById("provider-openai") || document.getElementById("provider-vertex");
    if (!anchor || !anchor.parentNode) return null;
    var root = document.getElementById(ROOT_ID);
    if (!root) {
      root = document.createElement("div");
      root.id = ROOT_ID;
      anchor.insertAdjacentElement("afterend", root);
    }
    if (!root._dsBound) {
      root.addEventListener("click", onClick);
      root._dsBound = true;
    }
    return root;
  }

  function render() {
    var root = ensureRoot();
    if (!root) return;
    ensureStyle();
    var statusHtml = state.status ? '<div class="ds-status' + (state.error ? ' error' : '') + '">' + esc(state.status) + '</div>' : '';
    var body;
    if (state.loading && !state.configs.length) {
      body = '<div class="ds-empty">正在加载 DeepSeek 提供商配置...</div>';
    } else if (!state.configs.length) {
      body = '<div class="ds-empty">暂无 DeepSeek API 密钥，点击上方"添加 DeepSeek 密钥"按钮添加。</div>';
    } else {
      body = '<div class="ds-list">' + state.configs.map(function (c, i) {
        var badges = '<span class="ds-badge">模型 ' + c.models.length + '</span>';
        if (c.excludedModels.length) badges += '<span class="ds-badge warn">已排除 ' + c.excludedModels.length + '</span>';
        return '<article class="ds-item">' +
          '<div class="ds-item-head"><div class="ds-item-title">DeepSeek 配置 #' + (i + 1) + '</div><div class="ds-badges">' + badges + '</div></div>' +
          '<div class="ds-grid">' +
          '<div class="ds-field"><span class="ds-label">API 密钥</span><span class="ds-value"><code>' + esc(mask(c.apiKey)) + '</code></span></div>' +
          '<div class="ds-field"><span class="ds-label">Base URL</span><span class="ds-value">' + esc(c.baseUrl || DEFAULT_BASE_URL) + '</span></div>' +
          '<div class="ds-field"><span class="ds-label">前缀</span><span class="ds-value">' + esc(c.prefix || "-") + '</span></div>' +
          '<div class="ds-field"><span class="ds-label">代理地址</span><span class="ds-value">' + esc(c.proxyUrl || "-") + '</span></div>' +
          '</div>' +
          '<div class="ds-item-foot"><div class="ds-note">' + esc(c.models.map(function(m){ return m.alias || m.name; }).slice(0,5).join("、") || "未配置模型") + '</div>' +
          '<div class="ds-actions"><button type="button" class="ds-btn secondary" data-action="edit" data-index="' + i + '">编辑</button>' +
          '<button type="button" class="ds-btn danger" data-action="delete" data-index="' + i + '">删除</button></div></div></article>';
      }).join('') + '</div>';
    }
    root.innerHTML = '<section class="ds-card">' +
      '<div class="ds-head"><div><div class="ds-title-row"><img class="ds-title-icon" alt="DeepSeek" src="' + DEEPSEEK_ICON_DATAURI + '"/><h2 class="ds-title">DeepSeek</h2></div>' +
      '<p class="ds-sub">DeepSeek AI 提供商，使用兼容 Anthropic 协议的端点（deepseek-v4-flash、deepseek-v4-pro）。</p></div>' +
      '<div class="ds-actions"><button type="button" class="ds-btn secondary" data-action="refresh"' + (state.loading ? ' disabled' : '') + '>刷新</button>' +
      '<button type="button" class="ds-btn primary" data-action="add"' + (state.loading ? ' disabled' : '') + '>添加 DeepSeek 密钥</button></div></div>' +
      statusHtml +
      '<div class="ds-meta"><span class="ds-pill">密钥 <strong>' + state.configs.length + '</strong></span>' +
      '<span class="ds-pill">模型 <strong>' + state.staticModels.length + '</strong></span>' +
      '<span class="ds-pill">端点 <strong>' + esc(DEFAULT_BASE_URL) + '</strong></span></div>' +
      body + '</section>';
  }

  function saveAll(configs, msg) {
    state.loading = true; state.status = "保存中..."; state.error = false; render();
    return request("/deepseek-api-key", { method: "PUT", body: JSON.stringify(configs.map(toPayload)) })
      .then(function () { return load(true).then(function(){ state.status = msg; state.error = false; render(); }); })
      .catch(function (e) { state.status = "保存失败：" + (e && e.message ? e.message : e); state.error = true; render(); })
      .finally(function () { state.loading = false; render(); });
  }

  function editConfig(index) {
    var cur = index >= 0 && state.configs[index] ? state.configs[index] : { apiKey: "", priority: "", prefix: "", baseUrl: "", proxyUrl: "", headers: {}, models: [], excludedModels: [] };
    var apiKey = prompt("DeepSeek API 密钥", cur.apiKey || "");
    if (apiKey === null) return;
    apiKey = apiKey.trim();
    if (!apiKey) { alert("API 密钥不能为空。"); return; }
    var prefix = prompt("前缀（可选，留空表示无）", cur.prefix || "");
    if (prefix === null) return;
    var baseUrl = prompt("Base URL（留空使用默认）", cur.baseUrl || "");
    if (baseUrl === null) return;
    var modelsRaw = prompt("模型：上游名称=>别名，多个使用分号分隔", cur.models.map(function(m){ return m.name + "=>" + m.alias; }).join("; "));
    if (modelsRaw === null) return;
    var models = [];
    String(modelsRaw || "").split(/[;\r\n]+/).forEach(function(p) {
      p = p.trim(); if (!p) return;
      var name, alias;
      if (p.indexOf("=>") !== -1) { name = p.split("=>")[0].trim(); alias = p.split("=>")[1].trim(); }
      else { name = p; alias = p; }
      if (name) models.push({ name: name, alias: alias || name });
    });
    var next = state.configs.slice();
    next[index >= 0 ? index : next.length] = { apiKey: apiKey, priority: cur.priority, prefix: prefix.trim(), baseUrl: baseUrl.trim(), proxyUrl: cur.proxyUrl, headers: cur.headers, models: models, excludedModels: cur.excludedModels };
    saveAll(next, index >= 0 ? "配置已更新。" : "配置已添加。");
  }

  function deleteConfig(index) {
    var c = state.configs[index];
    if (!c || !confirm("确认删除 DeepSeek 配置 #" + (index+1) + "？")) return;
    state.loading = true; state.status = "删除中..."; state.error = false; render();
    var q = "?api-key=" + encodeURIComponent(c.apiKey);
    if (c.baseUrl) q += "&base-url=" + encodeURIComponent(c.baseUrl);
    request("/deepseek-api-key" + q, { method: "DELETE" })
      .then(function () { return load(true).then(function(){ state.status = "已删除。"; state.error = false; render(); }); })
      .catch(function (e) { state.status = "删除失败：" + (e && e.message ? e.message : e); state.error = true; render(); })
      .finally(function () { state.loading = false; render(); });
  }

  function onClick(e) {
    var btn = e.target.closest("[data-action]");
    if (!btn) return;
    var idx = btn.hasAttribute("data-index") ? Number(btn.getAttribute("data-index")) : -1;
    var act = btn.getAttribute("data-action");
    if (act === "refresh") { load(false); return; }
    if (act === "add") { editConfig(-1); return; }
    if (act === "edit") { editConfig(idx); return; }
    if (act === "delete") { deleteConfig(idx); }
  }

  function load(silent) {
    if (!ensureRoot()) return Promise.resolve();
    state.loading = true;
    if (!silent) { state.status = "加载中..."; state.error = false; }
    render();
    return Promise.all([
      request("/deepseek-api-key").catch(function(){ return { "deepseek-api-key": [] }; }),
      request("/model-definitions/deepseek").catch(function(){ return { models: [] }; })
    ]).then(function(r) {
      state.configs = Array.isArray(r[0] && r[0]["deepseek-api-key"]) ? r[0]["deepseek-api-key"].map(normalizeConfig) : [];
      state.staticModels = Array.isArray(r[1] && r[1].models) ? r[1].models : [];
      if (!silent) { state.status = ""; state.error = false; }
    }).catch(function(e) {
      state.status = "加载失败：" + (e && e.message ? e.message : e); state.error = true;
    }).finally(function() { state.loading = false; state.loaded = true; render(); });
  }

  function ensureNavButton() {
    var list = document.querySelector('[class*="ProviderNav-module__navList"]');
    if (!list) return;
    if (list.querySelector('[data-ds-nav]')) return;
    var template = list.querySelector('button[title]');
    if (!template) return;
    var btn = document.createElement("button");
    btn.className = template.className;
    btn.type = "button";
    btn.title = "DeepSeek";
    btn.setAttribute("aria-label", "DeepSeek");
    btn.setAttribute("aria-pressed", "false");
    btn.setAttribute("data-ds-nav", "1");
    var img = document.createElement("img");
    img.alt = "DeepSeek";
    var tImg = template.querySelector("img");
    if (tImg) img.className = tImg.className;
    img.src = DEEPSEEK_ICON_DATAURI;
    btn.appendChild(img);
    list.appendChild(btn);
  }

  function installNavPatch() {
    var list = document.querySelector('[class*="ProviderNav-module__navList"]');
    if (!list || list._dsNavPatch) return;
    var indicator = list.querySelector('[class*="ProviderNav-module__indicator"]');
    if (!indicator) return;
    list._dsNavPatch = true;
    var lockedBtn = null;
    var lockUntil = 0;
    var applying = false;

    function applyIndicatorTo(btn) {
      if (!btn || !indicator) return;
      applying = true;
      try {
        indicator.style.transform = "translate3d(" + btn.offsetLeft + "px, " + btn.offsetTop + "px, 0px)";
        indicator.style.width = btn.offsetWidth + "px";
        indicator.style.height = btn.offsetHeight + "px";
        var btns = list.querySelectorAll("button[title]");
        for (var i = 0; i < btns.length; i++) {
          btns[i].setAttribute("aria-pressed", btns[i] === btn ? "true" : "false");
        }
      } finally {
        window.setTimeout(function () { applying = false; }, 30);
      }
    }

    list.addEventListener("click", function (e) {
      var btn = e.target.closest ? e.target.closest("button[title]") : null;
      if (!btn || btn.parentElement !== list) return;
      lockedBtn = btn;
      lockUntil = Date.now() + 900;
      applyIndicatorTo(btn);
      window.setTimeout(function () { applyIndicatorTo(btn); }, 120);
      window.setTimeout(function () { applyIndicatorTo(btn); }, 360);
      if (btn.getAttribute("data-ds-nav") === "1") {
        var dsRoot = document.getElementById(ROOT_ID);
        if (dsRoot && dsRoot.scrollIntoView) {
          dsRoot.scrollIntoView({ behavior: "smooth", block: "start" });
        }
      }
    }, false);

    if (typeof MutationObserver !== "undefined") {
      new MutationObserver(function () {
        if (applying) return;
        if (lockedBtn && Date.now() < lockUntil) {
          applyIndicatorTo(lockedBtn);
        }
      }).observe(indicator, { attributes: true, attributeFilter: ["style", "class"] });
    }
  }

  function mount() {
    var root = ensureRoot();
    ensureNavButton();
    installNavPatch();
    if (!root) return;
    if (!root._dsRendered) { render(); root._dsRendered = true; }
    if (!state.loading && !state.loaded) load(true);
  }

  ensureStyle();
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", mount);
  } else {
    mount();
  }
  window.setInterval(function() { mount(); }, 2000);
  window.setInterval(function() { if (ensureRoot()) load(true); }, 30000);
  if (typeof MutationObserver !== "undefined") {
    new MutationObserver(mount).observe(document.documentElement, { childList: true, subtree: true });
  }
})();
</script>`

// InjectDeepseekProvider inserts the DeepSeek provider UI script into the management
// HTML page immediately before the closing </body> tag.
func InjectDeepseekProvider(html string) string {
	closingBody := "</body>"
	if idx := strings.LastIndex(strings.ToLower(html), closingBody); idx >= 0 {
		return html[:idx] + deepseekProviderInjection + html[idx:]
	}
	return html + deepseekProviderInjection
}
