# Idempotency

Reservation requests hash sale ID, user ID, and quantity into a request fingerprint. The transaction first inserts the supplied key with `ON CONFLICT DO NOTHING`.

- The winner continues and stores the resource ID plus response metadata.
- Concurrent retries wait for the unique-key conflict to resolve, then replay the committed result.
- A different user or payload with the same key receives `IDEMPOTENCY_CONFLICT`.
- Failed sale, limit, and sold-out decisions are also stored and replayed.

Payment callbacks use a separate unique key and fingerprint. Only the first successful callback moves inventory from reserved to sold.
