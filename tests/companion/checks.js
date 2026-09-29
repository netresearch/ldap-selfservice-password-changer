const ASSETS = document.querySelector("meta[name=companion-assets]").content;
const THREE = await import(ASSETS + "/vendor/three.module.js");
const kind = "keyholder";
const results = document.getElementById("results");
const check = (condition, message) => {
  const row = document.createElement("li");
  row.textContent = `${condition ? "PASS" : "FAIL"}: ${message}`;
  results.append(row);
  if (!condition) throw new Error(message);
};
const wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
// Poll an observable result, with a deadline that fails instead of silently continuing.
const waitFor = async (condition, description, timeout = 20000) => {
  const deadline = performance.now() + timeout;
  while (!condition()) {
    if (performance.now() >= deadline)
      throw new Error(`Timed out: ${description}; animation=${avatar._time}, visibility=${document.visibilityState}`);
    await wait(50);
  }
};
const form = document.getElementById("form");
const avatar = document.querySelector("login-companion");
avatar.setAttribute("character", kind);
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
  await import(ASSETS + "/keyholder-login.js");
  form.addEventListener("submit", (event) => {
    if (!event.defaultPrevented) {
      submissions++;
      event.preventDefault();
    }
  });
  await waitFor(
    () => avatar.dataset.renderer && avatar._camera?.aspect > 0 && avatar._renderer?.domElement.width > 0,
    "renderer initialization"
  );
  check(THREE.REVISION === "186", "Pinned Three.js release is loaded");
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
    await waitFor(() => avatar._time >= avatar._actionStart + 3.5, "keyholder action settled", 120000);
    form.dispatchEvent(new Event("password-change-start"));
    check(actions === 2, "Validated password request triggers key jingle");
    check(submissions === 0, "Jingle does not submit the form itself");
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

  avatar.remove();
  check(!avatar._renderer && !avatar._scene, "Disconnect releases renderer and scene");
  const originalMatchMedia = window.matchMedia;
  window.matchMedia = (query) =>
    query.includes("prefers-reduced-motion")
      ? { matches: true, addEventListener() {}, removeEventListener() {} }
      : originalMatchMedia(query);
  const still = document.createElement("login-companion");
  still.setAttribute("character", kind);
  form.prepend(still);
  await waitFor(() => still.dataset.renderer === "webgl", "reduced-motion renderer");
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
