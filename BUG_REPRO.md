# Bug reproduction

Publish two events for one tenant concurrently at the start of a quota
window whose limit is one. Both events can be accepted because the quota
read, decision, and write are not atomic. A rejection with validation
reasons is also classified as accepted instead of dead-lettered.

Run the targeted quota test with the race detector and the receipt-status
test. The buggy branch reports a data race and accepts more events than the
configured quota.

Observed failure: `WARNING: DATA RACE` and `status=accepted`.
