# ibkrapi4go

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/License-Apache%202.0-blue?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/IBKR%20Web%20API-v2.39-brightgreen?style=flat-square" alt="IBKR API Version">
  <img src="https://img.shields.io/badge/Endpoints-185-orange?style=flat-square" alt="Endpoints">
  <img src="https://img.shields.io/badge/Schemas-443-blue?style=flat-square" alt="Schemas">
  <img src="https://img.shields.io/badge/Status-alpha-blue?style=flat-square" alt="Status">
  <a href="https://codecov.io/gh/shing1211/ibkrapi4go"><img src="https://codecov.io/gh/shing1211/ibkrapi4go/branch/main/graph/badge.svg" alt="Coverage"></a>
  <a href="https://shing1211.github.io/ibkrapi4go/"><img src="https://img.shields.io/badge/Docs-GitHub%20Pages-97CAFF?style=flat-square&logo=github" alt="Docs"></a>
</p>

> **⚠️ 비공식 & 프리알파.** ibkrapi4go는 Interactive Brokers Web API를 위한
> 커뮤니티 Go SDK입니다. **Interactive Brokers와 아무런 제휴 관계가 없습니다.**
> 모든 185개 API 오퍼레이션이 구현되었습니다 (115 CPAPI + 70 IB REST).
> [DISCLAIMER.md](./DISCLAIMER.md)와 [docs/ROADMAP.md](./docs/ROADMAP.md)를 참고하세요.

> **Go 네이티브 · 타입 안전 · OpenAPI 기반.** Interactive Brokers Web API를 위한
> 관용적인 Go 클라이언트 — 계좌 관리, 포트폴리오, 주문, 시세 데이터, 실시간
> WebSocket 스트리밍.

[English](./README.md) · [简体中文](./README.zh-Hans.md) · [繁體中文](./README.zh-Hant.md) · [日本語](./README.ja.md) · [한국어](./README.ko.md) · [Español](./README.es.md)

> 이 문서는 영어 [README](./README.md)의 커뮤니티 번역입니다. **영어판이 정본입니다.**
> 동기화 / Last synced: b7f2b81

## 목차

