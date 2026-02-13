# API v1 Contract (Frozen)

## Public Pay Response

```json
{
  "requestId": "uuid",
  "chainId": "eip155:84532",
  "amount": "1000000",
  "decimals": 6,
  "walletAddress": "0x...",
  "status": "PENDING",
  "expiresAt": "2026-02-13T10:00:00Z",
  "contractAddress": "0x...",
  "txData": {
    "to": "0x...",
    "hex": "0x...",
    "programId": "",
    "base58": ""
  }
}
```

## Conventions

- External API `chainId`: CAIP-2 string (example: `eip155:84532`, `solana:devnet`).
- Internal DB FK `chain_id`: UUID.
- `paymentRequests` list key uses camelCase plural.
- status enum uses uppercase: `PENDING`, `COMPLETED`, `EXPIRED`, `CANCELLED`.
