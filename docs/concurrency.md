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

Transactions that change an existing reservation serialize on its reservation row. When stock changes, lock inventory before sale allocation and update order state after stock. Expiration batches claim eligible reservations with SKIP LOCKED, then lock all affected inventory rows in product ID order before restoring any sale. Since a sale belongs to one product, a transaction touching the same sale has to pass that product's inventory lock first. Administrative sale state updates lock the sale alone.

Reserve takes the inventory lock inside the stock savepoint before checking sale allocation, preserving the sold-out response precedence. A rejected stock update rolls back to the savepoint before persisting the idempotency response. Payment locks the reservation explicitly. Its order foreign key takes KEY SHARE, which is compatible with the non-key order state update.

The PostgreSQL integration suite checks both lock orders with separate transactions and a wait-for edge observed via pg_blocking_pids. Mixed workload tests verify restoration and reconciliation across shared and distinct products. Run demo resets and schema migrations with application writers stopped.
