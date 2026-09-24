# ibkrapi4go

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/License-Apache%202.0-blue?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/IBKR%20Web%20API-v2.40-brightgreen?style=flat-square" alt="IBKR API Version">
  <img src="https://img.shields.io/badge/Endpoints-185-orange?style=flat-square" alt="Endpoints">
  <img src="https://img.shields.io/badge/Schemas-443-blue?style=flat-square" alt="Schemas">
  <img src="https://img.shields.io/badge/Status-stable-brightgreen?style=flat-square" alt="Status">
  <a href="https://codecov.io/gh/shing1211/ibkrapi4go"><img src="https://codecov.io/gh/shing1211/ibkrapi4go/branch/main/graph/badge.svg" alt="Coverage"></a>
  <a href="https://shing1211.github.io/ibkrapi4go/"><img src="https://img.shields.io/badge/Docs-GitHub%20Pages-97CAFF?style=flat-square&logo=github" alt="Docs"></a>
</p>

> **⚠️ 非官方。** ibkrapi4go 是社区维护的 Interactive Brokers
> Web API Go SDK，**与 Interactive Brokers 无任何隶属关系**。全部 185 个 API
> 操作已实现（115 CPAPI + 70 IB REST）。请阅读 [DISCLAIMER.md](./DISCLAIMER.md) 与
> [docs/ROADMAP.md](./docs/ROADMAP.md)。

> **Go 原生 · 类型安全 · OpenAPI 驱动。** 面向 Interactive Brokers Web API 的
> 惯用 Go 客户端 —— 账户管理、投资组合、交易、行情数据与实时 WebSocket 流。

[English](./README.md) · [简体中文](./README.zh-Hans.md) · [繁體中文](./README.zh-Hant.md) · [日本語](./README.ja.md) · [한국어](./README.ko.md) · [Español](./README.es.md)

> 本文件是英文 [README](./README.md) 的社区翻译。**英文版本为准。**
> 同步于 / Last synced: eb36ba2

## 目录

