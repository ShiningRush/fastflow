# 🎯 DAG Visualizer
 
> **专业的工作流可视化工具** - 基于React + ReactFlow的Chrome扩展

[![Chrome Extension](https://img.shields.io/badge/Chrome-Extension-blue.svg)](https://github.com/xkcoding/dag-visualization)
[![React](https://img.shields.io/badge/React-19.x-blue.svg)](https://reactjs.org/)
[![ReactFlow](https://img.shields.io/badge/ReactFlow-11.x-orange.svg)](https://reactflow.dev/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.x-blue.svg)](https://www.typescriptlang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## 项目简介
DAG Visualizer 是一个专业级的有向无环图（DAG）工作流可视化Chrome扩展，采用现代化的React技术栈构建。专注于为开发者、数据分析师和工作流设计师提供直观、高效、专业的DAG可视化体验。

## ✨ 核心特性

### 🧠 智能布局系统
- **自动连线优化** - 智能检测和避免连线穿越问题
- **层级感知布局** - 基于拓扑排序的智能节点分层
- **多种布局模式** - 支持纵向/横向布局切换
### 🎨 专业编辑体验
- **Monaco Editor集成** - VS Code级别的JSON编辑体验，完全本地化
- **离线编辑器** - Monaco Editor 0.52.2本地资源，无CDN依赖
- **智能节点创建** - 右键创建，支持多种预设类型
### 🎛️ 强大功能集
- **多格式导出** - PNG/JPG/SVG高质量图片导出
- **节点颜色管理** - 批量颜色控制 + localStorage持久化
- **画布操作** - 缩放、平移、小地图导航

## 🚀 快速开始

### 安装与运行

```bash
# 安装依赖
npm install

# 开发模式
npm run dev

# 构建扩展
npm run build
```

参考资料

https://github.com/xkcoding/dag-visualization