# Bug reproduction

Request the first page of events, modify a value in the returned payload,
then request the same page again. The modified payload is visible in the
second result. A next-page cursor can also point to the first item and repeat
the first page.

Run the page snapshot and cursor checks. The buggy branch exposes the
shared payload and returns the wrong continuation cursor.

Observed failure: `next cursor points at the first item`.
