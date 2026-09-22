import * as THREE from "./vendor/three.module.js";
import { buildGopher, animateGopher, gopherFallback, applyCelStyle } from "./gopher-rigs.js";

const STYLES = new URL("./avatar.css", import.meta.url).href;
const LOGO = new URL("./assets/logos/netresearch-symbol-only.svg", import.meta.url).href;
const clamp = (n, min, max) => Math.min(max, Math.max(min, n));
const WAVE_DURATION = 2.4;
const WAVE_SETTLE = 0.3;
const CELEBRATION_DURATION = 3.6;
const ACTION_DURATION = 3.2;
const mix = (current, target, amount) => current + (target - current) * amount;

// A rounded box with genuine depth and smooth, analytic corner normals.
function roundedBox(width, height, depth, radius) {
  const geometry = new THREE.BoxGeometry(width, height, depth, 10, 10, 8);
  const positions = geometry.attributes.position;
  const normals = geometry.attributes.normal;
  const point = new THREE.Vector3();
  const core = new THREE.Vector3();
  for (let i = 0; i < positions.count; i++) {
    point.fromBufferAttribute(positions, i);
    core.set(
      clamp(point.x, -width / 2 + radius, width / 2 - radius),
      clamp(point.y, -height / 2 + radius, height / 2 - radius),
      clamp(point.z, -depth / 2 + radius, depth / 2 - radius)
    );
    point.sub(core).normalize();
    normals.setXYZ(i, point.x, point.y, point.z);
    point.multiplyScalar(radius).add(core);
    positions.setXYZ(i, point.x, point.y, point.z);
  }
  return geometry;
}

function roundedPlate(w, h, depth, radius) {
  const x = -w / 2,
    y = -h / 2,
    r = Math.min(radius, w / 2, h / 2);
  const shape = new THREE.Shape();
  shape.moveTo(x + r, y);
  shape.lineTo(x + w - r, y);
  shape.quadraticCurveTo(x + w, y, x + w, y + r);
  shape.lineTo(x + w, y + h - r);
  shape.quadraticCurveTo(x + w, y + h, x + w - r, y + h);
  shape.lineTo(x + r, y + h);
  shape.quadraticCurveTo(x, y + h, x, y + h - r);
  shape.lineTo(x, y + r);
  shape.quadraticCurveTo(x, y, x + r, y);
  const bevel = Math.min(depth / 4, 0.012);
  const geometry = new THREE.ExtrudeGeometry(shape, {
    depth: depth - bevel * 2,
    bevelEnabled: true,
    bevelSize: bevel,
    bevelThickness: bevel,
    bevelSegments: 3,
    steps: 1,
    curveSegments: 12
  });
  geometry.translate(0, 0, -depth / 2 + bevel);
  return geometry;
}

/** Embeddable ScormIQ mascot. All asset URLs resolve relative to this module. */
export class ScormiqAvatar extends HTMLElement {
  constructor() {
    super();
    this.attachShadow({ mode: "open" });
    this._state = "idle";
    this._follow = true;
    this._paused = false;
    this._time = 0;
    this._waveStart = -100;
    this._celebrateStart = -100;
    this._actionStart = -100;
    this._actionAmount = 0;
    this._lastGreetingAt = -Infinity;
    this._quietUntil = -Infinity;
    this._poseReady = false;
    this._fxOpacity = 0;
    this._pointer = { x: 0, y: 0, at: -Infinity };
    this._look = { x: 0, y: 0 };
    this._visible = true;
    this._frame = 0;
    this._last = 0;
    this._blinkStart = -100;
  }

