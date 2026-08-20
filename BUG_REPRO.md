# Bug reproduction

Cancel the context passed to server shutdown and load configuration with a
zero shutdown timeout. The shutdown hook receives a background context and
cannot observe cancellation; the invalid timeout is accepted and can leave
shutdown waiting indefinitely.

Run the shutdown cancellation and configuration checks. The buggy branch
does not propagate cancellation and accepts the non-positive timeout.

Observed failure: `shutdown hook did not receive canceled context`.
