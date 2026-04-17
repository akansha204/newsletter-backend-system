import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    subscribe_load: {
      executor: 'ramping-vus',
      startVUs: 5,
      stages: [
        { duration: '30s', target: 50 },
        { duration: '1m', target: 100 },
        { duration: '30s', target: 0 },
      ],
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.5'],   
    http_req_duration: ['p(95)<500'], 
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://127.0.0.1:3001';

export default function () {
  const email = `k6-${__VU}-${__ITER}-${Date.now()}@example.com`;

  const res = http.post(
    `${BASE_URL}/api/v1/subscribe`,
    JSON.stringify({ email }),
    {
      headers: { 'Content-Type': 'application/json' },
    }
  );

  check(res, {
    'status is 201 or 429': (r) => r.status === 201 || r.status === 429,
  });

  sleep(0.2); 
}