# Solana Balance API

REST API untuk fetch balance banyak wallet Solana dengan fitur:
- Auth API key via Mongo
- Rate limiting per IP
- Cache 10s dengan singleflight
- Panic logger ke Discord

## Run lokal
```bash
go run ./cmd/server