  connectedCallback() {
    if (this._renderer) return;
    this._character = ["keyholder", "wizard"].includes(this.getAttribute("character"))
      ? this.getAttribute("character")
      : "robot";
    this._nextBlink = this._time + (this._character === "keyholder" ? 5.6 : 2.8);
    this._rig = null;
    this.shadowRoot.innerHTML = `
      <link rel="stylesheet" href="${STYLES}">
      <button class="avatar" type="button"><slot class="original-image" hidden></slot>
        <svg class="fallback" viewBox="0 0 400 440" role="img" aria-label="Friendly turquoise ScormIQ robot with the Netresearch symbol on its chest">
          <ellipse cx="200" cy="385" rx="75" ry="12" fill="#2f99a4" opacity=".12"/>
          <path d="M200 75V51" stroke="#585961" stroke-width="8"/><circle cx="200" cy="47" r="10" fill="#ff4d00"/>
          <rect x="137" y="249" width="126" height="101" rx="40" fill="#2f99a4"/>
          <rect x="88" y="261" width="35" height="70" rx="17" fill="#d6e5e5" transform="rotate(12 105 296)"/>
          <rect x="277" y="261" width="35" height="70" rx="17" fill="#d6e5e5" transform="rotate(-12 295 296)"/>
          <rect x="137" y="342" width="50" height="30" rx="14" fill="#d6e5e5"/><rect x="213" y="342" width="50" height="30" rx="14" fill="#d6e5e5"/>
          <rect x="88" y="81" width="224" height="163" rx="58" fill="#fff" stroke="#d6e5e5" stroke-width="5"/>
          <rect x="106" y="105" width="188" height="116" rx="41" fill="#173c43"/>
          <rect x="144" y="140" width="18" height="35" rx="9" fill="#c0f9ef"/><rect x="238" y="140" width="18" height="35" rx="9" fill="#c0f9ef"/>
          <path d="M182 186Q200 203 218 186" stroke="#c0f9ef" stroke-width="6" fill="none" stroke-linecap="round"/>
          <rect x="168" y="267" width="64" height="64" rx="18" fill="white"/>
          <image href="${LOGO}" x="170" y="269" width="60" height="60"/>
        </svg>
      </button><button class="motion-toggle" type="button" aria-pressed="false" hidden></button>`;
    this._button = this.shadowRoot.querySelector("button");
    if (this._character !== "robot")
      this.shadowRoot.querySelector(".fallback").outerHTML = gopherFallback(this._character);
    const german = (this.getAttribute("lang") || document.documentElement.lang).startsWith("de");
    const name =
      this._character === "robot"
        ? "ScormIQ robot"
        : this._character === "wizard"
          ? "wizard gopher"
          : "keyholder gopher";
    this._button.setAttribute(
      "aria-label",
      this.getAttribute("label") ||
        (german
          ? "Begleiter begrüßen. Pfeiltasten steuern den Blick."
          : "Greet the " + name + ". Arrow keys change its gaze.")
    );
    this._pauseButton = this.shadowRoot.querySelector(".motion-toggle");
    this._pauseButton.textContent = german ? "Animation pausieren" : "Pause animation";
    this._fallback = this.shadowRoot.querySelector(".fallback");
    this._abort = new AbortController();
    const on = (target, type, callback, options = {}) =>
      target.addEventListener(type, callback, { ...options, signal: this._abort.signal });
    this._motion = matchMedia("(prefers-reduced-motion: reduce)");
    try {
      this._renderer = new THREE.WebGLRenderer({ alpha: true, antialias: true, powerPreference: "low-power" });
      this._renderer.setPixelRatio(Math.min(devicePixelRatio || 1, 2));
      this._renderer.setClearColor(0x000000, 0);
      this._renderer.outputColorSpace = THREE.SRGBColorSpace;
      this._renderer.toneMapping = THREE.ACESFilmicToneMapping;
      this._renderer.toneMappingExposure = 1.2;
      this._button.append(this._renderer.domElement);
      this._fallback.setAttribute("hidden", "");
      this._buildScene();
      this.dataset.renderer = "webgl";
    } catch (error) {
      this._showFallback(error);
      return;
    }
    on(this._button, "click", (event) => {
      const keysClicked = this._character === "keyholder" && (event.detail === 0 || this._hitsKeys(event));
      this.getAttribute("action") === "special" || keysClicked ? this.performAction() : this.wave();
      this.dispatchEvent(new CustomEvent("avatar-activate", { bubbles: true, composed: true }));
    });
    on(this._pauseButton, "click", () => {
      this.paused = !this.paused;
    });
    on(
      window,
      "pointermove",
      (event) => {
        if (!this._follow || this._paused || this._motion.matches) return;
        if (event.pointerType === "touch" && !event.composedPath().includes(this)) return;
        const box = this.getBoundingClientRect();
        this._pointer.x = clamp((event.clientX - box.left - box.width / 2) / (box.width * 0.65), -1, 1);
        this._pointer.y = clamp((event.clientY - box.top - box.height * 0.4) / (box.height * 0.65), -1, 1);
        this._pointer.at = performance.now();
        this._pointer.input = "pointer";
      },
      { passive: true }
    );
    const resetGaze = () => {
      this._pointer.at = -Infinity;
    };
    on(document.documentElement, "pointerleave", resetGaze);
    on(window, "blur", resetGaze);
    on(this._button, "keydown", (event) => {
      if (!["ArrowLeft", "ArrowRight", "ArrowUp", "ArrowDown", "Escape"].includes(event.key)) return;
      event.preventDefault();
      if (this._paused || this._motion.matches) return;
      this._pointer.x = event.key === "ArrowLeft" ? -0.8 : event.key === "ArrowRight" ? 0.8 : 0;
      this._pointer.y = event.key === "ArrowUp" ? -0.7 : event.key === "ArrowDown" ? 0.7 : 0;
      this._pointer.at = performance.now();
      this._pointer.input = "keyboard";
    });
    on(this._motion, "change", () => this._syncLoop());
    on(document, "visibilitychange", () => this._syncLoop());
    on(this._renderer.domElement, "webglcontextlost", (event) => {
      event.preventDefault();
      this._showFallback(new Error("WebGL context lost"));
    });
    this._resizeObserver = new ResizeObserver(() => this._resize());
    this._resizeObserver.observe(this._button);
    this._intersectionObserver = new IntersectionObserver((entries) => {
      this._visible = entries[0].isIntersecting;
      this._syncLoop();
    });
    this._intersectionObserver.observe(this);
    this._resize();
    this._syncLoop();
  }

