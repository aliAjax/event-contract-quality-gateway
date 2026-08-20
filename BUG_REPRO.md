# Bug reproduction

Reading a contract returns a mutable view of stored enum data and truncates
the version detail to its first field. Mutating an enum in a returned contract
changes the value seen by a later read.

Run the two targeted contract snapshot checks. The buggy branch reports a
mutated enum or missing version fields; the fixed branch preserves the
stored snapshot and complete version detail.

Observed failure: `expected stored enum to remain unchanged`.
