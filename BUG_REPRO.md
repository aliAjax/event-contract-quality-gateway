# Bug reproduction

Start a worker with a cancelable context and cancel it, then call the gateway
signal wait helper with a canceled caller context. The worker continues
ticking because it watches a background context; the gateway path also waits
on a background context and can discard cancellation.

Run the worker cancellation and signal checks. The buggy branch does not
return after cancellation.

Observed failure: `worker ignored cancellation` and `err=<nil>`.
