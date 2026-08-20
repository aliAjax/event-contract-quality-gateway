# Bug reproduction

Finish a dead-letter replay unsuccessfully and inspect its audit entries,
then activate a stored rule version and read the active rules. The failed
replay has no completion audit event, while rule activation clears the
in-memory rule cache.

Run the replay audit and rule activation checks. The buggy branch leaves the
failed completion unrecorded and returns no active rules.

Observed failure: `rules=[]domain.Rule(nil)`.