  _buildScene() {
    this._poseReady = false;
    this._fxOpacity = 0;
    this._scene = new THREE.Scene();
    this._camera = new THREE.PerspectiveCamera(33, 1, 0.1, 40);
    this._camera.position.set(0, 0.65, 8.3);
    this._camera.lookAt(0, this._character === "wizard" ? 0.5 : 0.12, 0);
    this._scene.add(new THREE.HemisphereLight(0xffffff, 0x91b3b4, 2.6));
    const key = new THREE.DirectionalLight(0xffffff, 3.4);
    key.position.set(-3, 5, 6);
    this._scene.add(key);
    const rim = new THREE.DirectionalLight(0xc1f4f7, 2.2);
    rim.position.set(3, 2, -4);
    this._scene.add(rim);
    const fill = new THREE.DirectionalLight(0xffffff, 0.65);
    fill.position.set(4, -1, 4);
    this._scene.add(fill);
    const material = (color, roughness = 0.36, metalness = 0.08) =>
      new THREE.MeshStandardMaterial({ color, roughness, metalness });
    const white = material("#edf3f3", 0.3);
    const teal = material("#2F99A4", 0.34);
    const dark = material("#173c43", 0.22, 0.25);
    const joint = material("#585961", 0.45, 0.28);
    const lightTeal = material("#b0d6d8", 0.35);
    const orange = material("#FF4D00", 0.35);
    const glow = new THREE.MeshBasicMaterial({ color: "#bbfff0", transparent: true, depthWrite: false });
    const happyGlow = new THREE.MeshBasicMaterial({
      color: "#bbfff0",
      transparent: true,
      depthWrite: false,
      opacity: 0
    });
    const cheekGlow = new THREE.MeshBasicMaterial({
      color: "#2F99A4",
      transparent: true,
      depthWrite: false,
      opacity: 0
    });
    this._faceMaterials = { normal: glow, happy: happyGlow, cheeks: cheekGlow };
    this._robot = new THREE.Group();
    this._scene.add(this._robot);
    const mesh = (geometry, mat, parent, x = 0, y = 0, z = 0) => {
      const object = new THREE.Mesh(geometry, mat);
      object.position.set(x, y, z);
      parent.add(object);
      return object;
    };
    const box = (w, h, d, r, mat, parent, x = 0, y = 0, z = 0) => mesh(roundedBox(w, h, d, r), mat, parent, x, y, z);
    const plate = (w, h, d, r, mat, parent, x = 0, y = 0, z = 0) =>
      mesh(roundedPlate(w, h, d, r), mat, parent, x, y, z);
    const sphere = (radius, mat, parent, x, y, z) =>
      mesh(new THREE.SphereGeometry(radius, 28, 20), mat, parent, x, y, z);
    if (this._character !== "robot") {
      // Robot-only materials have no scene owner in the gopher rigs.
      [white, teal, dark, joint, lightTeal, orange, glow, happyGlow, cheekGlow].forEach((mat) => mat.dispose());
      buildGopher(this, this._character);
    } else {
      this.dataset.character = "robot";
      this._torso = new THREE.Group();
      this._robot.add(this._torso);
      box(1.32, 1.2, 0.94, 0.31, teal, this._torso, 0, -0.5);
      box(0.86, 0.92, 0.1, 0.045, lightTeal, this._torso, 0, -0.55, -0.475);
      mesh(new THREE.CylinderGeometry(0.24, 0.29, 0.25, 32), joint, this._torso, 0, 0.17);
      plate(0.69, 0.66, 0.12, 0.19, white, this._torso, 0, -0.48, 0.493);
      this._logoTexture = new THREE.TextureLoader().load(LOGO, () => {
        if (this.isConnected && this._scene && this.dataset.renderer === "webgl") this._render();
      });
      this._logoTexture.colorSpace = THREE.SRGBColorSpace;
      mesh(
        new THREE.PlaneGeometry(0.68, 0.68),
        new THREE.MeshBasicMaterial({ map: this._logoTexture, transparent: true, depthWrite: false }),
        this._torso,
        0,
        -0.48,
        0.561
      );
      for (let i = 0; i < 3; i++)
        box(0.065, 0.027, 0.016, 0.008, i === 0 ? orange : dark, this._torso, -0.1 + i * 0.1, -0.93, 0.441);
      this._head = new THREE.Group();
      this._head.position.y = 0.91;
      this._robot.add(this._head);
      box(2.08, 1.48, 1.3, 0.4, white, this._head);
      plate(1.81, 1.15, 0.14, 0.4, teal, this._head, 0, -0.015, 0.609);
      plate(1.73, 1.07, 0.16, 0.36, dark, this._head, 0, -0.015, 0.671);
      // A slim reflection gives the dark display a curved-glass feel.
      box(
        0.62,
        0.026,
        0.008,
        0.003,
        new THREE.MeshBasicMaterial({ color: "#48676c" }),
        this._head,
        -0.28,
        0.414,
        0.756
      );
      this._face = new THREE.Group();
      this._face.position.z = 0.763;
      this._head.add(this._face);
      this._eyes = [-0.4, 0.4].map((x) => plate(0.18, 0.34, 0.037, 0.09, glow, this._face, x, 0.06));
      const smileCurve = new THREE.QuadraticBezierCurve3(
        new THREE.Vector3(-0.16, -0.22, 0.018),
        new THREE.Vector3(0, -0.39, 0.018),
        new THREE.Vector3(0.16, -0.22, 0.018)
      );
      this._smile = mesh(new THREE.TubeGeometry(smileCurve, 20, 0.023, 8, false), glow, this._face);
      this._cheeks = [-0.6, 0.6].map((x) => box(0.14, 0.038, 0.02, 0.009, cheekGlow, this._face, x, -0.19));
      this._happyEyes = [-0.4, 0.4].map((x) => {
        const curve = new THREE.QuadraticBezierCurve3(
          new THREE.Vector3(-0.12, 0, 0.025),
          new THREE.Vector3(0, 0.25, 0.025),
          new THREE.Vector3(0.12, 0, 0.025)
        );
        return mesh(new THREE.TubeGeometry(curve, 18, 0.028, 8, false), happyGlow, this._face, x, 0.025);
      });
      const grin = new THREE.Shape();
      grin.moveTo(-0.27, -0.18);
      grin.quadraticCurveTo(0, -0.24, 0.27, -0.18);
      grin.quadraticCurveTo(0.23, -0.46, 0, -0.46);
      grin.quadraticCurveTo(-0.23, -0.46, -0.27, -0.18);
      this._grin = mesh(new THREE.ShapeGeometry(grin, 24), happyGlow, this._face, 0, 0.045, 0.025);
      for (const side of [-1, 1]) {
        const ear = mesh(new THREE.CylinderGeometry(0.3, 0.3, 0.19, 40), teal, this._head, side * 1.015, -0.03, -0.035);
        ear.rotation.z = Math.PI / 2;
        const earCap = sphere(0.17, lightTeal, this._head, side * 1.126, -0.03, -0.035);
        earCap.scale.x = 0.33;
      }
      mesh(new THREE.CylinderGeometry(0.06, 0.07, 0.25, 20), joint, this._head, 0.05, 0.842, -0.1);
      this._antenna = sphere(0.118, orange, this._head, 0.05, 1.02, -0.1);
      sphere(0.029, white, this._head, 0.02, 1.07, -0.005);
      this._arms = [-1, 1].map((side) => {
        const pivot = new THREE.Group();
        pivot.position.set(side * 0.78, -0.15, 0);
        this._torso.add(pivot);
        sphere(0.19, joint, pivot, 0, 0, 0);
        box(0.34, 0.67, 0.39, 0.15, white, pivot, side * 0.02, -0.33, 0.04);
        box(0.29, 0.09, 0.34, 0.043, teal, pivot, side * 0.02, -0.57, 0.04);
        sphere(0.168, white, pivot, side * 0.02, -0.7, 0.04);
        return pivot;
      });
      this._feet = [-1, 1].map((side) => {
        const foot = new THREE.Group();
        foot.position.set(side * 0.39, -1.19, 0.03);
        this._torso.add(foot);
        mesh(new THREE.CylinderGeometry(0.12, 0.13, 0.22, 20), joint, foot, 0, 0, 0);
        box(0.48, 0.32, 0.67, 0.14, white, foot, 0, -0.13, 0.105);
        box(0.45, 0.09, 0.62, 0.035, teal, foot, 0, -0.27, 0.105);
        return foot;
      });
    }
    const celRamp = applyCelStyle(this._robot, this._scene);
    const shadowCanvas = document.createElement("canvas");
    shadowCanvas.width = shadowCanvas.height = 128;
    const ctx = shadowCanvas.getContext("2d");
    const gradient = ctx.createRadialGradient(64, 64, 3, 64, 64, 64);
    gradient.addColorStop(0, "rgba(31,80,82,.26)");
    gradient.addColorStop(0.42, "rgba(31,80,82,.13)");
    gradient.addColorStop(1, "rgba(31,80,82,0)");
    ctx.fillStyle = gradient;
    ctx.fillRect(0, 0, 128, 128);
    const shadowTexture = new THREE.CanvasTexture(shadowCanvas);
    this._shadow = mesh(
      new THREE.PlaneGeometry(3.5, 2.4),
      new THREE.MeshBasicMaterial({ map: shadowTexture, transparent: true, depthWrite: false }),
      this._scene,
      0,
      -1.72,
      0
    );
    this._shadow.rotation.x = -Math.PI / 2;
    if (this._character === "keyholder") {
      this._shadow.geometry.dispose();
      this._shadow.material.dispose();
      shadowTexture.dispose();
      this._shadow.geometry = new THREE.CircleGeometry(1, 64).scale(1.42, 0.62, 1);
      this._shadow.material = new THREE.MeshBasicMaterial({
        color: "#91ac98",
        transparent: true,
        opacity: 0.28,
        depthWrite: false
      });
      this._shadow.position.set(0, -1.525, 0.14);
    }
    // Preallocated celebration props: no timers, new meshes, or allocations per burst.
    this._celebrationFX = new THREE.Group();
    this._scene.add(this._celebrationFX);
    const confettiGeometry = new THREE.BoxGeometry(0.055, 0.11, 0.018);
    const confettiMaterials = ["#2F99A4", "#FF4D00", "#b0d6d8", "#ffffff"].map(
      (color) =>
        new THREE.MeshToonMaterial({
          color,
          gradientMap: celRamp,
          toneMapped: false,
          emissive: new THREE.Color(color).multiplyScalar(0.065)
        })
    );
    this._confettiMaterials = confettiMaterials;
    confettiMaterials.forEach((mat) => {
      mat.transparent = true;
      mat.depthWrite = false;
    });
    this._confetti = Array.from({ length: 36 }, (_, i) => {
      const side = i % 2 ? 1 : -1;
      const piece = mesh(confettiGeometry, confettiMaterials[i % 4], this._celebrationFX);
      // Deterministic spread keeps repeated celebrations lively and testable.
      piece.userData.flight = {
        side,
        delay: (i % 6) * 0.035,
        vx: side * (0.36 + (i % 7) * 0.1),
        vy: 1.8 + (i % 5) * 0.18,
        z: -0.3 + (i % 4) * 0.2,
        spin: 2 + (i % 5)
      };
      return piece;
    });
    const starShape = new THREE.Shape();
    for (let i = 0; i < 8; i++) {
      const angle = Math.PI / 2 + (i * Math.PI) / 4;
      const radius = i % 2 ? 0.045 : 0.16;
      if (i === 0) starShape.moveTo(Math.cos(angle) * radius, Math.sin(angle) * radius);
      else starShape.lineTo(Math.cos(angle) * radius, Math.sin(angle) * radius);
    }
    starShape.closePath();
    const starGeometry = new THREE.ShapeGeometry(starShape);
    this._sparkles = [-1, 1].map((side, i) =>
      mesh(starGeometry, confettiMaterials[i], this._celebrationFX, side * 1.48, 0.88 + i * 0.4, 0.7)
    );
  }

