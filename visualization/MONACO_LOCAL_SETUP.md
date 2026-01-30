# Monaco Editor 本地化配置

本文档说明了如何将Monaco Editor的CDN资源本地化，以确保Chrome扩展能够正常加载编辑器。

## 问题背景

Chrome扩展在打包后可能无法加载以下CDN资源：
- `https://cdn.jsdelivr.net/npm/monaco-editor@0.52.2/min/vs/loader.js`
- `https://cdn.jsdelivr.net/npm/monaco-editor@0.52.2/min/vs/base/worker/workerMain.js`

## 解决方案

### 1. 本地资源准备

已将完整的Monaco Editor资源复制到本地：

```bash
# 创建目录结构
mkdir -p public/monaco-editor/min/vs/base/worker

# 从node_modules复制完整资源
cp -r node_modules/monaco-editor/min/vs/* public/monaco-editor/min/vs/
```

### 2. 代码配置

在 `src/components/JSONInputArea.tsx` 中配置本地路径：

```typescript
import Editor, { loader } from '@monaco-editor/react';

// 配置Monaco Editor使用本地资源
loader.config({
  paths: {
    vs: '/monaco-editor/min/vs'
  }
});
```

### 3. Chrome扩展权限

在 `public/manifest.json` 中确保web资源可访问：

```json
{
  "web_accessible_resources": [
    {
      "resources": ["*"],
      "matches": ["<all_urls>"]
    }
  ]
}
```

## 文件结构

构建后的Monaco Editor本地文件结构：

```
dist/
├── monaco-editor/
│   └── min/
│       └── vs/
│           ├── base/
│           │   └── worker/
│           │       └── workerMain.js     # Web Worker主文件
│           ├── basic-languages/          # 基础语言支持
│           ├── editor/                   # 编辑器核心
│           ├── language/                 # 语言服务
│           ├── loader.js                 # Monaco加载器
│           └── nls.messages.*.js         # 国际化文件
```

## 版本信息

- Monaco Editor版本: 0.52.2
- @monaco-editor/react版本: 4.7.0

## 验证方法

1. 构建项目：`npm run build`
2. 检查dist目录是否包含monaco-editor文件夹
3. 确认loader.js和workerMain.js存在
4. 在Chrome扩展中测试JSON编辑器功能

## 注意事项

- 本地化资源会增加扩展包大小（约1.8MB）
- 所有Monaco Editor功能（语法高亮、代码提示等）都能正常工作
- 支持多种语言和主题
- 离线环境下也能正常使用

## 更新Monaco Editor

如需更新Monaco Editor版本：

1. 更新package.json中的monaco-editor依赖
2. 重新复制本地资源：`cp -r node_modules/monaco-editor/min/vs/* public/monaco-editor/min/vs/`
3. 重新构建项目

通过本地化配置，Chrome扩展现在可以完全离线工作，不再依赖外部CDN资源。