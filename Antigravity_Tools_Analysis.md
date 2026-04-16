# Antigravity IDE — AI 工具完整分析文档

> 分析来源：`D:\Antigravity\resources\app\out\jetskiAgent\main.js` (11MB)  
> 分析时间：2026-04-16

---

## 一、架构概览

```
用户操作
  ↓
Electron UI (jetskiAgent/main.js)
  ↓ gRPC / JSON-RPC (本地 IPC)
Language Server (language_server_windows_x64.exe)
  ↓ HTTPS / protobuf
AI 后端 API (Anthropic Claude / Google Gemini)
```

- UI 与 Language Server 之间走**本地 RPC**，不走 HTTP
- Language Server 才是真正发起外部 AI API 请求的进程
- 反代要拦截的是 **Language Server 的出站请求**，不是 UI 层

---

## 二、支持的 AI 后端模型

从枚举值确认，Antigravity 同时支持两套 API：

| 标识 | 含义 |
|------|------|
| `USE_ANTHROPIC_TOKEN_EFFICIENT_TOOLS_BETA=296` | **Anthropic Claude API**（tool_use 格式） |
| `GOOGLE_GEMINI_RIFTRUNNER=348` | **Google Gemini API**（functionCall 格式） |
| `GOOGLE_GEMINI_RIFTRUNNER_THINKING_HIGH=353` | Gemini 思考模式（高） |
| `GOOGLE_GEMINI_RIFTRUNNER_THINKING_LOW=352` | Gemini 思考模式（低） |
| `XML_TOOL_PARSING_MODELS=268` | XML 格式工具调用（旧版兼容） |

**重要**：反代需要区分当前会话用的是哪个后端，工具调用格式完全不同。

---

## 三、全部工具类型（Cascade Step 枚举）

这是 Antigravity AI 能调用的所有工具，来自 `CascadeStep` 及相关枚举：

### 3.1 文件操作类
| 枚举值 | 编号 | 说明 |
|--------|------|------|
| `VIEW_FILE` | 8 | 查看文件内容 |
| `VIEW_FILE_OUTLINE` | 47 | 查看文件大纲 |
| `RECORD_FILES` | 47 | 记录文件列表 |
| `FILE_LINE_RANGE` | 9 | 读取文件行范围 |
| `WRITE_BLOB` | 128 | 写入二进制文件 |
| `CODE_SEARCH` | 73 | 代码搜索 |
| `GREP_SEARCH` | 7 | Grep 搜索 |
| `DIFF_SEARCH` | 2 | Diff 搜索 |
| `TRAJECTORY_SEARCH` | 60 | 轨迹搜索 |

### 3.2 终端 / Shell 执行类
| 枚举值 | 编号 | 说明 |
|--------|------|------|
| `RUN_COMMAND` | 21 | **运行命令（Bash）** |
| `SHELL_EXEC` | 112 | Shell 执行 |
| `TERMINAL_STEP_TYPE` | 8 | 终端步骤 |
| `READ_TERMINAL` | 65 | 读取终端输出 |
| `RUN_EXTENSION_CODE` | 57 | 运行扩展代码 |

### 3.3 Web / 网络类
| 枚举值 | 编号 | 说明 |
|--------|------|------|
| `SEARCH_WEB` | 33 | **Web 搜索** |
| `WEB_SEARCH` | 1/4 | Web 搜索（另一枚举组） |
| `READ_URL_CONTENT` | 31 | **读取 URL 内容（WebFetch）** |
| `READ_BROWSER_PAGE` | 67 | 读取浏览器页面 |
| `OPEN_BROWSER_URL` | 3 | 打开浏览器 URL |
| `EXECUTE_BROWSER_JAVASCRIPT` | 61 | 浏览器执行 JS |
| `GROUNDING_WITH_GOOGLE_SEARCH` | 1 | Google 搜索接地 |

### 3.4 Agent / Subagent 类
| 枚举值 | 编号 | 说明 |
|--------|------|------|
| `INVOKE_SUBAGENT` | 127 | **调用 Subagent** |
| `BROWSER_SUBAGENT` | 85 | **浏览器 Subagent** |
| `SUBAGENT` | 3/16 | Subagent 类型 |
| `BOTH_AGENTS` | 3 | 主 Agent + Subagent |
| `MAIN_AGENT_ONLY` | 1 | 仅主 Agent |
| `SUBAGENT_ONLY` | 4 | 仅 Subagent |
| `SUBAGENT_PRIMARILY` | 2 | 以 Subagent 为主 |
| `GLOBAL_AGENT` | 7 | 全局 Agent |
| `AGENCY_TOOL_CALL` | 103 | Agency 工具调用 |