  _hitsKeys(event) {
    if (!this._rig?.keys) return false;
    const bounds = this._renderer.domElement.getBoundingClientRect();
    const point = new THREE.Vector2(
      ((event.clientX - bounds.left) / bounds.width) * 2 - 1,
      1 - ((event.clientY - bounds.top) / bounds.height) * 2
    );
    this._scene.updateMatrixWorld(true);
    const raycaster = new THREE.Raycaster();
    raycaster.setFromCamera(point, this._camera);
    // Include the holes in the key ring and a small margin for touch input.
    const keys = new THREE.Box3().setFromObject(this._rig.keys).expandByScalar(0.09);
    return raycaster.ray.intersectsBox(keys);
  }

  _resize() {
    if (!this._renderer || !this._camera || this.dataset.renderer !== "webgl") return;
    const { width, height } = this._button.getBoundingClientRect();
    if (!width || !height) return;
    this._renderer.setSize(width, height, false);
    this._camera.aspect = width / height;
    this._camera.position.z =
      this._character === "wizard" ? Math.max(11, 7.2 / this._camera.aspect) : Math.max(8.3, 6.4 / this._camera.aspect);
    this._camera.updateProjectionMatrix();
    this._render();
  }

  _render(delta = 0) {
    if (this.dataset.renderer !== "webgl" || !this._scene) return;
    const staticPose = this._paused || this._motion.matches;
    // Re-target from the displayed pose, even when a state changes mid-transition.
    // Only animation frames advance the blend; click/resize renders cannot jump it.
    const blend = !this._poseReady || staticPose ? 1 : 1 - Math.exp(-Math.max(0, delta) * 11);
    const t = staticPose ? 0 : this._time;
    const actionAge = this._time - this._actionStart;
    const actionTarget =
      !staticPose && actionAge >= 0 && actionAge < ACTION_DURATION
        ? Math.sin((Math.PI * actionAge) / ACTION_DURATION) ** 0.5
        : 0;
    this._actionAmount = mix(this._actionAmount, actionTarget, blend);
    const action = this._actionAmount;
    const wizard = this._character === "wizard";
    const keyholder = this._character === "keyholder";
    const guarding = wizard && !staticPose && actionAge >= 0 && actionAge < ACTION_DURATION;
    const windup = guarding && actionAge < 0.78 ? Math.sin(Math.PI * clamp(actionAge / 0.78, 0, 1)) : 0;
    const brace = guarding
      ? THREE.MathUtils.smoothstep(actionAge, 0.56, 0.88) *
        (1 - THREE.MathUtils.smoothstep(actionAge, 2.18, ACTION_DURATION))
      : 0;
    const thinking = this._state === "thinking";
    const happy = this._state === "happy";
    const waveAge = this._time - this._waveStart;
    const wave =
      !staticPose && waveAge >= 0 && waveAge < WAVE_DURATION ? Math.sin((Math.PI * waveAge) / WAVE_DURATION) ** 2 : 0;
    const celebrationAge = this._time - this._celebrateStart;
    const burstActive = !staticPose && celebrationAge >= 0 && celebrationAge < CELEBRATION_DURATION;
    const celebrating = happy && burstActive;
    const c = burstActive ? celebrationAge : 0;
    const joy = celebrating ? Math.sin(Math.PI * clamp(c / CELEBRATION_DURATION, 0, 1)) : 0;
    // Two eager hops, then ease into a buoyant, hands-up happy idle.
    const hop =
      celebrating && !keyholder
        ? Math.max(0, Math.sin(((c - 0.13) * Math.PI) / 0.68)) * 0.42 * Math.max(0, 1 - c / 2.6)
        : 0;
    const anticipation = celebrating && !keyholder && c < 0.13 ? Math.sin((c / 0.13) * Math.PI) * 0.08 : 0;
    const dance = happy ? Math.sin(t * 4.5) : 0;
    const bob = keyholder ? 0 : Math.sin(t * 1.7) * 0.065;
    this._robot.position.y = mix(
      this._robot.position.y,
      keyholder
        ? 0
        : (bob + hop - anticipation + (happy ? Math.abs(Math.sin(t * 3.4)) * 0.055 : 0)) * (1 - brace) +
            windup * 0.12 -
            brace * 0.1,
      blend
    );
    this._robot.rotation.x = mix(this._robot.rotation.x, brace * 0.08, blend);
    this._robot.scale.set(
      mix(this._robot.scale.x, 1 + anticipation, blend),
      mix(this._robot.scale.y, 1 - anticipation + hop * 0.07, blend),
      mix(this._robot.scale.z, 1 + anticipation, blend)
    );
    this._robot.rotation.y = mix(this._robot.rotation.y, this._look.x * 0.16 + dance * joy * 0.2, blend);
    this._robot.rotation.z = mix(
      this._robot.rotation.z,
      keyholder ? 0 : Math.sin(t * 0.9) * 0.018 + dance * (happy ? 0.045 + joy * 0.06 : 0),
      blend
    );
    this._head.rotation.y = mix(
      this._head.rotation.y,
      this._look.x * (keyholder ? 0.11 : 0.4) * (1 - brace * 0.8) + Math.sin(t * 0.65) * (keyholder ? 0.008 : 0.028),
      blend
    );
    this._head.rotation.x = mix(
      this._head.rotation.x,
      this._look.y * (keyholder ? 0.035 : 0.26) * (1 - brace * 0.8) +
        Math.sin(t * 1.3) * (keyholder ? 0.004 : 0.018) +
        brace * 0.1,
      blend
    );
    this._head.rotation.z = mix(
      this._head.rotation.z,
      keyholder
        ? (thinking ? -0.028 : -0.008) + Math.sin(t * 1.1) * 0.004 - wave * 0.018 - dance * joy * 0.018
        : (thinking ? -0.12 : happy ? -0.09 : -0.025) + Math.sin(t * 1.1) * 0.016 + wave * -0.065 - dance * joy * 0.07,
      blend
    );
    this._face.position.x = mix(this._face.position.x, this._look.x * 0.085, blend);
    this._face.position.y = mix(this._face.position.y, -this._look.y * 0.06, blend);
    const cheer = happy ? 1.18 + joy * 0.94 + Math.sin(t * 13) * joy * 0.18 : 0;
    const wavePose = wave * (happy ? 0.3 : 2.3 + Math.sin(waveAge * 18) * 0.23);
    this._arms[0].rotation.z = mix(
      this._arms[0].rotation.z,
      wizard
        ? -0.52 - joy * 0.12 - windup * 0.65 - brace * 0.12
        : keyholder
          ? 0.26 + Math.sin(t * 1.7 + 0.6) * 0.04 - cheer * 0.65 - wavePose * 0.8
          : -0.2 + Math.sin(t * 1.7 + 0.6) * 0.065 - cheer,
      blend
    );
    this._arms[1].rotation.z = mix(
      this._arms[1].rotation.z,
      keyholder
        ? -0.28 + joy * 0.17 + action * 0.65
        : (wizard ? 0.64 + brace * 1.13 : 0.2) - Math.sin(t * 1.7 + 0.6) * 0.065 + cheer * (1 - brace) + wavePose,
      blend
    );
    this._arms[0].rotation.x = mix(this._arms[0].rotation.x, -joy * 0.28 - windup * 0.48, blend);
    this._arms[1].rotation.x = mix(this._arms[1].rotation.x, wave * -0.25 - joy * 0.28 - brace * 0.55, blend);
    this._feet.forEach((foot, i) => {
      if (wizard) foot.position.x = mix(foot.position.x, (i === 0 ? -1 : 1) * (0.49 + brace * 0.16), blend);
      foot.rotation.x = mix(foot.rotation.x, keyholder ? 0 : Math.sin(t * 1.7 + i * 1.3) * 0.035 - hop * 0.45, blend);
      foot.rotation.z = mix(foot.rotation.z, keyholder ? 0 : (i === 0 ? -1 : 1) * hop * 0.5, blend);
    });
    const blinkAge = this._time - this._blinkStart;
    const blink =
      !staticPose && blinkAge >= 0 && blinkAge < 0.19 ? 1 - Math.sin((blinkAge / 0.19) * Math.PI) * 0.94 : 1;
    if (this._rig) {
      const quietIdle =
        !staticPose &&
        !happy &&
        !thinking &&
        waveAge >= WAVE_DURATION + WAVE_SETTLE &&
        actionAge >= ACTION_DURATION + WAVE_SETTLE;
      animateGopher(this, {
        blend,
        t,
        blink,
        happy,
        thinking,
        wave,
        joy,
        action,
        staticPose,
        windup,
        brace,
        actionAge,
        quietIdle
      });
    } else {
      const delight = Math.max(happy ? 1 : 0, wave);
      this._faceMaterials.normal.opacity = mix(this._faceMaterials.normal.opacity, happy ? 0 : 1, blend);
      this._faceMaterials.happy.opacity = mix(this._faceMaterials.happy.opacity, happy ? 1 : 0, blend);
      this._faceMaterials.cheeks.opacity = mix(this._faceMaterials.cheeks.opacity, delight, blend);
      this._eyes.forEach((eye, i) => {
        eye.visible = this._faceMaterials.normal.opacity > 0.001;
        eye.userData.baseHeight = mix(
          eye.userData.baseHeight ?? 1,
          1 - delight * 0.55 - (thinking && i === 1 ? 0.35 * (1 - delight) : 0),
          blend
        );
        eye.scale.y = eye.userData.baseHeight * blink;
        eye.rotation.z = mix(eye.rotation.z, delight * (i === 0 ? -0.22 : 0.22), blend);
        eye.position.y = mix(eye.position.y, 0.06 + (thinking ? Math.sin(t * 2 + i) * 0.035 : 0), blend);
      });
      this._happyEyes.forEach((eye) => {
        eye.visible = this._faceMaterials.happy.opacity > 0.001;
      });
      this._grin.visible = this._faceMaterials.happy.opacity > 0.001;
      this._smile.visible = this._faceMaterials.normal.opacity > 0.001;
      this._smile.scale.set(
        mix(this._smile.scale.x, 1 + delight * 0.2 - (thinking ? 0.45 * (1 - delight) : 0), blend),
        mix(this._smile.scale.y, thinking ? 0.6 : 1, blend),
        1
      );
      this._cheeks.forEach((cheek) => {
        cheek.visible = this._faceMaterials.cheeks.opacity > 0.001;
      });
      this._antenna.scale.setScalar(
        mix(this._antenna.scale.x, 1 + (thinking ? Math.sin(t * 3) * 0.075 : happy ? Math.sin(t * 7) * 0.07 : 0), blend)
      );
    }
    this._shadow.scale.setScalar(mix(this._shadow.scale.x, 1 - bob * 0.45 - hop * 0.35, blend));
    this._shadow.material.opacity = mix(this._shadow.material.opacity, keyholder ? 0.28 : 1 - hop * 0.6, blend);
    this._fxOpacity = mix(this._fxOpacity, celebrating ? 1 : 0, blend);
    this._confettiMaterials.forEach((mat) => {
      mat.opacity = this._fxOpacity;
    });
    this._celebrationFX.visible = burstActive && this._fxOpacity > 0.001;
    if (this._celebrationFX.visible) {
      this._confetti.forEach((piece) => {
        const flight = piece.userData.flight;
        const age = c - 0.22 - flight.delay;
        piece.visible = age >= 0 && age < 2.8;
        piece.position.set(flight.side * 1.04 + flight.vx * age, 0.42 + flight.vy * age - 1.15 * age * age, flight.z);
        piece.rotation.set(age * flight.spin, age * 3, age * flight.spin * flight.side);
        piece.scale.setScalar(clamp((2.8 - age) / 0.45, 0, 1));
      });
      this._sparkles.forEach((star, i) => {
        star.scale.setScalar(
          Math.max(0, Math.sin(Math.min(c / 1.9, 1) * Math.PI)) * (1.05 + Math.sin(c * 7 + i) * 0.18)
        );
        star.rotation.z = c * (i === 0 ? 0.6 : -0.6);
      });
    }
    this._renderer.render(this._scene, this._camera);
    this._poseReady = true;
    // DOM-visible diagnostics make integration checks possible without a scene hook.
    this.dataset.gaze = `${this._look.x.toFixed(2)},${this._look.y.toFixed(2)}`;
    this.dataset.expression = this._state;
    this.dataset.motion = staticPose ? "still" : "animated";
    this.dataset.action = action > 0.01 ? (wizard ? "guard" : "jingle") : "none";
  }

