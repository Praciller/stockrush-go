import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, describe, expect, it, vi } from "vitest";
import App from "./App";

afterEach(() => {
  vi.unstubAllEnvs();
  vi.resetModules();
});

describe("portfolio navigation", () => {
  it("renders a real destination for every internal navigation link", () => {
    const markup = renderToStaticMarkup(createElement(App));
    const links = [...markup.matchAll(/<a\b[^>]*href="#([^"]+)"/g)];

    expect(links.length).toBeGreaterThan(0);
    for (const [, target] of links) {
      expect(markup, `Missing section for #${target}`).toContain(`id="${target}"`);
    }
  });

  it("does not advertise a simulator without local simulation controls", () => {
    const markup = renderToStaticMarkup(createElement(App));
    const simulatorSectionExists = markup.includes('id="simulator"');
    const simulatorLinkExists = markup.includes('href="#simulator"');

    expect(simulatorLinkExists).toBe(simulatorSectionExists);
  });

  it("labels the GitHub Pages ledger as historical and links the source report beside it", async () => {
    vi.stubEnv("VITE_STATIC_DEMO", "true");
    vi.resetModules();
    const { default: StaticApp } = await import("./App");
    const markup = renderToStaticMarkup(createElement(StaticApp));
    const proof = markup.match(/<section id="proof"[\s\S]*?<\/section>/)?.[0] ?? "";

    expect(markup).toContain("No oversell observed in a local PostgreSQL test with 1,000 concurrent attempts and 100 units of stock.");
    expect(proof).toContain("HISTORICAL TEST: ZERO OVERSELLING");
    expect(proof).toContain("Historical local PostgreSQL test results (July 13, 2026).");
    expect(proof).toMatch(/<dt>Concurrent attempts<\/dt><dd>1,000<\/dd>/);
    expect(proof).toMatch(/<dt>Successful<\/dt><dd>100<\/dd>/);
    expect(proof).toMatch(/<dt>Sold out<\/dt><dd>900<\/dd>/);
    expect(proof).toContain('href="https://github.com/Praciller/stockrush-go/blob/main/reports/local_portfolio_report.md"');
    expect(proof).toContain("Read the full test report");
    expect(markup).not.toContain(">Evidence report</a>");
  });

  it("keeps the live/local report link and verdict behavior", async () => {
    vi.stubEnv("VITE_STATIC_DEMO", "false");
    vi.resetModules();
    const { default: LiveApp } = await import("./App");
    const markup = renderToStaticMarkup(createElement(LiveApp));

    expect(markup).toContain(">Evidence report</a>");
    expect(markup).toContain("PASS: ZERO OVERSELLING");
    expect(markup).not.toContain("Historical local PostgreSQL test results");
    expect(markup).not.toContain("Read the full test report");
  });
});
