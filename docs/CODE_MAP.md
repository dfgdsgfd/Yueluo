# Yuem code map

This map points contributors and coding agents at the smallest useful ownership
boundary. Public facade files should remain stable; implementation modules should
import from the concrete owner listed below instead of importing through a facade.

## Frontend

| Area | Public facade or route | Implementation ownership |
| --- | --- | --- |
| API client | `front-end-nextjs/src/lib/api.ts` | `front-end-nextjs/src/lib/api/` |
| Shared API types | `front-end-nextjs/src/lib/types.ts` | `front-end-nextjs/src/lib/types/` |
| Locale messages | next-intl message object | `front-end-nextjs/src/messages/<locale>/` |
| Feed and post detail | app routes and existing component exports | `src/components/feed/explore/`, `src/components/feed/post-detail/` |
| Publishing | existing desktop/mobile entry components | `src/components/publish/workbench/`, `src/components/publish/mobile-publish/` |
| Wallet and notifications | existing page component exports | colocated controller, view, formatter and leaf modules |
| Admin console | existing admin page exports | `src/components/admin/admin-page/` and `runtime-panels/` |

## Go backend

Go declarations keep their existing package-level names. File names describe
ownership but are not API boundaries.

| Area | Package | File families |
| --- | --- | --- |
| Image processing | `internal/services` | `image_processor*`, `hidden_watermark*` |
| Queue runtime | `internal/services` | `queue*` |
| Points | `internal/repositories` | `points*` |
| Configuration | `internal/config` | `config*` |
| HTTP endpoints | `internal/http/handlers` | handlers grouped by route/workflow |

Routes, route matrices, Swagger output, database models and queue payloads are
compatibility contracts and must not change during structural splits.

## Python watermark service

`blind_watermark_fastapi/watermark.py` is the compatibility facade. Internal
implementation belongs under `blind_watermark_fastapi/_watermark/`; API paths and
Pydantic wire models remain owned by the FastAPI application modules.

## Repository tooling

`scripts/check-integration-readiness.mjs` remains the stable CLI. Its supporting
modules live under `scripts/integration-readiness/`. Run
`node scripts/check-source-size-budgets.mjs` to enforce the 40KB/800-line hard
limit for hand-written production sources.
