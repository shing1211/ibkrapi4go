# Design 02 — Client

`Client` is the composition root. It owns configuration, the transport, the
session, and the managers.

## Construction

```go
func NewClient(opts ...Option) (*Client, error)
```

- Options are functional (`WithGatewayURL`, `WithRequestTimeout`, ...).
- Environment variables fill gaps when options are absent ([../CONFIG.md](../CONFIG.md)).
- `NewClient` validates configuration and returns `*ibkr.ConfigError` on bad input.
- `NewClient` does **not** perform I/O and does **not** authenticate.

## Structure

```go
type Client struct {
    cfg        Config
    http       *http.Client      // transport chain
    session    *Session          // state machine + tickle
    generated  *client.ClientWithResponses
    // CPAPI managers
    sessionManager, accountManager, portfolioManager, tradeManager,
    marketDataManager, tradingAccountManager, alertManager, forecastManager,
    scannerManager, allocationManager, modelManager, fyiManager,
    oauthManager, watchlistManager, performanceManager
    rest       *RESTSurface      // lazily dialed (oauth2Bearer)
    ws         *wsConn           // lazily dialed
}
```

## Accessors

```go
func (c *Client) Session() *SessionManager
func (c *Client) Account() *AccountManager
func (c *Client) Portfolio() *PortfolioManager
func (c *Client) Trade() *TradeManager
func (c *Client) MarketData() *MarketDataManager
func (c *Client) TradingAccount() *TradingAccountManager
func (c *Client) Alert() *AlertManager
func (c *Client) Forecast() *ForecastManager
func (c *Client) Scanner() *ScannerManager
func (c *Client) Allocation() *AllocationManager
func (c *Client) Model() *ModelManager
func (c *Client) FYI() *FYIManager
func (c *Client) OAuth() *OAuthManager
func (c *Client) Watchlist() *WatchlistManager
func (c *Client) Performance() *PerformanceManager
func (c *Client) REST() (*RESTSurface, error)   // IB REST (oauth2Bearer)
```

Managers are created once and are safe to reuse.

## Lifecycle

1. `NewClient` — validate, build transport, construct managers.
2. `Session().Initialize(ctx)` — start the session and tickle loop.
3. Manager calls.
4. `Client.Close()` — cancel context, stop tickle, best-effort logout, close ws.

## Close semantics

```go
func (c *Client) Close() error
```

- Idempotent (second call returns nil).
- Stops background goroutines; no leaks (verified with `goleak`).
- After `Close`, manager methods return `ErrClosed`.

## Concurrency

`Client` and all managers are safe for concurrent use. See
[08-concurrency.md](./08-concurrency.md).

## Options (initial set)

```go
WithGatewayURL(string)
WithInsecureSkipVerify(bool)
WithRequestTimeout(time.Duration)
WithTickleInterval(time.Duration)
WithRateLimit(rps float64, burst int)
WithGlobalRateLimit(rps float64)
WithHTTPClient(*http.Client)
WithLogger(*slog.Logger)
WithUserAgent(string)
```
