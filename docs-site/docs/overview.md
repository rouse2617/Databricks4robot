---
title: 概览
sidebar_label: 概览
description: Cyber Databrew 是面向视频与多模态数据的流水线平台，提供资产管理、算法编排、检索、交付、REST API 与 Python SDK。
---

# 概览

**Cyber Databrew** 是一个面向视频与多模态数据的流水线平台，专注于**资产元数据管理、算法运行编排、检索与交付**。平台提供统一的 API 和 Python SDK，帮助团队高效管理从数据摄入、标注、算法处理到交付的完整生命周期。

<div className="overview-quicklinks">
  <a className="overview-quicklink" href="/doc/getting-started/quickstart">
    <span className="overview-quicklink__eyebrow">开始使用</span>
    <strong>快速开始</strong>
    <span>用最短路径完成认证、资产创建和基础查询。</span>
  </a>
  <a className="overview-quicklink" href="/doc/getting-started/sdk-installation">
    <span className="overview-quicklink__eyebrow">Python</span>
    <strong>SDK 安装</strong>
    <span>安装客户端并用代码接入 Databrew API。</span>
  </a>
  <a className="overview-quicklink" href="/doc/api/reference">
    <span className="overview-quicklink__eyebrow">REST</span>
    <strong>API Reference</strong>
    <span>查看可交互的接口、参数和响应结构。</span>
  </a>
</div>

## 核心能力

### 资产管理（Assets）
- 资产元数据的 CRUD 操作
- 版本管理和生命周期状态追踪
- 标签系统（多源标签合并）
- 血缘关系与谱系追踪（Lineage & Provenance）
- 浏览记录和收藏功能
- MCAP 文件关联与存储

### 算法编排（Algo Runs）
- 算法运行的全生命周期管理（创建 → 启动 → 完成/取消）
- 支持按资产列表批量执行
- 受影响资产自动关联
- 与注册中心集成，支持多种算法类型

### 交付管理（Deliveries）
- 面向客户的数据交付
- 草稿 → 添加内容 → 提交的两步提交模式
- 交付规则引擎
- 确认/取消/重试机制

### 检索与查询
- 基于 Elasticsearch 的全局搜索
- Query IR（中间表示）查询引擎
- 保存查询管理
- 标签和过滤搜索

### 湖仓集成（Lakehouse）
- BigQuery + BigLake-managed Iceberg 湖仓
- 自动同步状态监控
- 报表和概览

## 技术架构

| 组件 | 技术选型 |
|------|---------|
| 前端 | React 19 + AntD 5 + Vite 6 |
| 后端 | Go 1.25 + Gin |
| SDK | Python 3.10+ (httpx + pydantic) |
| 存储 | PostgreSQL |
| 检索 | Elasticsearch |
| 湖仓 | BigQuery / BigLake Iceberg |
| 部署 | Cloudflare Workers + Assets |
