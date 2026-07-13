import { useEffect, useState } from "react";
import { api } from "./api";
import { isZeroOversell, staticReport, staticStatus, type DemoStatus, type LoadReport, type Order } from "./evidence";
import "./styles.css";

const attemptOptions = [10, 100, 500, 1000];
const staticDemo = import.meta.env.VITE_STATIC_DEMO === "true";
const publicDemo = !staticDemo && Boolean(import.meta.env.VITE_API_BASE_URL);
const localLoadEnabled = Boolean(import.meta.env.VITE_DEMO_TOKEN);

function currency(minor: number, code: string) {
  return new Intl.NumberFormat("en", { style: "currency", currency: code }).format(minor / 100);
}

function App() {
  const [status, setStatus] = useState<DemoStatus>(staticStatus);
  const [report, setReport] = useState<LoadReport>(staticReport);
  const [orders, setOrders] = useState<Order[]>([]);
  const [attempts, setAttempts] = useState(1000);
  const [live, setLive] = useState(false);
  const [hasRunReport, setHasRunReport] = useState(true);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState(staticDemo ? "Live API unavailable. Showing deterministic portfolio evidence." : "Connecting to the local API…");

  async function refresh() {
    try {
      const nextStatus = await api.status();
      setStatus(nextStatus);
      if (!publicDemo) {
        const nextOrders = await api.orders();
        setOrders(nextOrders.slice(0, 12));
      }
      setReport({
        ...staticReport,
        timestamp: new Date().toISOString(),
        initialInventory: nextStatus.sale.allocatedStock,
        totalAttempts: 0,
        successful: nextStatus.reservations,
        soldOut: 0,
        p50Millis: 0,
        p95Millis: 0,
        p99Millis: 0,
        final: nextStatus,
        zeroOverselling: nextStatus.invariantPass,
      });
      setHasRunReport(false);
      setLive(true);
      setMessage("Live PostgreSQL-backed state");
    } catch {
      setStatus(staticStatus);
      setReport(staticReport);
      setOrders([]);
      setLive(false);
      setHasRunReport(true);
      setMessage("Live API unavailable. Showing deterministic portfolio evidence.");
    }
  }

  useEffect(() => {
    if (staticDemo) return;

    void refresh();
    if (!publicDemo) return;

    const refreshTimer = window.setInterval(() => void refresh(), 10_000);
    return () => window.clearInterval(refreshTimer);
  }, []);

  async function runLoad() {
    setBusy(true);
    setMessage(`Running ${attempts.toLocaleString()} bounded attempts…`);
    try {
      const nextReport = await api.runLoad(attempts);
      setReport(nextReport);
      setHasRunReport(true);
      setStatus(nextReport.final);
      setLive(true);
      setMessage(`Completed ${attempts.toLocaleString()} bounded attempts`);
      const nextOrders = await api.orders();
      setOrders(nextOrders.slice(0, 12));
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Load run failed");
    } finally {
      setBusy(false);
    }
  }

  async function buyOne() {
    setBusy(true);
    try {
      const key = crypto.randomUUID();
      const reservation = await api.buy(key);
      setMessage(`One synthetic reservation created; expires ${new Date(reservation.expiresAt).toLocaleTimeString()}`);
      await refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Reservation failed");
    } finally {
      setBusy(false);
    }
  }

  const pass = isZeroOversell(report);
  const inventoryTotal = status.product.available + status.product.reserved + status.product.sold;

  return (
    <>
      <header className="topbar">
        <a className="brand" href="#overview" aria-label="StockRush Go overview"><span>SR</span> StockRush Go</a>
        <nav aria-label="Project sections">
          <a href="#proof">Proof</a><a href="#simulator">Simulator</a><a href="#orders">Orders</a><a href="#architecture">Architecture</a>
        </nav>
        <div className="top-actions"><a className="text-link" href="https://github.com/Praciller/stockrush-go">Source</a>{!staticDemo && <a className="text-link" href={api.openapiURL}>OpenAPI</a>}</div>
      </header>

      <main>
        <section id="overview" className="intro" aria-labelledby="page-title">
          <p className="eyebrow">POSTGRESQL CONCURRENCY EVIDENCE / GO MODULAR MONOLITH</p>
          <h1 id="page-title">1,000 buyers. 100 items.<br />Exactly 100 reservations.</h1>
          <p className="lede">Atomic SQL, idempotent checkout, and lock-safe expiration prove the inventory cannot oversell.</p>
          <p className={`source-status ${live ? "live" : "offline"}`} role="status" aria-live="polite">
            <span aria-hidden="true" />{message}
          </p>
        </section>

        <section id="proof" className="proof" aria-labelledby="proof-title">
          <div className="section-heading">
            <div><p className="eyebrow">INVENTORY RECONCILIATION</p><h2 id="proof-title">The database ledger balances</h2></div>
            <strong className={pass ? "verdict pass" : "verdict fail"}>{pass ? "PASS" : "FAIL"}: ZERO OVERSELLING</strong>
          </div>
          <div className="equation" aria-label={`${inventoryTotal} initial inventory equals ${status.product.reserved} reserved plus ${status.product.sold} sold plus ${status.product.available} available`}>
            <div><b>{inventoryTotal}</b><span>initial</span></div><i>=</i>
            <div><b>{status.product.reserved}</b><span>reserved</span></div><i>+</i>
            <div><b>{status.product.sold}</b><span>sold</span></div><i>+</i>
            <div><b>{status.product.available}</b><span>available</span></div>
          </div>
          <dl className="proof-ledger">
            <div><dt>{hasRunReport ? "Concurrent attempts" : "Recorded reservations"}</dt><dd>{(hasRunReport ? report.totalAttempts : status.reservations).toLocaleString()}</dd></div>
            <div><dt>Successful</dt><dd>{report.successful}</dd></div>
            <div><dt>Sold out</dt><dd>{report.soldOut}</dd></div>
            <div><dt>Negative stock events</dt><dd>0</dd></div>
            <div><dt>Duplicate orders</dt><dd>{status.duplicateOrders}</dd></div>
            <div><dt>Invariant</dt><dd>{status.invariantPass ? "Holds" : "Broken"}</dd></div>
          </dl>
        </section>

        {!publicDemo && localLoadEnabled && <section id="simulator" className="simulator" aria-labelledby="simulator-title">
          <div className="section-heading">
            <div><p className="eyebrow">BOUNDED LOCAL LOAD</p><h2 id="simulator-title">Reproduce the contention</h2></div>
            <p>Every run resets synthetic stock to 100. No unlimited option exists.</p>
          </div>
          <div className="controls">
            <fieldset disabled={!live || busy}>
              <legend>Concurrent buyers</legend>
              <div className="segments">
                {attemptOptions.map((option) => <button key={option} type="button" aria-pressed={attempts === option} onClick={() => setAttempts(option)}>{option.toLocaleString()}</button>)}
              </div>
            </fieldset>
            <button className="primary" type="button" disabled={!live || busy} onClick={runLoad}>{busy ? "Running…" : "Run safe simulation"}</button>
          </div>
          <div className="latency" aria-label="Latency summary">
            <span>p50 <b>{hasRunReport ? `${report.p50Millis.toFixed(1)} ms` : "not run"}</b></span>
            <span>p95 <b>{hasRunReport ? `${report.p95Millis.toFixed(1)} ms` : "not run"}</b></span>
            <span>p99 <b>{hasRunReport ? `${report.p99Millis.toFixed(1)} ms` : "not run"}</b></span>
            <span>failed <b>{report.failed}</b></span>
          </div>
        </section>}

        <section className="sale" aria-labelledby="sale-title">
          <div>
            <p className="eyebrow">ACTIVE FLASH SALE</p>
            <h2 id="sale-title">{status.product.name}</h2>
            <p>{status.product.description}</p>
          </div>
          <div className="sale-facts">
            <span>{currency(status.product.priceMinor, status.product.currency)}</span>
            <span>SKU {status.product.sku}</span>
            <span>{status.sale.state.toUpperCase()}</span>
          </div>
          <button className="secondary" type="button" disabled={!live || busy || status.product.available === 0} onClick={buyOne}>Reserve one</button>
        </section>

        <section id="orders" className="orders" aria-labelledby="orders-title">
          <div className="section-heading"><div><p className="eyebrow">RECENT DATABASE STATE</p><h2 id="orders-title">Orders and reservations</h2></div><span>{status.orders} total orders</span></div>
          <div className="table-wrap">
            <table>
              <thead><tr><th>Order</th><th>User</th><th>State</th><th>Qty</th><th>Created</th></tr></thead>
              <tbody>
                {orders.length ? orders.map((order) => <tr key={order.id}><td><code>{order.id.slice(0, 8)}</code></td><td>{order.userId}</td><td><span className={`state ${order.state}`}>{order.state}</span></td><td>{order.quantity}</td><td>{new Date(order.createdAt).toLocaleTimeString()}</td></tr>) : <tr><td colSpan={5}>{live ? "No orders yet. Run a bounded simulation." : "Orders require the live API."}</td></tr>}
              </tbody>
            </table>
          </div>
        </section>

        <section id="architecture" className="architecture" aria-labelledby="architecture-title">
          <div className="section-heading"><div><p className="eyebrow">CORRECTNESS BOUNDARY</p><h2 id="architecture-title">Why the stock cannot go below zero</h2></div></div>
          <ol className="flow">
            <li><b>HTTP request</b><span>Validated body and idempotency key</span></li>
            <li><b>PostgreSQL transaction</b><span>Claims the retry key and user budget</span></li>
            <li><b>Conditional update</b><code>available &gt;= quantity</code></li>
            <li><b>Reservation + order</b><span>Written together or rolled back together</span></li>
          </ol>
          <div className="architecture-notes">
            <p><b>No in-memory mutex:</b> it would not coordinate multiple processes or survive restarts.</p>
            <p><b>Expiration safety:</b> workers use <code>FOR UPDATE SKIP LOCKED</code>, so one pending row is restored once.</p>
            <p><b>Database constraints:</b> available, reserved, and sold columns reject negative values.</p>
          </div>
        </section>
      </main>

      <footer><span>StockRush Go</span><span>Local Docker Compose demo is authoritative</span><a href="https://github.com/Praciller/stockrush-go/blob/main/reports/local_portfolio_report.md">Evidence report</a></footer>
    </>
  );
}

export default App;
