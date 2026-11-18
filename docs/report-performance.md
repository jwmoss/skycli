# Reduce status and analytics latency

A read-only `status` invocation on November 17, 2026 returned five successful GET responses. The five requests took 211, 120, 62, 72, and 201 ms in one trace. The requests run in sequence.

`runStatus` reads the frame, chores, calendar, categories, and point balances. The categories and balances only depend on one another during output assembly. `runAnalytics` also waits for five resource reads before building its report.

- [ ] Resolve the frame and credentials once before starting parallel work. Keep refresh and secret storage outside the workers.
- [ ] Compare a small concurrency limit with the sequential baseline. Capture command duration, request count, and rate-limit responses.
- [ ] Preserve JSON ordering, cancellation, and resource-specific errors. Exercise calendar and chore date boundaries on a real frame.
