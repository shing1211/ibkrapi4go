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

> **⚠️ 非公式 & プレアルファ版。** ibkrapi4go は Interactive Brokers Web API 向けの
> コミュニティ製 Go SDK です。**Interactive Brokers とは一切関係ありません。**
> すべての 185 の API オペレーションが実装済みです（115 CPAPI + 70 IB REST）。
> [DISCLAIMER.md](./DISCLAIMER.md) と [docs/ROADMAP.md](./docs/ROADMAP.md) をご覧ください。

> **Go ネイティブ · 型安全 · OpenAPI 駆動。** Interactive Brokers Web API 向けの
> 慣用的な Go クライアント — アカウント管理、ポートフォリオ、取引、マーケットデータ、
> リアルタイム WebSocket ストリーミング。

[English](./README.md) · [简体中文](./README.zh-Hans.md) · [繁體中文](./README.zh-Hant.md) · [日本語](./README.ja.md) · [한국어](./README.ko.md) · [Español](./README.es.md)

> 本書は英語版 [README](./README.md) のコミュニティ翻訳です。**英語版が正式です。**
> 同期 / Last synced: d49800b

## 目次

- [ステータス](#ステータス)
- [2 つの API](#2-つの-api)
- [インストール](#インストール)
- [使用例](#使用例)
- [認証](#認証)
- [パッケージ構成](#パッケージ構成)
- [リポジトリのドキュメント](#リポジトリのドキュメント)
- [ビルドとテスト](#ビルドとテスト)
- [コントリビュート](#コントリビュート)
- [セキュリティ](#セキュリティ)
- [ライセンス](#ライセンス)

---

## ステータス

| 項目 | 状態 |
|------|------|
| 計画とドキュメント | ✅ 完了 |
| OpenAPI コード生成の検証 | ✅ 検証済み（[docs/CODEGEN.md](./docs/CODEGEN.md)） |
| `client/` 生成コード | ✅ コミット済み（生成） |
| `pkg/ibkr` 公開 API | ✅ 実装済み（185/185 オペレーション） |
| `internal/` 実装 | ✅ 実装済み |
| テスト / サンプル | ✅ 実装済み |
| モックゲートウェイ（185/185 オペレーション） | ✅ 実装済み（[docs/MOCK-GATEWAY.md](./docs/MOCK-GATEWAY.md)） |
| ベンチマーク + ファズテスト | ✅ 実装済み（[docs/OBSERVABILITY.md](./docs/OBSERVABILITY.md)） |
| メトリクス + ロギング | ✅ 実装済み（[docs/OBSERVABILITY.md](./docs/OBSERVABILITY.md)） |
| CI 品質ゲート（カバレッジ、プレコミット） | ✅ 実装済み |
| ドキュメントサイト + Discussions | ✅ 実装済み（[docs/ROADMAP.md](./docs/ROADMAP.md)） |
| リリース | ✅ v1.0.4（GitHub + Gitee） |

現在のリポジトリには**生成済みの OpenAPI クライアント**、ドキュメント、
コード生成ツールが含まれます。すべての 185 の API オペレーションが実装済みです。
計画は [docs/ROADMAP.md](./docs/ROADMAP.md) を参照してください。

## 2 つの API

IBKR OpenAPI 仕様（v2.40.0）は、実際には**異なる認証方式を持つ 2 つの API
サーフェス**を記述しています。両者は互換ではありません。

| サーフェス | ベースパス | オペレーション数 | 認証 |
|------------|-----------|----------------:|------|
| Client Portal API (CPAPI) | `/v1/api/*` | 115 | `ssoBearer` |
| IB REST API | `/gw/api/v1/*`、`/gw/api/v2/*`、`/oauth2/*` | 70 | `oauth2Bearer` |
| **合計** | | **185** | |

SDK は当初 CPAPI（`ssoBearer`）のみを対象としていました。`oauth2Bearer`
サーフェスはフェーズ 5-6 で実装されました。[ADR 0001](./docs/adr/0001-two-api-surfaces.md)
と [ADR 0011](./docs/adr/0011-oauth2-surface.md) を参照してください。

## インストール

```bash
go get github.com/shing1211/ibkrapi4go/pkg/ibkr
```

**Go 1.26+** と、稼働中の [IBKR Client Portal Gateway](https://www.interactivebrokers.com/api/) が必要です。

## 使用例

最小限の Client Portal API (CPAPI) の例：

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

	// Client Portal Gateway はブラウザで対話的に認証されます。
	// SDK は認証済みのローカルゲートウェイと通信します。
	cli, err := ibkr.NewClient(
		ibkr.WithGatewayURL("https://localhost:5000"),
		ibkr.WithInsecureSkipVerify(true), // ローカルゲートウェイは自己署名証明書を使用
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

## 認証

Client Portal Gateway は**対話的に**認証されます（ブラウザログイン + 2FA）。SDK は
ユーザー名とパスワードを受け取らず、IBKR 仕様にもパスワードグラントは定義されて
いません。SDK は**認証済みのゲートウェイ**と通信し、結果として得られるセッション
トークンを管理します。

[docs/AUTH.md](./docs/AUTH.md) と [docs/SESSIONS.md](./docs/SESSIONS.md) を参照してください。

## パッケージ構成

```
ibkrapi4go/
├── client/          # 生成された OpenAPI 型 + HTTP クライアント（編集禁止）
├── pkg/ibkr/        # 公開 SDK サーフェス
├── internal/        # 内部実装
├── cmd/             # 単体バイナリ（ibkr-mock-gateway）
├── examples/        # 実行可能なサンプル（mock）
├── docs/            # 設計、リファレンス、ADR
├── scripts/         # コード生成 + 検証
└── specs/           # キャッシュされた OpenAPI 仕様（gitignore 済み）
```

## 全体の流れ

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

すべてのリクエストはゲートウェイに到達する前にトランスポートチェーンを通ります：

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

詳細なレイヤリングは [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) を参照。

## リポジトリのドキュメント

| ドキュメント | 内容 |
|--------------|------|
| [docs/SPEC.md](./docs/SPEC.md) | 正式なエンドポイント索引（サーフェス + 認証） |
| [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) | レイヤリング、構成、ミドルウェア |
| [docs/ROADMAP.md](./docs/ROADMAP.md) | 終了条件付きの段階的計画 |
| [docs/MOCK-GATEWAY.md](./docs/MOCK-GATEWAY.md) | リポジトリ内モックゲートウェイ（テスト、サンプル、単体バイナリ） |
| [docs/GATEWAY-SETUP.md](./docs/GATEWAY-SETUP.md) | Client Portal Gateway の実行と認証 |
| [docs/PERMISSIONS.md](./docs/PERMISSIONS.md) | 取引権限、マーケットデータ権利、遅延データ |
| [docs/AUTH.md](./docs/AUTH.md) | 2 つの認証モデル |
| [docs/CODEGEN.md](./docs/CODEGEN.md) | 仕様の取得、パッチ、生成、検証 |
| [docs/OBSERVABILITY.md](./docs/OBSERVABILITY.md) | メトリクス、ベンチマーク、ファズテスト、ロギング |
| [docs/TESTING.md](./docs/TESTING.md) | テスト戦略とティア |
| [docs/GLOSSARY.md](./docs/GLOSSARY.md) | 用語集 |
| [docs/adr/](./docs/adr/) | アーキテクチャ決定記録 |
| [docs/design/](./docs/design/) | モジュールごとの設計契約 |

## ビルドとテスト

```bash
# 利用可能なターゲットを表示
make help

# フォーマット、vet、テスト
make check

# OpenAPI 仕様から client/ を再生成
make codegen

# コード生成が再現可能か検証（ドリフト検査）
make codegen-verify
```

## コントリビュート

[CONTRIBUTING.md](./CONTRIBUTING.md) を参照してください。すべてのコミットは DCO
署名（`git commit -s`）が必要です。参加により
[Code of Conduct](./CODE_OF_CONDUCT.md) に同意したものとみなされます。
翻訳は [TRANSLATING.md](./TRANSLATING.md) に従います。

## セキュリティ

並行性とシークレットのモデルについては [SECURITY.md](./SECURITY.md) と
[docs/design/08-concurrency.md](./docs/design/08-concurrency.md) を参照してください。

## ライセンス

[Apache License 2.0](./LICENSE)。[NOTICE](./NOTICE) と
[THIRD_PARTY_NOTICES.md](./THIRD_PARTY_NOTICES.md) も参照してください。