- [상태](#상태)
- [두 개의 API](#두-개의-api)
- [설치](#설치)
- [사용법](#사용법)
- [인증](#인증)
- [패키지 구조](#패키지-구조)
- [저장소 문서](#저장소-문서)
- [빌드 및 테스트](#빌드-및-테스트)
- [기여](#기여)
- [보안](#보안)
- [라이선스](#라이선스)

---

## 상태

| 항목 | 상태 |
|------|------|
| 기획 및 문서 | ✅ 완료 |
| OpenAPI 코드 생성 검증 | ✅ 검증됨 ([docs/CODEGEN.md](./docs/CODEGEN.md)) |
| `client/` 생성 코드 | ✅ 커밋됨 (생성) |
| `pkg/ibkr` 공개 API | ✅ 구현됨 (185/185 오퍼레이션) |
| `internal/` 구현 | ✅ 구현됨 |
| 테스트 / 예제 | ✅ 구현됨 |
| 모의 게이트웨이 (185/185 연산) | ✅ 구현됨 ([docs/MOCK-GATEWAY.md](./docs/MOCK-GATEWAY.md)) |
| 벤치마크 + 퍼즈 테스트 | ✅ 구현됨 ([docs/OBSERVABILITY.md](./docs/OBSERVABILITY.md)) |
| 메트릭 + 로깅 | ✅ 구현됨 ([docs/OBSERVABILITY.md](./docs/OBSERVABILITY.md)) |
| CI 품질 게이트 (커버리지, 프리컴밋) | ✅ 구현됨 |
| 문서 웹사이트 + Discussions | ✅ 구현됨 ([docs/ROADMAP.md](./docs/ROADMAP.md)) |
| 릴리스 | ✅ v0.1.1 (GitHub + Gitee) |

현재 저장소에는 **생성된 OpenAPI 클라이언트**, 문서, 코드 생성 도구가 포함되어
있습니다. 모든 185개 API 오퍼레이션이 구현되었습니다. 계획은
[docs/ROADMAP.md](./docs/ROADMAP.md)를 참고하세요.

## 두 개의 API

IBKR OpenAPI 명세(v2.39.0)는 실제로 **서로 다른 인증 방식을 가진 두 개의 API
표면**을 기술합니다. 두 표면은 호환되지 않습니다.

| 표면 | 기본 경로 | 오퍼레이션 수 | 인증 |
|------|-----------|-------------:|------|
| Client Portal API (CPAPI) | `/v1/api/*` | 115 | `ssoBearer` |
| IB REST API | `/gw/api/v1/*`, `/gw/api/v2/*`, `/oauth2/*` | 70 | `oauth2Bearer` |
| **합계** | | **185** | |

SDK는 처음에 CPAPI(`ssoBearer`)만 대상으로 했으며, `oauth2Bearer` 표면은
단계 5-6에서 구현되었습니다. [ADR 0001](./docs/adr/0001-two-api-surfaces.md)과
[ADR 0011](./docs/adr/0011-oauth2-surface.md)를 참고하세요.

## 설치

```bash
go get github.com/shing1211/ibkrapi4go/pkg/ibkr
```

**Go 1.26+** 와 실행 중인 [IBKR Client Portal Gateway](https://www.interactivebrokers.com/api/)가 필요합니다.

## 사용법

최소한의 Client Portal API (CPAPI) 예제:

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

	// Client Portal Gateway는 브라우저에서 대화형으로 인증됩니다.
	// SDK는 이미 인증된 로컬 게이트웨이와 통신합니다.
	cli, err := ibkr.NewClient(
		ibkr.WithGatewayURL("https://localhost:5000"),
		ibkr.WithInsecureSkipVerify(true), // 로컬 게이트웨이는 자체 서명 인증서 사용
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

## 인증

Client Portal Gateway는 **대화형**으로 인증됩니다(브라우저 로그인 + 2FA). SDK는
사용자 이름과 비밀번호를 받지 않으며, IBKR 명세에도 비밀번호 그랜트가 정의되어
있지 않습니다. SDK는 **이미 인증된 게이트웨이**와 통신하고 그 결과 생성된 세션
토큰을 관리합니다.

[docs/AUTH.md](./docs/AUTH.md)와 [docs/SESSIONS.md](./docs/SESSIONS.md)를 참고하세요.

## 패키지 구조

```
ibkrapi4go/
├── client/          # 생성된 OpenAPI 타입 + HTTP 클라이언트 (편집 금지)
├── pkg/ibkr/        # 공개 SDK 표면
├── internal/        # 내부 구현
├── cmd/             # 독립 실행 파일 (ibkr-mock-gateway)
├── examples/        # 실행 가능한 예제 (mock)
├── docs/            # 설계, 레퍼런스, ADR
├── scripts/         # 코드 생성 + 검증
└── specs/           # 캐시된 OpenAPI 명세 (gitignore)
```

## 저장소 문서

| 문서 | 내용 |
|------|------|
| [docs/SPEC.md](./docs/SPEC.md) | 정본 엔드포인트 색인 (표면 + 인증) |
| [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) | 계층, 구성, 미들웨어 |
| [docs/ROADMAP.md](./docs/ROADMAP.md) | 종료 기준이 있는 단계별 계획 |
| [docs/MOCK-GATEWAY.md](./docs/MOCK-GATEWAY.md) | 저장소 내 모의 게이트웨이 (테스트, 예제, 독립 실행 파일) |
| [docs/AUTH.md](./docs/AUTH.md) | 두 가지 인증 모델 |
| [docs/CODEGEN.md](./docs/CODEGEN.md) | 명세 가져오기, 패치, 생성, 검증 |
| [docs/OBSERVABILITY.md](./docs/OBSERVABILITY.md) | 메트릭, 벤치마크, 퍼즈 테스트, 로깅 |
| [docs/TESTING.md](./docs/TESTING.md) | 테스트 전략 및 티어 |
| [docs/GLOSSARY.md](./docs/GLOSSARY.md) | 용어집 |
| [docs/adr/](./docs/adr/) | 아키텍처 결정 기록 |
| [docs/design/](./docs/design/) | 모듈별 설계 계약 |

## 빌드 및 테스트

```bash
# 사용 가능한 타깃 보기
make help

# 포맷, vet, 테스트
make check

# OpenAPI 명세로 client/ 재생성
make codegen

# 코드 생성 재현성 검증 (드리프트 검사)
make codegen-verify
```

## 기여

[CONTRIBUTING.md](./CONTRIBUTING.md)를 참고하세요. 모든 커밋은 DCO 서명
(`git commit -s`)이 필요합니다. 참여하면 [Code of Conduct](./CODE_OF_CONDUCT.md)에
동의한 것으로 간주됩니다. 번역은 [TRANSLATING.md](./TRANSLATING.md)를 따릅니다.

## 보안

동시성과 시크릿 모델은 [SECURITY.md](./SECURITY.md)와
[docs/design/08-concurrency.md](./docs/design/08-concurrency.md)를 참고하세요.

## 라이선스

[Apache License 2.0](./LICENSE). [NOTICE](./NOTICE)와
[THIRD_PARTY_NOTICES.md](./THIRD_PARTY_NOTICES.md)도 참고하세요.