  _tick = (now) => {
    this._frame = 0;
    if (!this._canAnimate()) return;
    const delta = this._last ? Math.min((now - this._last) / 1000, 0.05) : 0;
    this._last = now;
    this._time += delta;
    const attentive = (this._follow || this._pointer.input === "keyboard") && now - this._pointer.at < 3000;
    const smoothing = 1 - Math.exp(-delta * 5.8);
    this._look.x += ((attentive ? this._pointer.x : 0) - this._look.x) * smoothing;
    this._look.y += ((attentive ? this._pointer.y : 0) - this._look.y) * smoothing;
    if (this._time > this._nextBlink && !this._rig?.quirk?.type) {
      this._blinkStart = this._time;
      this._nextBlink = this._time + (2.7 + Math.random() * 3.1) * (this._character === "keyholder" ? 2 : 1);
    }
    this._render(delta);
    this._frame = requestAnimationFrame(this._tick);
  };

  _canAnimate() {
    return (
      this.isConnected &&
      this.dataset.renderer === "webgl" &&
      !this._paused &&
      !this._motion.matches &&
      !document.hidden &&
      this._visible
    );
  }

  _syncLoop() {
    if (this._pauseButton) {
      this._pauseButton.hidden =
        !this.hasAttribute("controls") || this._motion.matches || this.dataset.renderer !== "webgl";
      this._pauseButton.setAttribute("aria-pressed", String(this._paused));
    }
    cancelAnimationFrame(this._frame);
    this._frame = 0;
    this._last = 0;
    if (this._paused || this._motion.matches) {
      this._look.x = this._look.y = 0;
      this._pointer.at = -Infinity;
      this._waveStart = -100;
      this._celebrateStart = -100;
      this._actionStart = -100;
    }
    this._render();
    if (this._canAnimate()) this._frame = requestAnimationFrame(this._tick);
  }