### 3.5 MCP 工具类
| 枚举值 | 编号 | 说明 |
|--------|------|------|
| `MCP_TOOL` | 38 | **MCP 工具调用** |
| `MCP_RESOURCE` | 18 | MCP 资源读取 |
| `MCP_TOOLS` | 4 | MCP 工具集合 |
| `CASCADE_MCP_SERVER_INIT` | 64 | MCP 服务器初始化 |
| `CASCADE_ENABLE_MCP_TOOLS` | 245 | 启用 MCP 工具 |

### 3.6 Skill 类
| 枚举值 | 编号 | 说明 |
|--------|------|------|
| `SKILL` | 3 | **Skill 调用** |
| `SKILLS` | 4/5 | Skills 集合 |
| `GLOBAL_SKILLS` | 5 | 全局 Skills |
| `goto.google.com/jetski-skills` | URL | Skills 文档地址 |

### 3.7 代码编辑类
| 枚举值 | 编号 | 说明 |
|--------|------|------|
| `PROPOSE_CODE` | 24 | 提议代码 |
| `PRODUCED_CODE_DIFF` | 1 | 产生代码 Diff |
| `ACKNOWLEDGE_CASCADE_CODE_EDIT` | 44 | 确认代码编辑 |
| `COMMAND_EDIT` | 5 | 命令编辑 |
| `SEARCH_REPLACE` | 2 | 搜索替换 |
| `TAB_JUMP_EDIT` | 10 | Tab 跳转编辑 |
| `CASCADE_USE_REPLACE_CONTENT_EDIT_TOOL` | 228 | 替换内容工具 |

### 3.8 记忆 / 上下文类
| 枚举值 | 编号 | 说明 |
|--------|------|------|
| `RETRIEVE_MEMORY` | 34 | **检索记忆** |
| `MEMORY` | 8/29 | 记忆 |
| `CASCADE_MEMORIES_EDIT` | 85 | 编辑记忆 |
| `CASCADE_MEMORY_DELETED` | 42 | 删除记忆 |
| `READ_RESOURCE` | 52 | 读取资源 |

### 3.9 工具调用控制类
| 枚举值 | 编号 | 说明 |
|--------|------|------|
| `TOOL_CALL_PROPOSAL` | 12/40 | **工具调用提案** |
| `TOOL_CALL_CHOICE` | 41 | 工具调用选择 |
| `NO_TOOL_CALL` | 4 | 不调用工具 |
| `ALL_TOOLS` | 1 | 所有工具 |
| `CLIENT_TOOL_PARSE_ERROR` | 17 | 工具解析错误 |

---

## 四、缓存命中相关

| 枚举值 | 编号 | 说明 |
|--------|------|------|
| `PROMPT_CACHE` | 2 | Prompt 缓存 |
| `PROMPT_CACHE_TTL_EXPIRED` | 1 | 缓存 TTL 过期 |
| `CACHE` | 2 | 缓存 |
| `CACHED_MESSAGE` | 18 | 缓存命中消息 |
| `CACHE_VALUE` | 0 | 缓存值 |
| `COMMAND_PROMPT_CACHE_CONFIG` | 255 | 命令提示缓存配置 |
| `SUPERCOMPLETE_CACHE_HIT` | 196 | Supercomplete 缓存命中 |
| `TAB_JUMP_CACHE_HIT` | 189 | Tab Jump 缓存命中 |
| `DISABLE_COMPLETIONS_CACHE` | 118 | 禁用补全缓存 |

**Anthropic 缓存格式**（`USE_ANTHROPIC_TOKEN_EFFICIENT_TOOLS_BETA` 启用时）：
```json
{
  "type": "text",
  "text": "...",
  "cache_control": { "type": "ephemeral" }
}
```

---

## 五、Anthropic 工具调用原生格式

当后端为 Anthropic Claude 时，工具调用走标准 Messages API：

