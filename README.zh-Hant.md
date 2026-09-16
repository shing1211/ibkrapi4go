# ibkrapi4go

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/License-Apache%202.0-blue?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/IBKR%20Web%20API-v2.39-brightgreen?style=flat-square" alt="IBKR API Version">
  <img src="https://img.shields.io/badge/Endpoints-185-orange?style=flat-square" alt="Endpoints">
  <img src="https://img.shields.io/badge/Schemas-443-blue?style=flat-square" alt="Schemas">
  <img src="https://img.shields.io/badge/Status-pre--alpha-red?style=flat-square" alt="Status">
</p>

> **⚠️ 非官方 & 早期預覽版。** ibkrapi4go 是社群維護的 Interactive Brokers
> Web API Go SDK，**與 Interactive Brokers 無任何隸屬關係**。專案仍在開發中，
> 公開套件尚未實作。請閱讀 [DISCLAIMER.md](./DISCLAIMER.md) 與
> [docs/ROADMAP.md](./docs/ROADMAP.md)。

> **Go 原生 · 型別安全 · OpenAPI 驅動。** 面向 Interactive Brokers Web API 的
> 慣用 Go 用戶端 —— 帳戶管理、投資組合、交易、行情資料與即時 WebSocket 串流。

[English](./README.md) · [简体中文](./README.zh-Hans.md) · [繁體中文](./README.zh-Hant.md) · [日本語](./README.ja.md) · [한국어](./README.ko.md) · [Español](./README.es.md)

> 本文件是英文 [README](./README.md) 的社群翻譯。**英文版本為準。**
> 同步於 / Last synced: 5692cd9

## 目錄

- [狀態](#狀態)
- [兩套 API](#兩套-api)
- [安裝](#安裝)
- [規劃用法](#規劃用法)
- [認證](#認證)
- [套件結構](#套件結構)
- [倉庫文件](#倉庫文件)
- [建置與測試](#建置與測試)
- [貢獻](#貢獻)
- [安全性](#安全性)
- [授權條款](#授權條款)

---

## 狀態

| 項目 | 狀態 |
|------|------|
| 規劃與文件 | ✅ 完成 |
| OpenAPI 程式碼產生驗證 | ✅ 已驗證（見 [docs/CODEGEN.md](./docs/CODEGEN.md)） |
| `client/` 產生程式碼 | ✅ 已提交（產生） |
| `pkg/ibkr` 公開 API | 🚧 未實作 |
| `internal/` 實作 | 🚧 未實作 |
| 測試 / 範例 | 🚧 未實作 |

目前倉庫包含**產生的 OpenAPI 用戶端**、文件與程式碼產生工具。尚無手寫的 SDK
程式碼（`pkg/ibkr`、`internal/`）。建置計畫見 [docs/ROADMAP.md](./docs/ROADMAP.md)。

## 兩套 API

IBKR OpenAPI 規範（v2.39.0）實際上描述了**兩套 API 表面，使用兩種不同的認證方式**，
兩者不可互換：

| 表面 | 路徑前綴 | 介面數 | 認證 |
|------|----------|-------:|------|
| Client Portal API (CPAPI) | `/v1/api/*` | 115 | `ssoBearer` |
| IB REST API | `/gw/api/v1/*`、`/gw/api/v2/*`、`/oauth2/*` | 70 | `oauth2Bearer` |
| **合計** | | **185** | |

**SDK v1 僅針對 CPAPI（`ssoBearer`）**，`oauth2Bearer` 表面延後至後續階段。
參見 [ADR 0001](./docs/adr/0001-two-api-surfaces.md) 與
[ADR 0005](./docs/adr/0005-v1-scope.md)。

## 安裝

```bash
go get github.com/shing1211/ibkrapi4go/pkg/ibkr
```

需要 **Go 1.26+**，以及正在執行的 [IBKR Client Portal Gateway](https://www.interactivebrokers.com/api/)。

## 規劃用法

> 以下 API 為**目標設計**，**尚未實作**，僅用於說明預期的使用體驗，目前無法編譯。

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

	// Client Portal Gateway 透過瀏覽器互動式認證。
	// SDK 與已完成認證的本機閘道通訊。
	cli, err := ibkr.NewClient(
		ibkr.WithGatewayURL("https://localhost:5000"),
		ibkr.WithInsecureSkipVerify(true), // 本機閘道使用自簽憑證
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

## 認證

Client Portal Gateway 透過**互動式**方式認證（瀏覽器登入 + 2FA）。SDK 不接受
使用者名稱/密碼，IBKR 規範也未定義密碼授權。SDK 與**已完成認證的閘道**通訊，
並管理由此產生的工作階段權杖。

參見 [docs/AUTH.md](./docs/AUTH.md) 與 [docs/SESSIONS.md](./docs/SESSIONS.md)。

## 套件結構

```
ibkrapi4go/
├── client/          # 產生的 OpenAPI 型別 + HTTP 用戶端（請勿編輯）
├── pkg/ibkr/        # 公開 SDK 表面（規劃中）
├── internal/        # 私有實作（規劃中）
├── docs/            # 設計、參考、ADR
├── scripts/         # 程式碼產生 + 驗證
└── specs/           # 快取的 OpenAPI 規範（已 gitignore）
```

## 倉庫文件

| 文件 | 內容 |
|------|------|
| [docs/SPEC.md](./docs/SPEC.md) | 規範介面索引（表面 + 認證） |
| [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) | 分層、組合、中介軟體 |
| [docs/ROADMAP.md](./docs/ROADMAP.md) | 含退出標準的分階段計畫 |
| [docs/AUTH.md](./docs/AUTH.md) | 兩種認證模型 |
| [docs/CODEGEN.md](./docs/CODEGEN.md) | 規範取得、修補、產生、驗證 |
| [docs/GLOSSARY.md](./docs/GLOSSARY.md) | 術語表 |
| [docs/adr/](./docs/adr/) | 架構決策記錄 |
| [docs/design/](./docs/design/) | 各模組設計契約 |

## 建置與測試

```bash
# 查看可用目標
make help

# 格式化、靜態檢查、測試
make check

# 由 OpenAPI 規範重新產生 client/
make codegen

# 驗證程式碼產生是否可重現（漂移檢查）
make codegen-verify
```

## 貢獻

參見 [CONTRIBUTING.md](./CONTRIBUTING.md)。所有提交必須通過 DCO 簽署
（`git commit -s`）。參與即表示你同意 [Code of Conduct](./CODE_OF_CONDUCT.md)。
翻譯請遵循 [TRANSLATING.md](./TRANSLATING.md)。

## 安全性

並行與密鑰模型參見 [SECURITY.md](./SECURITY.md) 與
[docs/design/08-concurrency.md](./docs/design/08-concurrency.md)。

## 授權條款

[Apache License 2.0](./LICENSE)。另見 [NOTICE](./NOTICE) 與
[THIRD_PARTY_NOTICES.md](./THIRD_PARTY_NOTICES.md)。
