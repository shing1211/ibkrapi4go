# ibkrapi4go

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/License-Apache%202.0-blue?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/IBKR%20Web%20API-v2.39-brightgreen?style=flat-square" alt="IBKR API Version">
  <img src="https://img.shields.io/badge/Endpoints-185-orange?style=flat-square" alt="Endpoints">
  <img src="https://img.shields.io/badge/Schemas-443-blue?style=flat-square" alt="Schemas">
  <img src="https://img.shields.io/badge/Status-pre--alpha-red?style=flat-square" alt="Status">
</p>

> **⚠️ No oficial y pre-alfa.** ibkrapi4go es un SDK de Go de la comunidad para la
> Interactive Brokers Web API. **No está afiliado a Interactive Brokers.** Está en
> desarrollo activo: los paquetes públicos aún no están implementados.
> Consulta [DISCLAIMER.md](./DISCLAIMER.md) y [docs/ROADMAP.md](./docs/ROADMAP.md).

> **Nativo de Go · Con tipos seguros · Basado en OpenAPI.** Un cliente Go idiomático
> para la Interactive Brokers Web API: gestión de cuentas, cartera, trading, datos de
> mercado y streaming WebSocket en tiempo real.

[English](./README.md) · [简体中文](./README.zh-Hans.md) · [繁體中文](./README.zh-Hant.md) · [日本語](./README.ja.md) · [한국어](./README.ko.md) · [Español](./README.es.md)

> Este documento es una traducción comunitaria del [README](./README.md) en inglés.
> **La versión en inglés es la autoritativa.** Sincronizado / Last synced: 5692cd9

## Tabla de contenidos

- [Estado](#estado)
- [Las dos API](#las-dos-api)
- [Instalación](#instalación)
- [Uso previsto](#uso-previsto)
- [Autenticación](#autenticación)
- [Estructura de paquetes](#estructura-de-paquetes)
- [Documentación del repositorio](#documentación-del-repositorio)
- [Compilación y pruebas](#compilación-y-pruebas)
- [Contribuir](#contribuir)
- [Seguridad](#seguridad)
- [Licencia](#licencia)

---

## Estado

| Elemento | Estado |
|----------|--------|
| Planificación y documentación | ✅ Completo |
| Validación de generación de código OpenAPI | ✅ Verificado ([docs/CODEGEN.md](./docs/CODEGEN.md)) |
| Código generado en `client/` | 🚧 Generado bajo demanda (`make codegen`) |
| API pública `pkg/ibkr` | 🚧 No implementada |
| Implementación `internal/` | 🚧 No implementada |
| Pruebas / ejemplos | 🚧 No implementados |

Actualmente este repositorio contiene **documentación y herramientas de generación de
código**, además de un procedimiento verificado. Todavía no existe código público del
SDK. Consulta [docs/ROADMAP.md](./docs/ROADMAP.md) para el plan de construcción.

## Las dos API

La especificación OpenAPI de IBKR (v2.39.0) describe en realidad **dos superficies de
API con dos esquemas de autenticación distintos**. No son intercambiables:

| Superficie | Ruta base | Operaciones | Autenticación |
|------------|-----------|------------:|---------------|
| Client Portal API (CPAPI) | `/v1/api/*` | 115 | `ssoBearer` |
| IB REST API | `/gw/api/v1/*`, `/gw/api/v2/*`, `/oauth2/*` | 70 | `oauth2Bearer` |
| **Total** | | **185** | |

**La v1 de este SDK solo cubre CPAPI (`ssoBearer`).** La superficie `oauth2Bearer`
se pospone a una fase posterior. Consulta [ADR 0001](./docs/adr/0001-two-api-surfaces.md)
y [ADR 0005](./docs/adr/0005-v1-scope.md).

## Instalación

```bash
go get github.com/shing1211/ibkrapi4go/pkg/ibkr
```

Requiere **Go 1.26+** y un [IBKR Client Portal Gateway](https://www.interactivebrokers.com/api/) en ejecución.

## Uso previsto

> La API siguiente es el **diseño objetivo** y **aún no está implementada**. Se
> muestra para comunicar la ergonomía prevista. Hoy no compila.

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

	// El Client Portal Gateway se autentica de forma interactiva en el navegador.
	// El SDK se comunica con un gateway local ya autenticado.
	cli, err := ibkr.NewClient(
		ibkr.WithGatewayURL("https://localhost:5000"),
		ibkr.WithInsecureSkipVerify(true), // el gateway local usa un certificado autofirmado
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

## Autenticación

El Client Portal Gateway se autentica de forma **interactiva** (inicio de sesión en
el navegador + 2FA). El SDK no acepta usuario y contraseña, y la especificación de
IBKR no define ninguna concesión de contraseña. El SDK se comunica con un **gateway
ya autenticado** y gestiona el token de sesión resultante.

Consulta [docs/AUTH.md](./docs/AUTH.md) y [docs/SESSIONS.md](./docs/SESSIONS.md).

## Estructura de paquetes

```
ibkrapi4go/
├── client/          # Tipos OpenAPI generados + cliente HTTP (NO EDITAR)
├── pkg/ibkr/        # Superficie pública del SDK (prevista)
├── internal/        # Implementación privada (prevista)
├── docs/            # Diseño, referencia, ADR
├── scripts/         # Generación de código + validación
└── specs/           # Especificación OpenAPI en caché (gitignore)
```

## Documentación del repositorio

| Documento | Contenido |
|-----------|-----------|
| [docs/SPEC.md](./docs/SPEC.md) | Índice canónico de endpoints (superficie + autenticación) |
| [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) | Capas, composición, middleware |
| [docs/ROADMAP.md](./docs/ROADMAP.md) | Plan por fases con criterios de salida |
| [docs/AUTH.md](./docs/AUTH.md) | Los dos modelos de autenticación |
| [docs/CODEGEN.md](./docs/CODEGEN.md) | Obtención, parcheo, generación y verificación |
| [docs/GLOSSARY.md](./docs/GLOSSARY.md) | Terminología |
| [docs/adr/](./docs/adr/) | Registros de decisiones de arquitectura |
| [docs/design/](./docs/design/) | Contratos de diseño por módulo |

## Compilación y pruebas

```bash
# Mostrar los objetivos disponibles
make help

# Formatear, vet, probar
make check

# Regenerar client/ desde la especificación OpenAPI
make codegen

# Verificar que la generación sea reproducible (control de deriva)
make codegen-verify
```

## Contribuir

Consulta [CONTRIBUTING.md](./CONTRIBUTING.md). Todas las confirmaciones deben estar
firmadas con DCO (`git commit -s`). Al contribuir aceptas el
[Código de Conducta](./CODE_OF_CONDUCT.md). Las traducciones siguen
[TRANSLATING.md](./TRANSLATING.md).

## Seguridad

Consulta [SECURITY.md](./SECURITY.md) y
[docs/design/08-concurrency.md](./docs/design/08-concurrency.md) para el modelo de
concurrencia y secretos.

## Licencia

[Apache License 2.0](./LICENSE). Consulta también [NOTICE](./NOTICE) y
[THIRD_PARTY_NOTICES.md](./THIRD_PARTY_NOTICES.md).
