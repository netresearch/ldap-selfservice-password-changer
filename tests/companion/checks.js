const ASSETS = document.querySelector("meta[name=companion-assets]").content;
const THREE = await import(ASSETS + "/vendor/three.module.js");
const kind = document.body.dataset.character;
const results = document.getElementById("results");
const check = (condition, message) => {
  const row = document.createElement("li");
  row.textContent = `${condition ? "PASS" : "FAIL"}: ${message}`;
  results.append(row);
  if (!condition) throw new Error(message);
};
const wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
const form = document.getElementById("form");
const avatar = document.querySelector("scormiq-avatar");
avatar.setAttribute("character", kind === "portal" ? "robot" : kind);
if (kind === "wizard") avatar.setAttribute("action", "special");
let actions = 0;
avatar.addEventListener("avatar-action", () => actions++);
let submissions = 0;
try {
  const { animationRandom } = await import(ASSETS + "/animation-random.js");
  const originalGetRandomValues = globalThis.crypto.getRandomValues;
  try {
    for (const [sample, expected] of [
      [0, 0],
      [0x80000000, 0.5],
      [0xffffffff, 1 - 2 ** -32]
    ]) {
      globalThis.crypto.getRandomValues = (values) => {
        values[0] = sample;
        return values;
      };
      check(animationRandom() === expected, `Animation randomness preserves the [0, 1) range for ${sample}`);
    }
  } finally {
    globalThis.crypto.getRandomValues = originalGetRandomValues;
  }
  await import(
    ASSETS +
      (kind === "wizard" ? "/wizard-login.js" : kind === "keyholder" ? "/keyholder-login.js" : "/scormiq-avatar.js")
  );
  form.addEventListener("submit", (event) => {
    if (!event.defaultPrevented) {
      submissions++;
      event.preventDefault();
    }
  });
  await wait(250);
  check(avatar.dataset.renderer === "webgl", "WebGL renders under strict CSP");
  check(avatar.shadowRoot.querySelector(".motion-toggle").hidden, "No visible pause button");
  const box = avatar._button.getBoundingClientRect();
  const canvas = avatar._renderer.domElement;
  check(
    Math.abs(avatar._camera.aspect - box.width / box.height) < 0.001,
    "Camera uses actual canvas viewport proportions"
  );
  check(Math.abs(canvas.width / canvas.height - box.width / box.height) < 0.01, "Canvas is not stretched or squashed");
  if (kind === "keyholder") {
    avatar._scene.updateMatrixWorld(true);
    const keyCenter = new THREE.Box3()
      .setFromObject(avatar._rig.keys)
      .getCenter(new THREE.Vector3())
      .project(avatar._camera);
    const rect = canvas.getBoundingClientRect();
    const click = () =>
      canvas.dispatchEvent(
        new MouseEvent("click", {
          bubbles: true,
          detail: 1,
          clientX: rect.left + ((keyCenter.x + 1) / 2) * rect.width,
          clientY: rect.top + ((1 - keyCenter.y) / 2) * rect.height
        })
      );
    click();
    for (let i = 0; i < 20; i++) click();
    check(actions === 1, "Clicking the actual keys jingles once; repeated clicks do not restart");
    await wait(3700);
    form.dispatchEvent(new Event("password-change-start"));
    check(actions === 2, "Validated password request triggers key jingle");
    check(submissions === 0, "Jingle does not submit the form itself");
  }
  if (kind === "wizard") {
    form.requestSubmit();
    form.requestSubmit();
    check(actions === 1 && submissions === 0, "Login begins one guard animation and suppresses repeated submits");
    await wait(1600);
    check(submissions === 1, "Login posts once after impact, including a null submitter");
    check(form.getAttribute("aria-busy") === null, "Busy state clears after login replay");
    await wait(2100);
    const button = form.querySelector("button[type=submit]");
    form.requestSubmit(button);
    await wait(1600);
    check(submissions === 2, "Click submission also preserves the original submitter");
  }
  const image = avatar.querySelector("img");
  canvas.dispatchEvent(new Event("webglcontextlost", { cancelable: true }));
  check(avatar.dataset.renderer === "fallback", "Lost WebGL context switches to fallback");
  const slot = avatar.shadowRoot.querySelector("slot");
  check(
    !slot.hidden && slot.assignedElements().includes(image),
    "Original supplied image is visible after WebGL failure"
  );
  check(
    avatar._button.disabled && avatar.shadowRoot.querySelector(".motion-toggle").hidden,
    "Fallback has no false interactive controls"
  );
  if (kind === "wizard") {
    form.requestSubmit();
    check(submissions === 3, "WebGL fallback submits immediately");
  }
  avatar.remove();
  check(!avatar._renderer && !avatar._scene, "Disconnect releases renderer and scene");
  const originalMatchMedia = window.matchMedia;
  window.matchMedia = (query) =>
    query.includes("prefers-reduced-motion")
      ? { matches: true, addEventListener() {}, removeEventListener() {} }
      : originalMatchMedia(query);
  const still = document.createElement("scormiq-avatar");
  still.setAttribute("character", kind === "portal" ? "robot" : kind);
  form.prepend(still);
  await wait(100);
  check(
    still.dataset.motion === "still" && !still._frame,
    "Reduced motion renders a still pose without an animation loop"
  );
  still.remove();
  window.matchMedia = originalMatchMedia;
  document.getElementById("status").textContent = `PASS: ${results.children.length} checks`;
} catch (error) {
  document.getElementById("status").textContent = `FAIL: ${error.message}`;
}
