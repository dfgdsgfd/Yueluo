# Local development: keep browser requests on the Next.js origin and let
# next.config.ts rewrite /api/* to the remote backend.
BACKEND_ORIGIN=https://xse.yuelk.com
NEXT_PUBLIC_BACKEND_ORIGIN=https://xse.yuelk.com

# Optional: use when browser code must call the backend origin directly instead
# of the Next.js /api rewrite. This also affects full-page auth redirects such
# as the user-center one-click login button.
NEXT_PUBLIC_API_BASE_URL=https://xse.yuelk.com

# Optional: backend Swagger docs path. Must match backend SWAGGER_DOCS_PATH.
NEXT_PUBLIC_SWAGGER_DOCS_PATH=swagger-MYQD6LuH0heYgcK5DT10Al00dj6OW8Wc

# Optional: private page entries. These are server-only values and must start
# with "/". When customized, the default /admin or /backend-api URL returns 404.
# Keep both paths distinct and do not use /api or /_next prefixes.
ADMIN_ENTRY_PATH=/admin
BACKEND_API_ENTRY_PATH=/api-docs-91c23wI2WAMCwHItDJZqZBJeu5kOTmtYcfHb


# Optional: server-side requests can use a different internal backend origin.
# Falls back to BACKEND_ORIGIN when omitted.
API_BASE_URL=https://xse.yuelk.com/api

# Optional: comma-separated origins allowed to load Next.js dev assets from this
# dev server. Useful when testing the local app through a mapped domain.
# NEXT_ALLOWED_DEV_ORIGINS=cs.yuelk.com,xse.yuelk.com

# See ../frontend-backend-api-integration-env.md for the full integration
# environment matrix, smoke credentials, and write-smoke safety notes.

# Optional: keep home/explore pages renderable when a guest request is rejected.
# This does not fabricate feed data; it only prevents unauthenticated preview
# sessions from being redirected to /login while inspecting the page shell.
# Set both keys so server rendering and browser-side feed actions agree.
# ALLOW_GUEST_PAGE_PREVIEW=true
# NEXT_PUBLIC_ALLOW_GUEST_PAGE_PREVIEW=true

# Optional: render typed fixture feed data when the backend is unavailable.
# Keep this disabled for real integration so API failures and guest restrictions
# remain visible.
FEED_FIXTURE_FALLBACK=true

# Optional: HTTP timeout for scripts/check-integration-readiness.mjs.
# INTEGRATION_HTTP_TIMEOUT_MS=5000
# INTEGRATION_HTTP_RETRY_COUNT=1

# Optional: enable write smoke checks. This can temporarily follow/unfollow between
# user A and user B, then attempts to restore the initial follow state. It also
# creates/reuses an IM conversation and sends one uniquely tagged smoke message.
# With user A credentials, it creates one draft post and then attempts to delete
# it. Draft creation should not notify followers, but it still writes post rows.
# If user A already has at least one withdraw payment-code URL, it temporarily
# replaces both payment-code URLs with smoke URLs, then attempts to restore them.
# With admin credentials, it temporarily flips AI review and guest-access runtime
# toggles, then attempts to restore their initial values.
# INTEGRATION_ENABLE_WRITE_SMOKE=false
# Required when write smoke should exercise post like/collect/comment mutations.
# Use a dedicated test post because creating a comment may notify the post owner
# even though the smoke deletes the comment afterward.
# INTEGRATION_WRITE_SMOKE_POST_ID=

# Optional: real integration smoke accounts for scripts/check-integration-readiness.mjs.
# Prefer short-lived tokens when available; otherwise provide username/password pairs.
# INTEGRATION_USER_A_ACCESS_TOKEN=
# INTEGRATION_USER_A_ID=
# INTEGRATION_USER_A_PASSWORD=
# INTEGRATION_USER_B_ACCESS_TOKEN=
# INTEGRATION_USER_B_ID=
# INTEGRATION_USER_B_PASSWORD=
# INTEGRATION_ADMIN_ACCESS_TOKEN=
# INTEGRATION_ADMIN_USERNAME=
# INTEGRATION_ADMIN_PASSWORD=


# The full-screen viewer can still browse every image the viewer may access.
# NEXT_PUBLIC_POST_SLIDESHOW_MAX_IMAGES=25

# Optional: /explore infinite scroll window.
# Retain only this many loaded feed pages while auto-loading up/down; lower
# values reduce mounted cards and lag on long sessions. Supported range: 3-40.
NEXT_PUBLIC_EXPLORE_FEED_RETAIN_PAGES=5

# Optional: /explore auto-load trigger distance in pixels for both top and
# bottom sentinels. Supported range: 0-4000.
NEXT_PUBLIC_EXPLORE_FEED_AUTOLOAD_MARGIN_PX=900