// One scenario against one API, with a fixed number of virtual users.
// Env: BASE_URL, SCENARIO (light | list | deep | tree | filter), VUS, DURATION, AUTH (user:password).
import http from "k6/http";
import { check } from "k6";
import encoding from "k6/encoding";

const SCENARIOS = {
  // One day through the order_date index: COUNT and page stay cheap, so the API side dominates.
  light: () => {
    const day = `2026-${pad(rand(1, 10))}-${pad(rand(1, 28))}`;
    return `/api/v1/orders?date_from=${day}&date_to=${day}&per_page=50`;
  },
  // Offsets below 10,000: plain LIMIT/OFFSET.
  list: () => `/api/v1/orders?page=${rand(1, 200)}&per_page=50`,
  // Offsets from 100,000: the deferred-join path.
  deep: () => `/api/v1/orders?page=${rand(2001, 20000)}&per_page=50&sort=order_date&order=desc`,
  // 500 rows grouped into the customer → product → order tree.
  tree: () => `/api/v1/orders/tree?page=${rand(1, 100)}&per_page=500&sort=customer_code`,
  // LIKE, equality and a date range together.
  filter: () =>
    `/api/v1/orders?customer_name=Customer%2001&status=shipped&date_from=2026-03-01&date_to=2026-08-31&page=${rand(1, 5)}&per_page=50`,
};

const path = SCENARIOS[__ENV.SCENARIO];
if (!path) throw new Error(`unknown SCENARIO: ${__ENV.SCENARIO}`);

export const options = {
  scenarios: {
    load: { executor: "constant-vus", vus: Number(__ENV.VUS), duration: __ENV.DURATION },
  },
  discardResponseBodies: true,
  summaryTrendStats: ["avg", "med", "p(90)", "p(99)", "max"],
};

const headers = { Authorization: `Basic ${encoding.b64encode(__ENV.AUTH)}` };

function pad(n) {
  return String(n).padStart(2, "0");
}

function rand(lo, hi) {
  return lo + Math.floor(Math.random() * (hi - lo + 1));
}

export default function () {
  const res = http.get(`${__ENV.BASE_URL}${path()}`, { headers });
  check(res, { "status is 200": (r) => r.status === 200 });
}