### AI 调用工具（tool_use）
```json
{
  "role": "assistant",
  "content": [
    {
      "type": "tool_use",
      "id": "toolu_01Xxxxxxxxxxxxxxxxxxxxxq",
      "name": "bash",
      "input": {
        "command": "ls -la /tmp"
      }
    }
  ]
}
```

### 工具返回结果（tool_result）
```json
{
  "role": "user",
  "content": [
    {
      "type": "tool_result",
      "tool_use_id": "toolu_01Xxxxxxxxxxxxxxxxxxxxxq",
      "content": "total 8\ndrwxr-xr-x..."
    }
  ]
}
```

### 工具定义（tools 数组）
```json
{
  "tools": [
    {
      "name": "bash",
      "description": "Run a bash command",
      "input_schema": {
        "type": "object",
        "properties": {
          "command": { "type": "string", "description": "The bash command to run" }
        },
        "required": ["command"]
      }
    },
    {
      "name": "web_search",
      "description": "Search the web",
      "input_schema": {
        "type": "object",
        "properties": {
          "query": { "type": "string" }
        },
        "required": ["query"]
      }
    },
    {
      "name": "read_url",
      "description": "Fetch content from a URL",
      "input_schema": {
        "type": "object",
        "properties": {
          "url": { "type": "string" }
        },
        "required": ["url"]
      }
    }
  ]
}
```

---

## 六、Gemini 工具调用原生格式

当后端为 `GOOGLE_GEMINI_RIFTRUNNER` 时：

### AI 调用工具（functionCall）
```json
{
  "role": "model",
  "parts": [
    {
      "functionCall": {
        "name": "bash",
        "args": {
          "command": "ls -la /tmp"
        }
      }
    }
  ]
}
```

### 工具返回结果（functionResponse）
```json
{
  "role": "user",
  "parts": [
    {
      "functionResponse": {
        "name": "bash",
        "response": {
          "output": "total 8\ndrwxr-xr-x..."
        }
      }
    }
  ]
}
```

### 工具定义（tools 数组）
```json
{
  "tools": [
    {
      "functionDeclarations": [
        {
          "name": "bash",
          "description": "Run a bash command",
          "parameters": {
            "type": "OBJECT",
            "properties": {
              "command": { "type": "STRING" }
            },
            "required": ["command"]
          }
        }
      ]
    }
  ]
}
```

---

## 七、Subagent / TeamAgent 调用格式

从代码中找到的 subagent 调用 RPC：

```javascript
// 跳过浏览器 subagent
skipBrowserSubagent(cascadeId, stepIndex)

// 取消特定 step（包含 subagent steps）
cancelCascadeSteps(cascadeId, stepIndices)

// Subagent 类型枚举
INVOKE_SUBAGENT = 127    // 通用 subagent 调用
BROWSER_SUBAGENT = 85   // 浏览器专用 subagent
```

Subagent 在 Cascade Step 中的结构（protobuf 反序列化后）：
```json
{
  "stepType": "INVOKE_SUBAGENT",
  "cascadeId": "...",
  "stepIndex": 3,
  "subagentConfig": {
    "agentType": "BROWSER_SUBAGENT",
    "instructions": "...",
    "tools": ["READ_BROWSER_PAGE", "EXECUTE_BROWSER_JAVASCRIPT"]
  }
}
```

---

## 八、MCP 工具调用格式

MCP 服务器配置（`schemas/mcp_config.schema.json`）：
```json
{
  "mcpServers": {
    "my-server": {
      "command": "node",
      "args": ["server.js"],
      "env": { "KEY": "value" },
      "serverUrl": "http://localhost:3000",
      "disabled": false,
      "disabledTools": ["tool_name"],
      "headers": { "Authorization": "Bearer xxx" },
      "tools": {
        "my_tool": { "background": "always" }
      }
    }
  }
}
```

MCP 工具调用在 Cascade Step 中（枚举 `MCP_TOOL=38`）：
```json
{
  "stepType": "MCP_TOOL",
  "toolName": "server_name__tool_name",
  "toolInput": { ... },
  "toolResult": { ... }
}
```

---

## 九、Skill 调用

Skill 通过 Language Server RPC 获取和调用：

```javascript
// 获取所有 Skills
getAllSkills(workspaceUris)  // → skills[]

// 扫描 Skill 配置文件
scanSkillsConfigFile(configFilePath, workspaceUri)

// Skill 显示在 sidebar customView
customView: { customPaneKey: "skills", serializedInputs: "{}" }
```

