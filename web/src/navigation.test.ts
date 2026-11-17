import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import App from "./App";

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
});
