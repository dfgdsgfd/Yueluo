# External Moon Coin and Upload Workspaces

Moon Coin is owned exclusively by the user center. The application resolves the signed-in user's numeric `oauth2_id`, reads the current balance from `GET /api/external/user`, and applies mutations through `POST /api/external/balance`. It does not cache remote balances or read a local Moon Coin column.

Required runtime configuration:

```env
BALANCE_API_URL=https://user.yuelk.com
BALANCE_API_KEY=replace-me
BALANCE_API_TIMEOUT=10s
```

The balance center is available when both `BALANCE_API_URL` and `BALANCE_API_KEY` are configured. There is no separate enable switch.

Mutations are recorded in `external_balance_transactions` and serialized by OAuth2 account. `unknown` means the remote result cannot be proven and remains locked for administrator investigation; it must not be retried automatically. Only an `applied` transaction can use the administrator compensation endpoint.

Run [`20260619_remote_moon_coin.sql`](../scripts/20260619_remote_moon_coin.sql) explicitly during deployment to remove the obsolete `user_wallet.moon_coin` column. Schema repair intentionally does not perform this destructive migration.

Temporary upload workspaces and fixed profile media use plural `uploads` paths:

```env
UPLOAD_TEMP_DIR=uploads/tmp
UPLOAD_TEMP_CLEANUP_INTERVAL=15m
UPLOAD_TEMP_RETENTION=24h
PROTECTED_PACKAGE_RETENTION=2h
AVATAR_UPLOAD_DIR=uploads/avatar
BANNER_UPLOAD_DIR=uploads/banner
```

The temporary root is created and swept at startup, then cleaned periodically. Local files are staged in a random workspace and installed atomically. Avatar and banner filenames are fixed per user and replace the previous owned file.
