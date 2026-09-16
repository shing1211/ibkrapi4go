# ibkrapi4go

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/License-Apache%202.0-blue?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/IBKR%20Web%20API-v2.39-brightgreen?style=flat-square" alt="IBKR API 版本">
  <img src="https://img.shields.io/badge/Status-pre--alpha-red?style=flat-square" alt="状态">
</p>

> **⚠️ 非官方 & 早期预览版。** ibkrapi4go 是社区维护的 Interactive Brokers
> Web API Go SDK，**与 Interactive Brokers 无任何隶属关系**。项目仍在开发中，
> 公开包尚未实现。请阅读 [DISCLAIMER.md](./DISCLAIMER.md) 与
> [docs/ROADMAP.md](./docs/ROADMAP.md)。

[English](./README.md) · [简体中文](./README.zh-CN.md)

## 状态

| 项目 | 状态 |
|------|------|
| 规划与文档 | ✅ 完成 |
| OpenAPI 代码生成验证 | ✅ 已验证（见 [docs/CODEGEN.md](./docs/CODEGEN.md)） |
| `pkg/ibkr` 公开 API | 🚧 未实现 |
| `internal/` 实现 | 🚧 未实现 |
| 测试 / 示例 | 🚧 未实现 |

当前仓库包含**文档、代码生成工具**，以及一套经过验证的代码生成方案。
尚无公开 SDK 代码。

## 两套 API

IBKR OpenAPI 规范（v2.39.0）实际描述**两套 API 表面，使用两种不同的认证方式**，
二者不可互换：

| 表面 | 路径前缀 | 接口数 | 认证 |
|------|----------|-------:|------|
| Client Portal API (CPAPI) | `/v1/api/*` | 115 | `ssoBearer` |
| IB REST API | `/gw/api/v1/*`、`/gw/api/v2/*`、`/oauth2/*` | 70 | `oauth2Bearer` |
| **合计** | | **185** | |

**SDK v1 仅针对 CPAPI（`ssoBearer`）**，`oauth2Bearer` 表面推迟到后续阶段。
参见 [ADR 0001](./docs/adr/0001-two-api-surfaces.md)。

## 安装

```bash
go get github.com/shing1211/ibkrapi4go/pkg/ibkr
```

需要 **Go 1.26+**，以及正在运行的
[IBKR Client Portal Gateway](https://www.interactivebrokers.com/api/)。

## 规划中的用法

> 以下 API 为**目标设计**，**尚未实现**，不会编译通过。

```go
cli, err := ibkr.NewClient(
	ibkr.WithGatewayURL("https://localhost:5000"),
	ibkr.WithInsecureSkipVerify(true),
)
if err != nil {
	log.Fatal(err)
}
defer cli.Close()

if err := cli.Session().Initialize(ctx); err != nil {
	log.Fatal(err)
}

accounts, err := cli.Account().List(ctx)
```

## 认证

Client Portal Gateway 通过**浏览器交互式登录**（含 2FA）完成认证。SDK 不接受
用户名/密码，IBKR 规范也未定义密码授权。SDK 与**已完成认证的本地网关**通信，
并管理由此产生的会话令牌。参见 [docs/AUTH.md](./docs/AUTH.md)。

## 构建与测试

```bash
make help           # 查看全部目标
make check          # 格式化、静态检查、测试
make codegen        # 由 OpenAPI 规范重新生成 client/
make codegen-verify # 校验代码生成是否可复现
```

## 文档

| 文档 | 内容 |
|------|------|
| [docs/SPEC.md](./docs/SPEC.md) | 规范接口索引（表面 + 认证） |
| [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) | 分层、组合、中间件 |
| [docs/ROADMAP.md](./docs/ROADMAP.md) | 分阶段实施计划 |
| [docs/AUTH.md](./docs/AUTH.md) | 两种认证模型 |
| [docs/CODEGEN.md](./docs/CODEGEN.md) | 规范获取、修补、生成、验证 |

## 贡献

参见 [CONTRIBUTING.md](./CONTRIBUTING.md)。所有提交必须通过 DCO 签署
（`git commit -s`）。

## 许可证

[Apache License 2.0](./LICENSE)。另见 [NOTICE](./NOTICE) 与
[THIRD_PARTY_NOTICES.md](./THIRD_PARTY_NOTICES.md)。
