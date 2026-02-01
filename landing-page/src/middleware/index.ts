import type { MiddlewareNext, APIContext } from "astro";
export const onRequest = async (
  { request }: APIContext,
  next: MiddlewareNext,
) => {
  const start = Date.now();
  const response = await next();
  console.log(
    JSON.stringify({
      type: "http_access",
      method: request.method,
      path: new URL(request.url).pathname,
      status: response.status,
      duration_ms: Date.now() - start,
    }),
  );
  return response;
};
