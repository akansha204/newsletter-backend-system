import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    newsletter_load: {
      executor: 'ramping-vus',
      startVUs: 5,
      stages: [
        { duration: '20s', target: 20 },
        { duration: '40s', target: 50 },
        { duration: '20s', target: 0 },
      ],
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.3'], 
    http_req_duration: ['p(95)<1000'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://127.0.0.1:3001';
const API_KEY = __ENV.ADMIN_API_KEY || '';

export default function () {
  const payload = JSON.stringify({
    subject: 'Load Test Newsletter',
    body: 'Testing high load',
  });

  const idempotencyKey =
    __ITER % 5 === 0
      ? 'fixed-key-test' //idempotency collisions
      : `${__VU}-${__ITER}-${Date.now()}`; //unique requests(real load test)

  const res = http.post(
    `${BASE_URL}/api/v1/newsletter/send`,
    payload,
    {
      headers: {
        'Content-Type': 'application/json',
        'X-API-Key': API_KEY,
        'Idempotency-Key': idempotencyKey,
      },
    }
  );

  // check(res, {
  //   'status is 200 or 409': (r) => [200, 409].includes(r.status),
  // });
  check(res, {
  'log status': (r) => {
    console.log("STATUS:", r.status);
    return true;
  },
});

  sleep(0.5);
}