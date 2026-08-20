# Bug reproduction

Send a request body that exceeds the configured limit, then build an error
response with structured details. The decode layer turns the limit error into
text so callers cannot recover its type, and the response layer drops the
provided details.

Run the decode and error-response checks. The buggy branch cannot match
`http.MaxBytesError` and returns an error envelope with nil details.

Observed failure: `Decode lost MaxBytesError: request decode failed: http: request body too large`.
