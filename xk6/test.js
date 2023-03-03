import { sleep, check } from 'k6';
import { Counter } from 'k6/metrics';
import frontend from 'k6/x/frontend';

// A simple counter for http requests

export const requests = new Counter('http_reqs');

// you can specify stages of your test (ramp up/down patterns) through the options object
// target is the number of VUs you are aiming for

const total = 100;

export const options = {

    stages: [
        { target: total, duration: '30s' },
        { target: total/2, duration: '30s' },
        { target: 0, duration: '30s' },
    ],
    thresholds: {
        http_reqs: ['count < 100'],
    },
};

export default function () {
    const host = "192.168.64.134:30706"

    // our HTTP request, note that we are saving the response to res, which can be accessed later
    console.log(frontend.matchRequest(host, {
        "matchmaking_queue": "default"
    }));
    console.log(frontend.matchRequest(host, {
        "matchmaking_queue": "default"
    }));
}
