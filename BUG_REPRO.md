# Bug reproduction

Read a contract's rule history and modify the returned list. The current
active rule set is then lost because the returned list shares internal
state. Separately, execute a rule containing an invalid regular expression.

Run the catalog isolation and invalid-regex checks. The buggy branch loses
the active set and panics instead of returning a failed evaluation.

Observed failure: `active rule set not found` and `panic: regexp: Compile`.
