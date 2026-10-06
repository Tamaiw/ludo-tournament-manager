# Hexagonal (ports & adapters) backend architecture

The Go backend follows hexagonal architecture: the application core (`internal/core/{domain,ports,services}`) is independent of every external concern (HTTP, SQLite, SMTP, …); concrete adapters under `internal/adapters/` implement the ports; `cmd/server/main.go` is the composition root that wires adapters to ports. Domain knows no other layer; services depend on ports and domain; adapters depend on the core. Hexagonal was chosen because it makes the application core unit-testable in isolation (mock adapters replace real I/O) and constrains business-logic leakage outward, supporting the user's stated fallback to a future JS-app + Go-JSON-API split.

Full architecture writeup: [`../architecture/hexagonal.md`](../architecture/hexagonal.md).