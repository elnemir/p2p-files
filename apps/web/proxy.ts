import { NextResponse, type NextRequest } from "next/server";
export function proxy(request: NextRequest) {
 const nonce = Buffer.from(crypto.randomUUID()).toString("base64");
 const csp = `default-src 'self'; script-src 'self' 'nonce-${nonce}' 'strict-dynamic'${process.env.NODE_ENV === "development" ? " 'unsafe-eval'" : ""}; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; font-src 'self'; object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'`;
 const headers = new Headers(request.headers); headers.set("x-nonce", nonce); headers.set("Content-Security-Policy", csp);
 const response = NextResponse.next({ request: { headers } }); response.headers.set("Content-Security-Policy", csp); return response;
}
export const config = { matcher: ["/((?!api|_next/static|_next/image|favicon.ico).*)"] };
