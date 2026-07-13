import { describe, expect, it } from "vitest";
import { isZeroOversell, staticReport } from "./evidence";

describe("portfolio evidence", () => {
  it("accepts the reconciled static proof", () => {
    expect(isZeroOversell(staticReport)).toBe(true);
  });

  it("rejects negative inventory even when a verdict claims pass", () => {
    expect(isZeroOversell({ ...staticReport, final: { ...staticReport.final, product: { ...staticReport.final.product, available: -1 } } })).toBe(false);
  });
});
