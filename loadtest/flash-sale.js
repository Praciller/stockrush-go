import http from "k6/http";
import { check } from "k6";
import { Counter } from "k6/metrics";

const baseURL = __ENV.API_BASE_URL || "http://host.docker.internal:8080";
const strategy = __ENV.IDEMPOTENCY_STRATEGY || "unique";
const successes = new Counter("stockrush_successes");
const soldOut = new Counter("stockrush_sold_out");
const duplicates = new Counter("stockrush_duplicates");
const rateLimited = new Counter("stockrush_rate_limited");
const failed = new Counter("stockrush_failed");

http.setResponseCallback(http.expectedStatuses({ min: 200, max: 201 }, 409, 429));

export const options = {
  vus: Number(__ENV.VUS || 100),
  duration: __ENV.DURATION || "10s",
  thresholds: {
    http_req_failed: ["rate<0.95"],
    http_req_duration: ["p(95)<2000"],
  },
};

export function setup() {
  if (__ENV.SALE_ID) return { saleId: __ENV.SALE_ID };
  const response = http.get(`${baseURL}/api/v1/demo/status`, { headers: { "X-User-ID": "k6-setup" } });
  check(response, { "demo status available": (r) => r.status === 200 });
  return { saleId: response.json("data.sale.id") };
}

export default function (data) {
  const userId = `k6-${__VU}-${__ITER}`;
  const key = strategy === "shared" ? "k6-shared" : strategy === "user" ? `k6-${__VU}` : userId;
  const response = http.post(
    `${baseURL}/api/v1/sales/${data.saleId}/reservations`,
    JSON.stringify({ userId, quantity: 1 }),
    { headers: { "Content-Type": "application/json", "Idempotency-Key": key, "X-User-ID": userId } },
  );
  const code = response.json("error.code") || "";
  if (response.status === 201) successes.add(1);
  else if (response.status === 200) duplicates.add(1);
  else if (code === "INVENTORY_SOLD_OUT") soldOut.add(1);
  else if (response.status === 429) rateLimited.add(1);
  else failed.add(1);
  check(response, { "expected flash-sale response": (r) => [200, 201, 409, 429].includes(r.status) });
}
