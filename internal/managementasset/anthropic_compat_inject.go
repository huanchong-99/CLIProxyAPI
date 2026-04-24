package managementasset

import (
	"strings"
)

const anthropicCompatProviderInjection = `<script id="anthropic-compat-provider-injection">
(function () {
  var MANAGEMENT_BASE = "/v0/management";
  var ROOT_ID = "provider-anthropic-compat";
  var STYLE_ID = "anthropic-compat-provider-style";
  var _acAuthToken = "";
  function _acTryStorage() {
    try { var k = localStorage.getItem("managementKey"); if (k) _acAuthToken = "Bearer " + k; } catch(e) {}
    try { for (var i = 0; i < localStorage.length; i++) { var sk = localStorage.key(i); if (sk && sk.indexOf("enc-") === 0 && sk.indexOf("managementKey") !== -1) { var v = localStorage.getItem(sk); if (v) { try { v = atob(v); } catch(e2) {} if (v) _acAuthToken = "Bearer " + v; } } } } catch(e) {}
  }
  _acTryStorage();
  var _origXHROpen = XMLHttpRequest.prototype.open;
  var _origXHRSetHdr = XMLHttpRequest.prototype.setRequestHeader;
  XMLHttpRequest.prototype.open = function () { this._acReqArgs = arguments; return _origXHROpen.apply(this, arguments); };
  XMLHttpRequest.prototype.setRequestHeader = function (n, v) {
    if (n && n.toLowerCase() === "authorization" && v && v.indexOf("Bearer ") === 0) _acAuthToken = v;
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
          if (auth && auth.indexOf("Bearer ") === 0) _acAuthToken = auth;
        }
      } catch (e) {}
    }
    return _origFetch.apply(this, a);
  };
  var state = { configs: [], loading: false, status: "", error: false, loaded: false };

  var ANTHROPIC_SVG = '<svg xmlns="http://www.w3.org/2000/svg" width="1em" height="1em" viewBox="0 0 24 24" fill="none" style="flex:none;line-height:1"><title>Anthropic Compatible</title><path d="M13.827 3.52h3.603L24 20.48h-3.603l-6.57-16.96zm-7.258 0h3.767L16.906 20.48h-3.674l-1.472-3.906H5.69l-1.482 3.906H.6L6.569 3.52zM9.9 13.418l-2.584-6.76-2.585 6.76H9.9z" fill="#D97757"/></svg>';
  var ANTHROPIC_ICON_DATAURI = "data:image/svg+xml;utf8," + encodeURIComponent(ANTHROPIC_SVG);

  function request(path, options) {
    var opts = options || {};
    var headers = opts.headers || {};
    if (_acAuthToken) headers["Authorization"] = _acAuthToken;
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
      name: String(item.name || "").trim(),
      prefix: String(item.prefix || "").trim(),
      baseUrl: String(item["base-url"] || item.baseUrl || "").trim(),
      apiKeyEntries: Array.isArray(item["api-key-entries"]) ? item["api-key-entries"] : [],
      models: Array.isArray(item.models) ? item.models : [],
      headers: item.headers && typeof item.headers === "object" && !Array.isArray(item.headers) ? item.headers : {}
    };
  }

  function toPayload(c) {
    var p = { name: c.name, "base-url": c.baseUrl };
    if (c.prefix) p.prefix = c.prefix;
    var h = c.headers || {};
    if (Object.keys(h).length) p.headers = h;
    if (c.apiKeyEntries.length) p["api-key-entries"] = c.apiKeyEntries;
    if (c.models.length) p.models = c.models;
    return p;
  }

  var AC_CSS = [
    "#" + ROOT_ID + " { margin-top: 16px; }",
    ".ac-card { border: 1px solid var(--border-color, rgba(148,163,184,.35)); border-radius: 18px; padding: 18px; background: var(--card-bg, rgba(255,255,255,.96)); box-shadow: 0 18px 50px rgba(15,23,42,.08); }",
    ".ac-head { display: flex; justify-content: space-between; gap: 12px; flex-wrap: wrap; align-items: flex-start; margin-bottom: 14px; }",
    ".ac-title-row { display: flex; align-items: center; gap: 8px; }",
    ".ac-title-icon { width: 22px; height: 22px; flex: none; display: inline-block; }",
    ".ac-title { margin: 0; font-size: 20px; font-weight: 700; color: var(--text-primary, #0f172a); }",
    ".ac-sub { margin: 4px 0 0; color: var(--text-secondary, #475569); font-size: 13px; line-height: 1.5; }",
    ".ac-actions { display: flex; gap: 8px; flex-wrap: wrap; }",
    ".ac-btn { border: 0; border-radius: 999px; padding: 8px 14px; font-size: 13px; font-weight: 600; cursor: pointer; transition: opacity .15s; }",
    ".ac-btn:hover { opacity: .85; }",
    ".ac-btn.primary { background: #D97757; color: #fff; }",
    ".ac-btn.secondary { background: var(--btn-secondary-bg, rgba(15,23,42,.07)); color: var(--text-primary, #0f172a); }",
    ".ac-btn.danger { background: rgba(220,38,38,.09); color: #b91c1c; }",
    ".ac-btn:disabled { opacity: .5; cursor: not-allowed; }",
    ".ac-status { margin: 0 0 12px; padding: 10px 12px; border-radius: 12px; font-size: 13px; line-height: 1.5; background: rgba(217,119,87,.08); color: #b45309; }",
    ".ac-status.error { background: rgba(220,38,38,.09); color: #b91c1c; }",
    ".ac-meta { display: flex; gap: 8px 12px; flex-wrap: wrap; margin-bottom: 14px; }",
    ".ac-pill { display: inline-flex; gap: 6px; align-items: center; padding: 5px 10px; border-radius: 999px; background: rgba(217,119,87,.08); color: #b45309; font-size: 12px; font-weight: 600; }",
    ".ac-empty { padding: 18px; border: 1px dashed rgba(148,163,184,.5); border-radius: 14px; color: #64748b; text-align: center; }",
    ".ac-list { display: grid; gap: 12px; }",
    ".ac-item { border: 1px solid rgba(148,163,184,.25); border-radius: 16px; padding: 16px; background: var(--item-bg, rgba(248,250,252,.8)); }",
    ".ac-item-head, .ac-item-foot { display: flex; justify-content: space-between; gap: 12px; flex-wrap: wrap; align-items: center; }",
    ".ac-item-head { margin-bottom: 10px; }",
    ".ac-item-title { font-size: 15px; font-weight: 700; }",
    ".ac-badges { display: flex; gap: 6px; flex-wrap: wrap; }",
    ".ac-badge { display: inline-flex; padding: 4px 8px; border-radius: 999px; background: rgba(148,163,184,.15); color: #334155; font-size: 11px; font-weight: 700; }",
    ".ac-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px,1fr)); gap: 10px; margin-bottom: 10px; }",
    ".ac-field { display: flex; flex-direction: column; gap: 3px; }",
    ".ac-label { font-size: 11px; font-weight: 700; letter-spacing: .04em; text-transform: uppercase; color: #64748b; }",
    ".ac-value { font-size: 13px; line-height: 1.4; word-break: break-all; }",
    ".ac-note { font-size: 12px; color: #64748b; }"
  ].join("\n");

  function ensureStyle() {
    if (document.getElementById(STYLE_ID)) return;
    var s = document.createElement("style");
    s.id = STYLE_ID;
    s.textContent = AC_CSS;
    document.head.appendChild(s);
  }

  function ensureRoot() {
    var anchor = document.getElementById("provider-deepseek") || document.getElementById("provider-zhipu") || document.getElementById("provider-openai") || document.getElementById("provider-vertex");
    if (!anchor || !anchor.parentNode) return null;
    var root = document.getElementById(ROOT_ID);
    if (!root) {
      root = document.createElement("div");
      root.id = ROOT_ID;
      anchor.insertAdjacentElement("afterend", root);
    }
    if (!root._acBound) {
      root.addEventListener("click", onClick);
      root._acBound = true;
    }
    return root;
  }

  function render() {
    var root = ensureRoot();
    if (!root) return;
    ensureStyle();
    var statusHtml = state.status ? '<div class="ac-status' + (state.error ? ' error' : '') + '">' + esc(state.status) + '</div>' : '';
    var body;
    if (state.loading && !state.configs.length) {
      body = '<div class="ac-empty">正在加载 Anthropic 兼容提供商配置...</div>';
    } else if (!state.configs.length) {
      body = '<div class="ac-empty">暂无 Anthropic 兼容提供商，点击上方"添加提供商"按钮添加。支持任何使用 Anthropic/Claude API 协议的第三方服务。</div>';
    } else {
      body = '<div class="ac-list">' + state.configs.map(function (c, i) {
        var keyCount = c.apiKeyEntries.length;
        var badges = '<span class="ac-badge">密钥 ' + keyCount + '</span><span class="ac-badge">模型 ' + c.models.length + '</span>';
        return '<article class="ac-item">' +
          '<div class="ac-item-head"><div class="ac-item-title">' + esc(c.name || "未命名") + '</div><div class="ac-badges">' + badges + '</div></div>' +
          '<div class="ac-grid">' +
          '<div class="ac-field"><span class="ac-label">Base URL</span><span class="ac-value">' + esc(c.baseUrl || "-") + '</span></div>' +
          '<div class="ac-field"><span class="ac-label">前缀</span><span class="ac-value">' + esc(c.prefix || "-") + '</span></div>' +
          '<div class="ac-field"><span class="ac-label">API 密钥</span><span class="ac-value"><code>' + esc(c.apiKeyEntries.map(function(k){ return mask(k["api-key"] || k.apiKey); }).join(", ") || "-") + '</code></span></div>' +
          '</div>' +
          '<div class="ac-item-foot"><div class="ac-note">' + esc(c.models.map(function(m){ return m.alias || m.name; }).slice(0,5).join("、") || "未配置模型") + '</div>' +
          '<div class="ac-actions"><button type="button" class="ac-btn secondary" data-action="edit" data-index="' + i + '">编辑</button>' +
          '<button type="button" class="ac-btn danger" data-action="delete" data-index="' + i + '">删除</button></div></div></article>';
      }).join('') + '</div>';
    }
    root.innerHTML = '<section class="ac-card">' +
      '<div class="ac-head"><div><div class="ac-title-row"><img class="ac-title-icon" alt="Anthropic" src="' + ANTHROPIC_ICON_DATAURI + '"/><h2 class="ac-title">Anthropic 格式兼容提供商</h2></div>' +
      '<p class="ac-sub">通用 Anthropic/Claude API 兼容提供商。支持任何实现了 Anthropic Messages API 协议的第三方服务（如 DeepSeek、自建代理等）。</p></div>' +
      '<div class="ac-actions"><button type="button" class="ac-btn secondary" data-action="refresh"' + (state.loading ? ' disabled' : '') + '>刷新</button>' +
      '<button type="button" class="ac-btn primary" data-action="add"' + (state.loading ? ' disabled' : '') + '>添加提供商</button></div></div>' +
      statusHtml +
      '<div class="ac-meta"><span class="ac-pill">提供商 <strong>' + state.configs.length + '</strong></span></div>' +
      body + '</section>';
  }

  function saveAll(configs, msg) {
    state.loading = true; state.status = "保存中..."; state.error = false; render();
    return request("/anthropic-compatibility", { method: "PUT", body: JSON.stringify(configs.map(toPayload)) })
      .then(function () { return load(true).then(function(){ state.status = msg; state.error = false; render(); }); })
      .catch(function (e) { state.status = "保存失败：" + (e && e.message ? e.message : e); state.error = true; render(); })
      .finally(function () { state.loading = false; render(); });
  }

  function editConfig(index) {
    var cur = index >= 0 && state.configs[index] ? state.configs[index] : { name: "", prefix: "", baseUrl: "", apiKeyEntries: [], models: [], headers: {} };
    var name = prompt("提供商名称（如 my-claude-proxy）", cur.name || "");
    if (name === null) return;
    name = name.trim();
    if (!name) { alert("名称不能为空。"); return; }
    var baseUrl = prompt("Base URL（Anthropic 兼容端点）", cur.baseUrl || "");
    if (baseUrl === null) return;
    baseUrl = baseUrl.trim();
    if (!baseUrl) { alert("Base URL 不能为空。"); return; }
    var apiKeysRaw = prompt("API 密钥（多个使用分号分隔）", cur.apiKeyEntries.map(function(k){ return k["api-key"] || k.apiKey || ""; }).join("; "));
    if (apiKeysRaw === null) return;
    var apiKeyEntries = [];
    String(apiKeysRaw || "").split(/[;\r\n]+/).forEach(function(k) {
      k = k.trim(); if (k) apiKeyEntries.push({ "api-key": k });
    });
    var prefix = prompt("前缀（可选，留空表示无）", cur.prefix || "");
    if (prefix === null) return;
    var modelsRaw = prompt("模型：上游名称=>别名，多个使用分号分隔", cur.models.map(function(m){ return m.name + "=>" + m.alias; }).join("; "));
    if (modelsRaw === null) return;
    var models = [];
    String(modelsRaw || "").split(/[;\r\n]+/).forEach(function(p) {
      p = p.trim(); if (!p) return;
      var mname, alias;
      if (p.indexOf("=>") !== -1) { mname = p.split("=>")[0].trim(); alias = p.split("=>")[1].trim(); }
      else { mname = p; alias = p; }
      if (mname) models.push({ name: mname, alias: alias || mname });
    });
    var next = state.configs.slice();
    next[index >= 0 ? index : next.length] = { name: name, prefix: prefix.trim(), baseUrl: baseUrl, apiKeyEntries: apiKeyEntries, models: models, headers: cur.headers };
    saveAll(next, index >= 0 ? "配置已更新。" : "配置已添加。");
  }

  function deleteConfig(index) {
    var c = state.configs[index];
    if (!c || !confirm("确认删除 Anthropic 兼容提供商 \"" + (c.name || "#" + (index+1)) + "\"？")) return;
    state.loading = true; state.status = "删除中..."; state.error = false; render();
    var q = c.name ? "?name=" + encodeURIComponent(c.name) : "?index=" + index;
    request("/anthropic-compatibility" + q, { method: "DELETE" })
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
    return request("/anthropic-compatibility").catch(function(){ return { "anthropic-compatibility": [] }; })
    .then(function(r) {
      state.configs = Array.isArray(r && r["anthropic-compatibility"]) ? r["anthropic-compatibility"].map(normalizeConfig) : [];
      if (!silent) { state.status = ""; state.error = false; }
    }).catch(function(e) {
      state.status = "加载失败：" + (e && e.message ? e.message : e); state.error = true;
    }).finally(function() { state.loading = false; state.loaded = true; render(); });
  }

  function ensureNavButton() {
    var list = document.querySelector('[class*="ProviderNav-module__navList"]');
    if (!list) return;
    if (list.querySelector('[data-ac-nav]')) return;
    var template = list.querySelector('button[title]');
    if (!template) return;
    var btn = document.createElement("button");
    btn.className = template.className;
    btn.type = "button";
    btn.title = "Anthropic Compat";
    btn.setAttribute("aria-label", "Anthropic Compatible Providers");
    btn.setAttribute("aria-pressed", "false");
    btn.setAttribute("data-ac-nav", "1");
    var img = document.createElement("img");
    img.alt = "Anthropic Compat";
    var tImg = template.querySelector("img");
    if (tImg) img.className = tImg.className;
    img.src = ANTHROPIC_ICON_DATAURI;
    btn.appendChild(img);
    list.appendChild(btn);
  }

  function installNavPatch() {
    var list = document.querySelector('[class*="ProviderNav-module__navList"]');
    if (!list || list._acNavPatch) return;
    var indicator = list.querySelector('[class*="ProviderNav-module__indicator"]');
    if (!indicator) return;
    list._acNavPatch = true;
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
      if (btn.getAttribute("data-ac-nav") === "1") {
        var acRoot = document.getElementById(ROOT_ID);
        if (acRoot && acRoot.scrollIntoView) {
          acRoot.scrollIntoView({ behavior: "smooth", block: "start" });
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
    if (!root._acRendered) { render(); root._acRendered = true; }
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

// InjectAnthropicCompatProvider inserts the Anthropic-compatible provider UI script
// into the management HTML page immediately before the closing </body> tag.
func InjectAnthropicCompatProvider(html string) string {
	closingBody := "</body>"
	if idx := strings.LastIndex(strings.ToLower(html), closingBody); idx >= 0 {
		return html[:idx] + anthropicCompatProviderInjection + html[idx:]
	}
	return html + anthropicCompatProviderInjection
}
