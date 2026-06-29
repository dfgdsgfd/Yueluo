import type { NextRequest } from "next/server";
import { createNetworkServerSnapshot } from "@/lib/network-diagnostics";

export async function GET(request: NextRequest) {
  return Response.json(createNetworkServerSnapshot(request.headers, request.url), {
    headers: {
      "Cache-Control": "no-store",
    },
  });
}
