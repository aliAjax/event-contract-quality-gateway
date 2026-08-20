# Bug reproduction

Validate an event containing an object field represented by a map, then
validate a schema with a duplicate field name. Both recoverable inputs panic
on the buggy branch: one in reflection and one while writing to an
uninitialized lookup map.

Run the object validation and duplicate-schema checks. The fixed branch
reports validation errors without crashing.

Observed failure: `reflect: call of reflect.Value.Elem on map Value`.
