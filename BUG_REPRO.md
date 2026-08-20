# Bug reproduction

Obtain a lineage snapshot, change an item in its Attempts slice, and obtain
another snapshot. The later result is contaminated by the caller mutation.
Also inspect the metrics snapshot after accepted and dead-lettered events.

Run the lineage snapshot and metrics checks. The buggy branch shares
Attempts backing storage and reports the accepted count as dead-lettered.

Observed failure: `dead_lettered counter matched accepted events`.