  setState(state) {
    if (!["idle", "thinking", "happy"].includes(state)) throw new RangeError("State must be idle, thinking, or happy.");
    this._state = state;
    if (state === "happy" && this._time - this._celebrateStart >= CELEBRATION_DURATION) {
      this._celebrateStart = this._motion && !this._motion.matches && !this._paused ? this._time : -100;
    }
    this._render();
  }
  wave() {
    if (!this.isConnected || this.dataset.renderer !== "webgl" || this._paused) return false;
    const greetingTime = performance.now();
    if (this._motion.matches) {
      if (greetingTime < this._quietUntil) return false;
    } else {
      // Ignore clicks throughout the whole wave and return-to-rest; never queue them.
      if (
        this._time - this._waveStart < WAVE_DURATION + WAVE_SETTLE ||
        this._time - this._actionStart < ACTION_DURATION + WAVE_SETTLE
      )
        return false;
      this._waveStart = this._time;
    }
    this._lastGreetingAt = greetingTime;
    this._quietUntil = greetingTime + (WAVE_DURATION + WAVE_SETTLE) * 1000;
    this.dispatchEvent(new CustomEvent("avatar-wave", { bubbles: true, composed: true }));
    this._render();
    return true;
  }
  performAction() {
    if (this._character === "robot" || !this.isConnected || this.dataset.renderer !== "webgl" || this._paused)
      return false;
    const now = performance.now();
    if (this._motion.matches) {
      if (now < this._quietUntil) return false;
    } else {
      if (
        this._time - this._actionStart < ACTION_DURATION + WAVE_SETTLE ||
        this._time - this._waveStart < WAVE_DURATION + WAVE_SETTLE
      )
        return false;
      this._actionStart = this._time;
    }
    this._lastGreetingAt = now;
    this._quietUntil = now + (ACTION_DURATION + WAVE_SETTLE) * 1000;
    this.dispatchEvent(
      new CustomEvent("avatar-action", {
        detail: { action: this._character === "wizard" ? "guard" : "jingle" },
        bubbles: true,
        composed: true
      })
    );
    this._render();
    return true;
  }
  set followPointer(value) {
    this._follow = Boolean(value);
    if (!this._follow) this._pointer.at = -Infinity;
  }
  get followPointer() {
    return this._follow;
  }
  set paused(value) {
    this._paused = Boolean(value);
    if (this._renderer) this._syncLoop();
  }
  get paused() {
    return this._paused;
  }

