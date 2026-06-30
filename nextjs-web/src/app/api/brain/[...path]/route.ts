import { NextRequest } from "next/server";
import { auth } from "@/auth";
import { env } from "@/lib/env";

// Thin server-side proxy: forwards /api/brain/v1/* to the Go backend, attaching the
// current user's Go JWT. The token never reaches the browser; Go enforces clinic
// scoping. Only /v1/* paths are allowed.
async function proxy(req: NextRequest, path: string[]): Promise<Response> {
  if (path[0] !== "v1") {
    return Response.json({ error: "not_found" }, { status: 404 });
  }
  const session = await auth();
  if (!session?.brainToken) {
    return Response.json({ error: "unauthorized" }, { status: 401 });
  }
  const search = new URL(req.url).search;
  const target = `${env.BRAIN_API_URL}/${path.join("/")}${search}`;
  const init: RequestInit = {
    method: req.method,
    headers: {
      "content-type": "application/json",
      Authorization: `Bearer ${session.brainToken}`,
    },
    cache: "no-store",
  };
  if (req.method === "POST") init.body = await req.text();
  const res = await fetch(target, init);
  return new Response(await res.text(), {
    status: res.status,
    headers: { "content-type": "application/json" },
  });
}

export async function GET(
  req: NextRequest,
  ctx: { params: Promise<{ path: string[] }> },
) {
  return proxy(req, (await ctx.params).path);
}

export async function POST(
  req: NextRequest,
  ctx: { params: Promise<{ path: string[] }> },
) {
  return proxy(req, (await ctx.params).path);
}

export async function DELETE(
  req: NextRequest,
  ctx: { params: Promise<{ path: string[] }> },
) {
  return proxy(req, (await ctx.params).path);
}
