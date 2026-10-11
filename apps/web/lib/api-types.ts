import type { components, paths } from "./api-schema";

export type Schema<K extends keyof components["schemas"]> =
  components["schemas"][K];
export type Response<
  P extends keyof paths,
  M extends keyof paths[P] = "get" & keyof paths[P],
  C extends number = 200,
> = paths[P][M] extends { responses: infer R }
  ? C extends keyof R
    ? R[C] extends { content: { "application/json": infer T } }
      ? T
      : never
    : never
  : never;
export type Api = <T = unknown>(
  path: string,
  method?: string,
  value?: unknown,
  signal?: AbortSignal,
) => Promise<T>;
