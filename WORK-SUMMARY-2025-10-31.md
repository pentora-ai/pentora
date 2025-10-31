Date: 2025-10-31

Summary
- Fixed slow-network banner grabbing timeouts and made banner-grabber respect the global --timeout flag.
- Increased safe defaults (Read=10s, Connect=5s) to reduce empty service detections on high-latency targets.
- Added targeted planner tests to cover timeout propagation, instance ID uniqueness, and module selection.

Changes
- pkg/engine/planner.go: propagate ScanIntent.CustomTimeout to banner-grabber (read_timeout, connect_timeout).
- pkg/modules/scan/banner_grab.go: raise default ReadTimeout and ConnectTimeout.
- pkg/modules/scan/banner_grab_test.go: update expectations for new defaults.
- pkg/engine/planner_test.go: new tests for PlanDAG/configureModule/ID uniqueness.

Validation
- make test: PASS
- make validate: PASS (lint/format/spell/shell)
- Codecov: all modified and coverable lines covered.

PR
- fix(scanner): banner grab respects --timeout; raise safe defaults
- URL: https://github.com/pentora-ai/pentora/pull/122 (merged)

Follow-up
- Opened #124: Enhance banner-grabber timeouts with heuristic connect_timeout and optional fine-grained flags (--banner-read-timeout, --banner-connect-timeout).

Notes
- Kept behavior simple by mapping global --timeout to both read/connect; revisit heuristic or flags after user feedback.