Skill 在 Cascade Step 中（枚举 `SKILL=3`）：
```json
{
  "stepType": "SKILL",
  "skillId": "...",
  "skillInput": { ... }
}
```

---

## 十、Language Server RPC 完整接口列表

UI 通过这些 RPC 调用 Language Server：

### 会话管理
- `startCascade(config)` — 启动新的 AI 会话
- `sendUserCascadeMessage(message)` — 发送用户消息
- `handleCascadeUserInteraction(cascadeId, interaction)` — 处理用户交互
- `cancelCascadeInvocation(cascadeId)` — 取消 AI 调用
- `cancelCascadeSteps(cascadeId, stepIndices)` — 取消特定步骤
- `sendAllQueuedMessages(cascadeId, cascadeConfig)` — 发送所有队列消息

### 流式更新
- `streamCascadeReactiveUpdates(...)` — 接收 AI 流式输出
- `streamAgentStateUpdates(...)` — Agent 状态流
- `requestAgentStatePageUpdate(...)` — 请求状态页更新

### 工具与资源
- `getAllSkills(workspaceUris)` — 获取所有 Skills
- `getAllWorkflows(workspaceUris)` — 获取所有 Workflows
- `getAllRules(workspaceUris)` — 获取所有 Rules
- `getMcpServerStates()` — 获取 MCP 服务器状态
- `refreshMcpServers(config)` — 刷新 MCP 服务器
- `listMcpResources(params)` — 列出 MCP 资源
- `listMcpPrompts()` — 列出 MCP Prompts
- `getMcpPrompt(serverName, name, arguments)` — 获取 MCP Prompt
- `getAllCustomAgentConfigs()` — 获取自定义 Agent 配置
- `getAllPlugins()` — 获取所有插件

### 文件操作
- `readFile(uri)` — 读取文件
- `writeFile(uri, content, overwrite)` — 写入文件
- `searchFiles(params)` — 搜索文件

### 浏览器操作
- `listBrowserPages()` — 列出浏览器页面
- `focusUserPage(pageId)` — 聚焦页面
- `captureScreenshot(pageId)` — 截图
- `captureConsoleLogs(pageId)` — 捕获控制台日志
- `skipBrowserSubagent(cascadeId, stepIndex)` — 跳过浏览器 Subagent

### 记忆与上下文
- `getUserMemories()` — 获取用户记忆
- `getTokenBase(params)` — 获取 Token 基础

---

## 十一、已知后端 API 端点

| 端点 | 用途 |
|------|------|
| `https://agent-marketplace.corp.google.com` | Agent 市场（内部） |
| `https://wf-boring-coyote.corp.google.com/?trajectory_id=` | 轨迹调试日志 |
| `https://feedback-pa.googleapis.com/v1/feedback/products/{id}/web:submit` | 用户反馈提交 |
| `https://main.vscode-cdn.net/sourcemaps/.../jetskiAgent/main.js.map` | Source Map |

---

## 十二、反代修复建议

### 问题根因
反代可能在以下环节出错：

1. **工具格式不匹配**：Anthropic 用 `tool_use`/`tool_result`，Gemini 用 `functionCall`/`functionResponse`，需要根据请求 URL 或 model 字段区分

2. **Subagent 调用缺失**：`INVOKE_SUBAGENT=127` 和 `BROWSER_SUBAGENT=85` 是两个独立步骤类型，需要分别处理

3. **缓存字段丢失**：Anthropic 格式需要在 content block 中保留 `cache_control` 字段，Antigravity 会用到 `USE_ANTHROPIC_TOKEN_EFFICIENT_TOOLS_BETA`

4. **MCP 工具名格式**：MCP 工具名是 `{server_name}__{tool_name}` 双下划线格式

5. **Skill 调用**：走 Language Server 本地 RPC，不经过外部 API，反代无法拦截，除非代理 Language Server 的 socket

6. **流式协议**：`streamCascadeReactiveUpdates` 是 gRPC 流，不是 HTTP SSE，如果反代的是 Language Server 端口需要支持 gRPC streaming

### 快速验证
抓 Language Server 进程的网络流量（不是 Electron UI），才能看到真实 AI API 请求格式。

---

*文档生成于 2026-04-16，基于 D:\Antigravity\resources\app\out\jetskiAgent\main.js 静态分析*