- [状态](#状态)
- [两套 API](#两套-api)
- [安装](#安装)
- [用法](#用法)
- [认证](#认证)
- [包结构](#包结构)
- [仓库文档](#仓库文档)
- [构建与测试](#构建与测试)
- [贡献](#贡献)
- [安全](#安全)
- [许可证](#许可证)

---

## 状态

| 项目 | 状态 |
|------|------|
| 规划与文档 | ✅ 完成 |
| OpenAPI 代码生成验证 | ✅ 已验证（见 [docs/CODEGEN.md](./docs/CODEGEN.md)） |
| `client/` 生成代码 | ✅ 已提交（生成） |
| `pkg/ibkr` 公开 API | ✅ 已实现（185/185 操作） |
| `internal/` 实现 | ✅ 已实现 |
| 测试 / 示例 | ✅ 已实现 |
| 模拟网关（185/185 操作） | ✅ 已实现（[docs/MOCK-GATEWAY.md](./docs/MOCK-GATEWAY.md)） |
| 基准测试 + 模糊测试 | ✅ 已实现（[docs/OBSERVABILITY.md](./docs/OBSERVABILITY.md)） |
| 指标 + 日志 | ✅ 已实现（[docs/OBSERVABILITY.md](./docs/OBSERVABILITY.md)） |
| CI 质量门禁（覆盖率、预提交） | ✅ 已实现 |
| 文档网站 + Discussions | ✅ 已实现（[docs/ROADMAP.md](./docs/ROADMAP.md)） |
| 发布 | ✅ v1.0.5（GitHub + Gitee） |

当前仓库包含**生成的 OpenAPI 客户端**、文档与代码生成工具。全部 185 个 API
操作已实现。构建计划见 [docs/ROADMAP.md](./docs/ROADMAP.md)。

## 两套 API

IBKR OpenAPI 规范（v2.40.0）实际描述了**两套 API 表面，使用两种不同的认证方式**，
二者不可互换：

| 表面 | 路径前缀 | 接口数 | 认证 |
|------|----------|-------:|------|
| Client Portal API (CPAPI) | `/v1/api/*` | 115 | `ssoBearer` |
| IB REST API | `/gw/api/v1/*`、`/gw/api/v2/*`、`/oauth2/*` | 70 | `oauth2Bearer` |
| **合计** | | **185** | |

SDK 最初仅覆盖 CPAPI（`ssoBearer`）；`oauth2Bearer` 表面已在第 5-6 阶段实现。
参见 [ADR 0001](./docs/adr/0001-two-api-surfaces.md) 与
[ADR 0011](./docs/adr/0011-oauth2-surface.md)。

## 安装

```bash
go get github.com/shing1211/ibkrapi4go/pkg/ibkr
```

需要 **Go 1.26+**，以及正在运行的 [IBKR Client Portal Gateway](https://www.interactivebrokers.com/api/)。

## 用法

一个最小的 Client Portal API (CPAPI) 示例：

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	ibkr "github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

func main() {
	ctx := context.Background()

	// Client Portal Gateway 通过浏览器交互式认证。
	// SDK 与已完成认证的本地网关通信。
	cli, err := ibkr.NewClient(
		ibkr.WithGatewayURL("https://localhost:5000"),
		ibkr.WithInsecureSkipVerify(true), // 本地网关使用自签名证书
	)
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	if err := cli.Session().Initialize(ctx); err != nil {
		log.Fatal(err)
	}

	accounts, err := cli.Account().List(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, acc := range accounts {
		fmt.Fprintf(os.Stdout, "account: %s\n", acc.AccountID)
	}
}
```

## 认证

Client Portal Gateway 通过**交互式**方式认证（浏览器登录 + 2FA）。SDK 不接受
用户名/密码，IBKR 规范也未定义密码授权。SDK 与**已完成认证的网关**通信，
并管理由此产生的会话令牌。

参见 [docs/AUTH.md](./docs/AUTH.md) 与 [docs/SESSIONS.md](./docs/SESSIONS.md)。

## 包结构

```
ibkrapi4go/
├── client/          # 生成的 OpenAPI 类型 + HTTP 客户端（请勿编辑）
├── pkg/ibkr/        # 公开 SDK 表面
├── internal/        # 私有实现
├── cmd/             # 独立二进制（ibkr-mock-gateway）
├── examples/        # 可运行示例（mock）
├── docs/            # 设计、参考、ADR
├── scripts/         # 代码生成 + 校验
└── specs/           # 缓存的 OpenAPI 规范（已 gitignore）
```

## 整体如何协作

```text
NewClient(options...)
   │
   ▼
SessionManager.Initialize(ctx)   → tickle goroutine starts
   │
   ▼
Manager calls (Account, Portfolio, Trade, MarketData)
   │
   ▼
Client.Close()                   → tickle stops, logout, ws closed
```

每个请求在到达网关之前都会经过一条传输链：

```text
Request
  → request ID + User-Agent
  → auth header injection (bearer)
  → logging + telemetry hooks
  → circuit breaker (optional)
  → retry (safe methods only; honors Retry-After)
  → per-endpoint rate limiter
  → global rate limiter
  → per-request timeout (when the caller sets none)
  → HTTP call
  → error parsing (IBKR envelope → *ibkr.Error)
Response
```

完整分层见 [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md)。

## 仓库文档

| 文档 | 内容 |
|------|------|
| [docs/SPEC.md](./docs/SPEC.md) | 规范接口索引（表面 + 认证） |
| [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) | 分层、组合、中间件 |
| [docs/ROADMAP.md](./docs/ROADMAP.md) | 带退出标准的分阶段计划 |
| [docs/MOCK-GATEWAY.md](./docs/MOCK-GATEWAY.md) | 仓库内模拟网关（测试、示例、独立二进制） |
| [docs/GATEWAY-SETUP.md](./docs/GATEWAY-SETUP.md) | 运行并认证 Client Portal Gateway |
| [docs/PERMISSIONS.md](./docs/PERMISSIONS.md) | 交易权限、行情数据授权、延迟数据 |
| [docs/AUTH.md](./docs/AUTH.md) | 两种认证模型 |
| [docs/CODEGEN.md](./docs/CODEGEN.md) | 规范获取、修补、生成、验证 |
| [docs/OBSERVABILITY.md](./docs/OBSERVABILITY.md) | 指标、基准测试、模糊测试、日志 |
| [docs/TESTING.md](./docs/TESTING.md) | 测试策略与分层 |
| [docs/GLOSSARY.md](./docs/GLOSSARY.md) | 术语表 |
| [docs/adr/](./docs/adr/) | 架构决策记录 |
| [docs/design/](./docs/design/) | 各模块设计契约 |

## 构建与测试

```bash
# 查看可用目标
make help

# 格式化、静态检查、测试
make check

# 由 OpenAPI 规范重新生成 client/
make codegen

# 校验代码生成是否可复现（漂移检查）
make codegen-verify
```

## 贡献

参见 [CONTRIBUTING.md](./CONTRIBUTING.md)。所有提交必须通过 DCO 签署
（`git commit -s`）。参与即表示你同意 [Code of Conduct](./CODE_OF_CONDUCT.md)。
翻译请遵循 [TRANSLATING.md](./TRANSLATING.md)。

## 安全

并发与密钥模型参见 [SECURITY.md](./SECURITY.md) 与
[docs/design/08-concurrency.md](./docs/design/08-concurrency.md)。

## 许可证

[Apache License 2.0](./LICENSE)。另见 [NOTICE](./NOTICE) 与
[THIRD_PARTY_NOTICES.md](./THIRD_PARTY_NOTICES.md)。
