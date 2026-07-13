import type { DemoStatus, LoadReport, Order } from "./evidence";

const baseURL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";
const demoToken = import.meta.env.VITE_DEMO_TOKEN;

type Envelope<T> = { data: T };

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${baseURL}${path}`, init);
  const body = await response.json();
  if (!response.ok) {
    throw new Error(body.error?.message ?? `Request failed with ${response.status}`);
  }
  return (body as Envelope<T>).data;
}

export const api = {
  status: () => request<DemoStatus>("/api/v1/demo/status"),
  orders: () => request<Order[]>("/api/v1/orders"),
  runLoad: (attempts: number) =>
    request<LoadReport>("/api/v1/demo/load-test", {
      method: "POST",
      headers: { "Content-Type": "application/json", ...(demoToken ? { "X-Demo-Token": demoToken } : {}) },
      body: JSON.stringify({ attempts }),
    }),
  buy: (idempotencyKey: string) =>
    request<{ reservationId: string; expiresAt: string }>("/api/v1/public-demo/reservations", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Idempotency-Key": idempotencyKey,
      },
      body: "{}",
    }),
  openapiURL: `${baseURL}/openapi.yaml`,
};
