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

## Admin Crosschain Config

All endpoints below require admin auth.

### 1. Preflight Route

`GET /api/v1/admin/crosschain-config/preflight?sourceChainId=<CAIP2|UUID>&destChainId=<CAIP2|UUID>`

Response:
```json
{
  "preflight": {
    "sourceChainId": "eip155:8453",
    "destChainId": "eip155:42161",
    "defaultBridgeType": 0,
    "fallbackMode": "strict",
    "fallbackOrder": [0],
    "policyExecutable": true,
    "issues": [],
    "bridges": [
      {
        "bridgeType": 0,
        "bridgeName": "HYPERBRIDGE",
        "ready": true,
        "checks": {
          "adapterRegistered": true,
          "routeConfigured": true,
          "feeQuoteHealthy": true
        },
        "errorCode": "",
        "errorMessage": ""
      },
      {
        "bridgeType": 1,
        "bridgeName": "CCIP",
        "ready": false,
        "checks": {
          "adapterRegistered": true,
          "routeConfigured": false,
          "feeQuoteHealthy": false
        },
        "errorCode": "CCIP_NOT_CONFIGURED",
        "errorMessage": "missing chain selector or destination adapter"
      }
    ]
  }
}
```

### 2. Route Policies (DB-level)

#### List
`GET /api/v1/admin/route-policies?page=1&limit=20&sourceChainId=&destChainId=`

#### Create
`POST /api/v1/admin/route-policies`

Request:
```json
{
  "sourceChainId": "eip155:8453",
  "destChainId": "eip155:42161",
  "defaultBridgeType": 0,
  "fallbackMode": "strict",
  "fallbackOrder": [0, 1, 2]
}
```

#### Update
`PUT /api/v1/admin/route-policies/:id`

Request body same as create.

#### Delete
`DELETE /api/v1/admin/route-policies/:id`

### 3. LayerZero Configs (DB-level)

#### List
`GET /api/v1/admin/layerzero-configs?page=1&limit=20&sourceChainId=&destChainId=&activeOnly=true`

#### Create
`POST /api/v1/admin/layerzero-configs`

Request:
```json
{
  "sourceChainId": "eip155:8453",
  "destChainId": "eip155:42161",
  "dstEid": 30110,
  "peerHex": "0x000000000000000000000000bc75055bdf937353721bfba9dd1dccfd0c70b8dd",
  "optionsHex": "0x",
  "isActive": true
}
```

#### Update
`PUT /api/v1/admin/layerzero-configs/:id`

Request body same as create.

#### Delete
`DELETE /api/v1/admin/layerzero-configs/:id`

### 4. LayerZero On-chain Adapter Config

`POST /api/v1/admin/onchain-adapters/layerzero-config`

Request:
```json
{
  "sourceChainId": "eip155:8453",
  "destChainId": "eip155:42161",
  "dstEid": 30110,
  "peerHex": "0x000000000000000000000000bc75055bdf937353721bfba9dd1dccfd0c70b8dd",
  "optionsHex": "0x"
}
```

Response:
```json
{
  "adapterAddress": "0x...",
  "txHashes": ["0x..."],
  "destChainId": "eip155:42161"
}
```

### 5. Route Error Diagnostics

`GET /api/v1/admin/diagnostics/route-error/:paymentId?sourceChainId=<CAIP2|UUID>`
or
`GET /api/v1/payment-app/diagnostics/route-error/:paymentId?sourceChainId=<CAIP2|UUID>`

Response:
```json
{
  "diagnostics": {
    "sourceChainId": "eip155:8453",
    "gatewayAddress": "0x4b6e4259016Dc94b6E78221BCdEAC59F954823E8",
    "paymentIdHex": "0x6a9f...<bytes32>",
    "decoded": {
      "rawHex": "0x08c379a0...",
      "selector": "0x08c379a0",
      "name": "Error",
      "message": "route not configured for destination eip155:42161",
      "details": {
        "destChainId": "eip155:42161"
      }
    }
  }
}
```
