# Concurrency correctness

The inventory update is a single conditional SQL statement:

```sql
UPDATE inventory
SET available = available - $1,
    reserved = reserved + $1,
    updated_at = now()
WHERE product_id = $2
  AND available >= $1;
```

One affected row means the reservation won stock. Zero rows means sold out. The database check constraints remain a final guard against negative counters.

The integration suite synchronizes 1,000 goroutines on a start channel. With 100 available units it asserts exactly 100 successes, available inventory of zero, reserved plus sold inventory of 100, and no duplicate orders.

Additional deterministic tests cover 1,000 retries of one idempotency key, competing expiration workers, duplicate payments, cancellation versus expiration, and sale closure.
