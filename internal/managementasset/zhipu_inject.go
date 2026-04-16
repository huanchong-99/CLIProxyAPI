package managementasset

import (
	"strings"
)

const zhipuProviderInjection = `<script id="zhipu-provider-injection">
(function () {
  var MANAGEMENT_BASE = "/v0/management";
  var ROOT_ID = "provider-zhipu";
  var STYLE_ID = "zhipu-provider-style";
  var DEFAULT_BASE_URL = "https://open.bigmodel.cn/api/anthropic";
  var _zpAuthToken = "";
  function _zpTryStorage() {
    try { var k = localStorage.getItem("managementKey"); if (k) _zpAuthToken = "Bearer " + k; } catch(e) {}
    try { for (var i = 0; i < localStorage.length; i++) { var sk = localStorage.key(i); if (sk && sk.indexOf("enc-") === 0 && sk.indexOf("managementKey") !== -1) { var v = localStorage.getItem(sk); if (v) { try { v = atob(v); } catch(e2) {} if (v) _zpAuthToken = "Bearer " + v; } } } } catch(e) {}
  }
  _zpTryStorage();
  var _origXHROpen = XMLHttpRequest.prototype.open;
  var _origXHRSetHdr = XMLHttpRequest.prototype.setRequestHeader;
  XMLHttpRequest.prototype.open = function () { this._zpReqArgs = arguments; return _origXHROpen.apply(this, arguments); };
  XMLHttpRequest.prototype.setRequestHeader = function (n, v) {
    if (n && n.toLowerCase() === "authorization" && v && v.indexOf("Bearer ") === 0) _zpAuthToken = v;
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
          if (auth && auth.indexOf("Bearer ") === 0) _zpAuthToken = auth;
        }
      } catch (e) {}
    }
    return _origFetch.apply(this, a);
  };
  var state = { configs: [], staticModels: [], loading: false, status: "", error: false, loaded: false };

  var ZHIPU_SVG = '<svg xmlns="http://www.w3.org/2000/svg" fill="#3859FF" fill-rule="evenodd" height="1em" width="1em" viewBox="0 0 24 24" style="flex:none;line-height:1"><title>Zhipu</title><path d="M11.991 23.503a.24.24 0 00-.244.248.24.24 0 00.244.249.24.24 0 00.245-.249.24.24 0 00-.22-.247l-.025-.001zM9.671 5.365a1.697 1.697 0 011.099 2.132l-.071.172-.016.04-.018.054c-.07.16-.104.32-.104.498-.035.71.47 1.279 1.186 1.314h.366c1.309.053 2.338 1.173 2.286 2.523-.052 1.332-1.152 2.38-2.478 2.327h-.174c-.715.018-1.274.64-1.239 1.368 0 .124.018.23.053.337.209.373.54.658.96.8.75.23 1.517-.125 1.9-.782l.018-.035c.402-.64 1.17-.96 1.92-.711.854.284 1.378 1.226 1.099 2.167a1.661 1.661 0 01-2.077 1.102 1.711 1.711 0 01-.907-.711l-.017-.035c-.2-.323-.463-.58-.851-.711l-.056-.018a1.646 1.646 0 00-1.954.746 1.66 1.66 0 01-1.065.764 1.677 1.677 0 01-1.989-1.279c-.209-.906.332-1.83 1.257-2.043a1.51 1.51 0 01.296-.035h.018c.68-.071 1.151-.622 1.116-1.333a1.307 1.307 0 00-.227-.693 2.515 2.515 0 01-.366-1.403 2.39 2.39 0 01.366-1.208c.14-.195.21-.444.227-.693.018-.71-.506-1.261-1.186-1.332l-.07-.018a1.43 1.43 0 01-.299-.07l-.05-.019a1.7 1.7 0 01-1.047-2.114 1.68 1.68 0 012.094-1.101zm-5.575 10.11c.26-.264.639-.367.994-.27.355.096.633.379.728.74.095.362-.007.748-.267 1.013-.402.41-1.053.41-1.455 0a1.062 1.062 0 010-1.482zm14.845-.294c.359-.09.738.024.992.297.254.274.344.665.237 1.025-.107.36-.396.634-.756.718-.551.128-1.1-.22-1.23-.781a1.05 1.05 0 01.757-1.26zm-.064-4.39c.314.32.49.753.49 1.206 0 .452-.176.886-.49 1.206-.315.32-.74.5-1.185.5-.444 0-.87-.18-1.184-.5a1.727 1.727 0 010-2.412 1.654 1.654 0 012.369 0zm-11.243.163c.364.484.447 1.128.218 1.691a1.665 1.665 0 01-2.188.923c-.855-.36-1.26-1.358-.907-2.228a1.68 1.68 0 011.33-1.038c.593-.08 1.183.169 1.547.652zm11.545-4.221c.368 0 .708.2.892.524.184.324.184.724 0 1.048a1.026 1.026 0 01-.892.524c-.568 0-1.03-.47-1.03-1.048 0-.579.462-1.048 1.03-1.048zm-14.358 0c.368 0 .707.2.891.524.184.324.184.724 0 1.048a1.026 1.026 0 01-.891.524c-.569 0-1.03-.47-1.03-1.048 0-.579.461-1.048 1.03-1.048zm10.031-1.475c.925 0 1.675.764 1.675 1.706s-.75 1.705-1.675 1.705-1.674-.763-1.674-1.705c0-.942.75-1.706 1.674-1.706zm-2.626-.684c.362-.082.653-.356.761-.718a1.062 1.062 0 00-.238-1.028 1.017 1.017 0 00-.996-.294c-.547.14-.881.7-.752 1.257.13.558.675.907 1.225.783zm0 16.876c.359-.087.644-.36.75-.72a1.062 1.062 0 00-.237-1.019 1.018 1.018 0 00-.985-.301 1.037 1.037 0 00-.762.717c-.108.361-.017.754.239 1.028.245.263.606.377.953.305l.043-.01zM17.19 3.5a.631.631 0 00.628-.64c0-.355-.279-.64-.628-.64a.631.631 0 00-.628.64c0 .355.28.64.628.64zm-10.38 0a.631.631 0 00.628-.64c0-.355-.28-.64-.628-.64a.631.631 0 00-.628.64c0 .355.279.64.628.64zm-5.182 7.852a.631.631 0 00-.628.64c0 .354.28.639.628.639a.63.63 0 00.627-.606l.001-.034a.62.62 0 00-.628-.64zm5.182 9.13a.631.631 0 00-.628.64c0 .355.279.64.628.64a.631.631 0 00.628-.64c0-.355-.28-.64-.628-.64zm10.38.018a.631.631 0 00-.628.64c0 .355.28.64.628.64a.631.631 0 00.628-.64c0-.355-.279-.64-.628-.64zm5.182-9.148a.631.631 0 00-.628.64c0 .354.279.639.628.639a.631.631 0 00.628-.64c0-.355-.28-.64-.628-.64zm-.384-4.992a.24.24 0 00.244-.249.24.24 0 00-.244-.249.24.24 0 00-.244.249c0 .142.122.249.244.249zM11.991.497a.24.24 0 00.245-.248A.24.24 0 0011.99 0a.24.24 0 00-.244.249c0 .133.108.236.223.247l.021.001zM2.011 6.36a.24.24 0 00.245-.249.24.24 0 00-.244-.249.24.24 0 00-.244.249.24.24 0 00.244.249zm0 11.263a.24.24 0 00-.243.248.24.24 0 00.244.249.24.24 0 00.244-.249.252.252 0 00-.244-.248zm19.995-.018a.24.24 0 00-.245.248.24.24 0 00.245.25.24.24 0 00.244-.25.252.252 0 00-.244-.248z"/></svg>';
  var ZHIPU_ICON_DATAURI = "data:image/svg+xml;utf8," + encodeURIComponent(ZHIPU_SVG);

  function request(path, options) {
    var opts = options || {};
    var headers = opts.headers || {};
    if (_zpAuthToken) headers["Authorization"] = _zpAuthToken;
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

  var ZHIPU_CSS = [
    "#" + ROOT_ID + " { margin-top: 16px; }",
    ".zp-card { border: 1px solid var(--border-color, rgba(148,163,184,.35)); border-radius: 18px; padding: 18px; background: var(--card-bg, rgba(255,255,255,.96)); box-shadow: 0 18px 50px rgba(15,23,42,.08); }",
    ".zp-head { display: flex; justify-content: space-between; gap: 12px; flex-wrap: wrap; align-items: flex-start; margin-bottom: 14px; }",
    ".zp-title-row { display: flex; align-items: center; gap: 8px; }",
    ".zp-title-icon { width: 22px; height: 22px; flex: none; display: inline-block; }",
    ".zp-title { margin: 0; font-size: 20px; font-weight: 700; color: var(--text-primary, #0f172a); }",
    ".zp-sub { margin: 4px 0 0; color: var(--text-secondary, #475569); font-size: 13px; line-height: 1.5; }",
    ".zp-actions { display: flex; gap: 8px; flex-wrap: wrap; }",
    ".zp-btn { border: 0; border-radius: 999px; padding: 8px 14px; font-size: 13px; font-weight: 600; cursor: pointer; transition: opacity .15s; }",
    ".zp-btn:hover { opacity: .85; }",
    ".zp-btn.primary { background: #2563eb; color: #fff; }",
    ".zp-btn.secondary { background: var(--btn-secondary-bg, rgba(15,23,42,.07)); color: var(--text-primary, #0f172a); }",
    ".zp-btn.danger { background: rgba(220,38,38,.09); color: #b91c1c; }",
    ".zp-btn:disabled { opacity: .5; cursor: not-allowed; }",
    ".zp-status { margin: 0 0 12px; padding: 10px 12px; border-radius: 12px; font-size: 13px; line-height: 1.5; background: rgba(37,99,235,.08); color: #1d4ed8; }",
    ".zp-status.error { background: rgba(220,38,38,.09); color: #b91c1c; }",
    ".zp-meta { display: flex; gap: 8px 12px; flex-wrap: wrap; margin-bottom: 14px; }",
    ".zp-pill { display: inline-flex; gap: 6px; align-items: center; padding: 5px 10px; border-radius: 999px; background: rgba(37,99,235,.08); color: #1d4ed8; font-size: 12px; font-weight: 600; }",
    ".zp-empty { padding: 18px; border: 1px dashed rgba(148,163,184,.5); border-radius: 14px; color: #64748b; text-align: center; }",
    ".zp-list { display: grid; gap: 12px; }",
    ".zp-item { border: 1px solid rgba(148,163,184,.25); border-radius: 16px; padding: 16px; background: var(--item-bg, rgba(248,250,252,.8)); }",
    ".zp-item-head, .zp-item-foot { display: flex; justify-content: space-between; gap: 12px; flex-wrap: wrap; align-items: center; }",
    ".zp-item-head { margin-bottom: 10px; }",
    ".zp-item-title { font-size: 15px; font-weight: 700; }",
    ".zp-badges { display: flex; gap: 6px; flex-wrap: wrap; }",
    ".zp-badge { display: inline-flex; padding: 4px 8px; border-radius: 999px; background: rgba(148,163,184,.15); color: #334155; font-size: 11px; font-weight: 700; }",
    ".zp-badge.warn { background: rgba(245,158,11,.14); color: #b45309; }",
    ".zp-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px,1fr)); gap: 10px; margin-bottom: 10px; }",
    ".zp-field { display: flex; flex-direction: column; gap: 3px; }",
    ".zp-label { font-size: 11px; font-weight: 700; letter-spacing: .04em; text-transform: uppercase; color: #64748b; }",
    ".zp-value { font-size: 13px; line-height: 1.4; word-break: break-all; }",
    ".zp-note { font-size: 12px; color: #64748b; }"
  ].join("\n");

  function ensureStyle() {
    if (document.getElementById(STYLE_ID)) return;
    var s = document.createElement("style");
    s.id = STYLE_ID;
    s.textContent = ZHIPU_CSS;
    document.head.appendChild(s);
  }

  function ensureRoot() {
    var anchor = document.getElementById("provider-openai") || document.getElementById("provider-vertex");
    if (!anchor || !anchor.parentNode) return null;
    var root = document.getElementById(ROOT_ID);
    if (!root) {
      root = document.createElement("div");
      root.id = ROOT_ID;
      anchor.insertAdjacentElement("afterend", root);
    }
    if (!root._zpBound) {
      root.addEventListener("click", onClick);
      root._zpBound = true;
    }
    return root;
  }

  function render() {
    var root = ensureRoot();
    if (!root) return;
    ensureStyle();
    var statusHtml = state.status ? '<div class="zp-status' + (state.error ? ' error' : '') + '">' + esc(state.status) + '</div>' : '';
    var body;
    if (state.loading && !state.configs.length) {
      body = '<div class="zp-empty">正在加载智谱提供商配置...</div>';
    } else if (!state.configs.length) {
      body = '<div class="zp-empty">暂无智谱 API 密钥，点击上方“添加智谱密钥”按钮添加第一个 GLM 密钥。</div>';
    } else {
      body = '<div class="zp-list">' + state.configs.map(function (c, i) {
        var badges = '<span class="zp-badge">模型 ' + c.models.length + '</span>';
        if (c.excludedModels.length) badges += '<span class="zp-badge warn">已排除 ' + c.excludedModels.length + '</span>';
        return '<article class="zp-item">' +
          '<div class="zp-item-head"><div class="zp-item-title">智谱配置 #' + (i + 1) + '</div><div class="zp-badges">' + badges + '</div></div>' +
          '<div class="zp-grid">' +
          '<div class="zp-field"><span class="zp-label">API 密钥</span><span class="zp-value"><code>' + esc(mask(c.apiKey)) + '</code></span></div>' +
          '<div class="zp-field"><span class="zp-label">Base URL</span><span class="zp-value">' + esc(c.baseUrl || DEFAULT_BASE_URL) + '</span></div>' +
          '<div class="zp-field"><span class="zp-label">前缀</span><span class="zp-value">' + esc(c.prefix || "-") + '</span></div>' +
          '<div class="zp-field"><span class="zp-label">代理地址</span><span class="zp-value">' + esc(c.proxyUrl || "-") + '</span></div>' +
          '</div>' +
          '<div class="zp-item-foot"><div class="zp-note">' + esc(c.models.map(function(m){ return m.alias || m.name; }).slice(0,5).join("、") || "未配置模型") + '</div>' +
          '<div class="zp-actions"><button type="button" class="zp-btn secondary" data-action="edit" data-index="' + i + '">编辑</button>' +
          '<button type="button" class="zp-btn danger" data-action="delete" data-index="' + i + '">删除</button></div></div></article>';
      }).join('') + '</div>';
    }
    root.innerHTML = '<section class="zp-card">' +
      '<div class="zp-head"><div><div class="zp-title-row"><img class="zp-title-icon" alt="Zhipu" src="' + ZHIPU_ICON_DATAURI + '"/><h2 class="zp-title">Zhipu / GLM</h2></div>' +
      '<p class="zp-sub">智谱 AI 提供商，使用兼容 Anthropic 协议的 BigModel 端点（glm-5、glm-5.1 等）。</p></div>' +
      '<div class="zp-actions"><button type="button" class="zp-btn secondary" data-action="refresh"' + (state.loading ? ' disabled' : '') + '>刷新</button>' +
      '<button type="button" class="zp-btn primary" data-action="add"' + (state.loading ? ' disabled' : '') + '>添加智谱密钥</button></div></div>' +
      statusHtml +
      '<div class="zp-meta"><span class="zp-pill">密钥 <strong>' + state.configs.length + '</strong></span>' +
      '<span class="zp-pill">模型 <strong>' + state.staticModels.length + '</strong></span>' +
      '<span class="zp-pill">端点 <strong>' + esc(DEFAULT_BASE_URL) + '</strong></span></div>' +
      body + '</section>';
  }

  function saveAll(configs, msg) {
    state.loading = true; state.status = "保存中..."; state.error = false; render();
    return request("/zhipu-api-key", { method: "PUT", body: JSON.stringify(configs.map(toPayload)) })
      .then(function () { return load(true).then(function(){ state.status = msg; state.error = false; render(); }); })
      .catch(function (e) { state.status = "保存失败：" + (e && e.message ? e.message : e); state.error = true; render(); })
      .finally(function () { state.loading = false; render(); });
  }

  function editConfig(index) {
    var cur = index >= 0 && state.configs[index] ? state.configs[index] : { apiKey: "", priority: "", prefix: "", baseUrl: "", proxyUrl: "", headers: {}, models: [], excludedModels: [] };
    var apiKey = prompt("智谱 API 密钥", cur.apiKey || "");
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
    if (!c || !confirm("确认删除智谱配置 #" + (index+1) + "？")) return;
    state.loading = true; state.status = "删除中..."; state.error = false; render();
    var q = "?api-key=" + encodeURIComponent(c.apiKey);
    if (c.baseUrl) q += "&base-url=" + encodeURIComponent(c.baseUrl);
    request("/zhipu-api-key" + q, { method: "DELETE" })
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
      request("/zhipu-api-key").catch(function(){ return { "zhipu-api-key": [] }; }),
      request("/model-definitions/zhipu").catch(function(){ return { models: [] }; })
    ]).then(function(r) {
      state.configs = Array.isArray(r[0] && r[0]["zhipu-api-key"]) ? r[0]["zhipu-api-key"].map(normalizeConfig) : [];
      state.staticModels = Array.isArray(r[1] && r[1].models) ? r[1].models : [];
      if (!silent) { state.status = ""; state.error = false; }
    }).catch(function(e) {
      state.status = "加载失败：" + (e && e.message ? e.message : e); state.error = true;
    }).finally(function() { state.loading = false; state.loaded = true; render(); });
  }

  function ensureNavButton() {
    var list = document.querySelector('[class*="ProviderNav-module__navList"]');
    if (!list) return;
    if (list.querySelector('[data-zp-nav]')) return;
    var template = list.querySelector('button[title]');
    if (!template) return;
    var btn = document.createElement("button");
    btn.className = template.className;
    btn.type = "button";
    btn.title = "Zhipu / GLM";
    btn.setAttribute("aria-label", "Zhipu / GLM");
    btn.setAttribute("aria-pressed", "false");
    btn.setAttribute("data-zp-nav", "1");
    var img = document.createElement("img");
    img.alt = "Zhipu";
    var tImg = template.querySelector("img");
    if (tImg) img.className = tImg.className;
    img.src = ZHIPU_ICON_DATAURI;
    btn.appendChild(img);
    list.appendChild(btn);
  }

  function installNavPatch() {
    var list = document.querySelector('[class*="ProviderNav-module__navList"]');
    if (!list || list._zpNavPatch) return;
    var indicator = list.querySelector('[class*="ProviderNav-module__indicator"]');
    if (!indicator) return;
    list._zpNavPatch = true;
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
      if (btn.getAttribute("data-zp-nav") === "1") {
        var zpRoot = document.getElementById(ROOT_ID);
        if (zpRoot && zpRoot.scrollIntoView) {
          zpRoot.scrollIntoView({ behavior: "smooth", block: "start" });
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
    if (!root._zpRendered) { render(); root._zpRendered = true; }
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

// InjectZhipuProvider inserts the Zhipu provider UI script into the management
// HTML page immediately before the closing </body> tag. This ensures the Zhipu
// provider section appears on the AI Providers page alongside Claude, Vertex,
// Codex, and OpenAI providers. The injection works for both local files and
// auto-downloaded management assets in Docker deployments.
func InjectZhipuProvider(html string) string {
	closingBody := "</body>"
	if idx := strings.LastIndex(strings.ToLower(html), closingBody); idx >= 0 {
		return html[:idx] + zhipuProviderInjection + html[idx:]
	}
	return html + zhipuProviderInjection
}