  _showFallback(error) {
    cancelAnimationFrame(this._frame);
    this.dataset.renderer = "fallback";
    if (this.querySelector("img")) {
      this._fallback.setAttribute("hidden", "");
      this.shadowRoot.querySelector("slot").removeAttribute("hidden");
    } else {
      this._fallback.removeAttribute("hidden");
    }
    if (this._renderer) this._renderer.domElement.hidden = true;
    this._button.disabled = true;
    this._button.setAttribute("aria-label", this.getAttribute("label") || `${this._character}. Static illustration.`);
    if (this._pauseButton) this._pauseButton.hidden = true;
    queueMicrotask(() =>
      this.dispatchEvent(
        new CustomEvent("avatar-error", { detail: { message: error.message }, bubbles: true, composed: true })
      )
    );
  }

  disconnectedCallback() {
    cancelAnimationFrame(this._frame);
    this._abort?.abort();
    this._resizeObserver?.disconnect();
    this._intersectionObserver?.disconnect();
    const textures = new Set();
    const materials = new Set();
    const geometries = new Set();
    this._scene?.traverse((object) => {
      if (object.geometry) geometries.add(object.geometry);
      if (object.material) materials.add(object.material);
    });
    geometries.forEach((geometry) => geometry.dispose());
    materials.forEach((material) => {
      if (material.map) textures.add(material.map);
      if (material.gradientMap) textures.add(material.gradientMap);
      material.dispose();
    });
    textures.forEach((texture) => texture.dispose());
    this._renderer?.dispose();
    // Switching companions repeatedly must not retain inactive GPU contexts.
    this._renderer?.forceContextLoss();
    this._renderer = null;
    this._scene = null;
  }
}

if (!customElements.get("scormiq-avatar")) customElements.define("scormiq-avatar", ScormiqAvatar);
