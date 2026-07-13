export type Product = {
  id: string;
  sku: string;
  name: string;
  description: string;
  priceMinor: number;
  currency: string;
  active: boolean;
  available: number;
  reserved: number;
  sold: number;
};

export type Sale = {
  id: string;
  productId: string;
  startsAt: string;
  endsAt: string;
  allocatedStock: number;
  maxQuantityPerUser: number;
  state: string;
};

export type DemoStatus = {
  product: Product;
  sale: Sale;
  reservations: number;
  orders: number;
  duplicateOrders: number;
  invariantPass: boolean;
};

export type LoadReport = {
  timestamp: string;
  initialInventory: number;
  totalAttempts: number;
  successful: number;
  soldOut: number;
  duplicate: number;
  rateLimited: number;
  failed: number;
  p50Millis: number;
  p95Millis: number;
  p99Millis: number;
  final: DemoStatus;
  zeroOverselling: boolean;
};

export type Order = {
  id: string;
  userId: string;
  state: string;
  quantity: number;
  createdAt: string;
};

export const staticStatus: DemoStatus = {
  product: {
    id: "static-product",
    sku: "FLASH-100",
    name: "StockRush Limited Drop",
    description: "Pre-generated deterministic portfolio evidence",
    priceMinor: 9900,
    currency: "THB",
    active: true,
    available: 0,
    reserved: 100,
    sold: 0,
  },
  sale: {
    id: "static-sale",
    productId: "static-product",
    startsAt: "2026-07-13T00:00:00Z",
    endsAt: "2026-07-13T01:00:00Z",
    allocatedStock: 100,
    maxQuantityPerUser: 1,
    state: "ended",
  },
  reservations: 100,
  orders: 100,
  duplicateOrders: 0,
  invariantPass: true,
};

export const staticReport: LoadReport = {
  timestamp: "2026-07-13T00:00:00Z",
  initialInventory: 100,
  totalAttempts: 1000,
  successful: 100,
  soldOut: 900,
  duplicate: 0,
  rateLimited: 0,
  failed: 0,
  p50Millis: 0,
  p95Millis: 0,
  p99Millis: 0,
  final: staticStatus,
  zeroOverselling: true,
};

export function isZeroOversell(report: LoadReport): boolean {
  const inventory = report.final.product;
  return (
    report.zeroOverselling &&
    report.successful <= report.initialInventory &&
    inventory.available >= 0 &&
    inventory.available + inventory.reserved + inventory.sold === report.initialInventory &&
    report.final.duplicateOrders === 0
  );
}
