/**
 * Live password-policy checklist.
 *
 * Renders a <li> per rule inside the provided <ul>, then returns an
 * `update(value)` function the caller invokes on every keystroke. On each
 * call, the list items flip between "unmet" (neutral dot) and "met" (green
 * check). Screen readers pick up the change via `aria-live="polite"` on the
 * <ul> and the per-item `aria-label` that includes met/unmet status.
 *
 * SECURITY: SVG nodes are built via createElementNS — no innerHTML, no
 * string concatenation. Rule labels reach the DOM via textContent.
 */

import type { PolicyRule } from "./validators.js";

const SVG_NS = "http://www.w3.org/2000/svg";

function makeDotIcon(): SVGSVGElement {
  const svg = document.createElementNS(SVG_NS, "svg");
  svg.setAttribute("viewBox", "0 0 20 20");
  svg.setAttribute("fill", "currentColor");
  svg.setAttribute("aria-hidden", "true");
  svg.setAttribute("class", "inline-block h-4 w-4 shrink-0");
  const circle = document.createElementNS(SVG_NS, "circle");
  circle.setAttribute("cx", "10");
  circle.setAttribute("cy", "10");
  circle.setAttribute("r", "3");
  svg.appendChild(circle);
  return svg;
}

function makeCheckIcon(): SVGSVGElement {
  const svg = document.createElementNS(SVG_NS, "svg");
  svg.setAttribute("viewBox", "0 0 20 20");
  svg.setAttribute("fill", "currentColor");
  svg.setAttribute("aria-hidden", "true");
  svg.setAttribute("class", "inline-block h-4 w-4 shrink-0");
  const path = document.createElementNS(SVG_NS, "path");
  path.setAttribute("fill-rule", "evenodd");
  path.setAttribute("clip-rule", "evenodd");
  path.setAttribute(
    "d",
    "M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
  );
  svg.appendChild(path);
  return svg;
}

/**
 * Render the policy list once. Returns an updater that toggles met/unmet
 * state on each <li> based on the current input value.
 */
export function renderPolicyList(list: HTMLElement, rules: PolicyRule[]): (value: string) => void {
  while (list.firstChild) list.removeChild(list.firstChild);

  const items = rules.map((rule) => {
    const li = document.createElement("li");
    li.className = "flex items-center gap-2 transition-colors";
    li.dataset["rule"] = rule.id;
    li.dataset["met"] = "false";
    li.setAttribute("aria-label", `${rule.label} — unmet`);

    const iconWrap = document.createElement("span");
    iconWrap.dataset["purpose"] = "icon";
    iconWrap.className = "inline-flex text-gray-500 dark:text-gray-400";
    iconWrap.appendChild(makeDotIcon());
    li.appendChild(iconWrap);

    const text = document.createElement("span");
    text.textContent = rule.label;
    li.appendChild(text);

    list.appendChild(li);
    return { li, iconWrap, rule };
  });

  return (value: string) => {
    for (const { li, iconWrap, rule } of items) {
      const met = rule.check(value);
      const prevMet = li.dataset["met"] === "true";
      if (met === prevMet) continue;

      li.dataset["met"] = met.toString();
      li.setAttribute("aria-label", `${rule.label} — ${met ? "met" : "unmet"}`);

      // Color: green-800 on light / green-300 on dark meets WCAG AAA 7:1.
      // Neutral gray for unmet (never red — unmet is not an error yet).
      li.classList.toggle("text-green-800", met);
      li.classList.toggle("dark:text-green-300", met);
      iconWrap.classList.toggle("text-green-800", met);
      iconWrap.classList.toggle("dark:text-green-300", met);
      iconWrap.classList.toggle("text-gray-500", !met);
      iconWrap.classList.toggle("dark:text-gray-400", !met);

      while (iconWrap.firstChild) iconWrap.removeChild(iconWrap.firstChild);
      iconWrap.appendChild(met ? makeCheckIcon() : makeDotIcon());
    }
  };
}
