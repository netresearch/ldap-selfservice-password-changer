import { animationRandom } from "./animation-random.js";
import * as THREE from "./vendor/three.module.js";

// Both gophers are original, procedural meshes interpreted from the supplied images.
// They use the same lifecycle, motion clock and input guards as the robot.
const mix = (a, b, t) => a + (b - a) * t;
const material = (color, roughness = 0.7, metalness = 0) =>
  new THREE.MeshStandardMaterial({ color, roughness, metalness });
function mesh(parent, geometry, mat, x = 0, y = 0, z = 0) {
  const object = new THREE.Mesh(geometry, mat);
  object.position.set(x, y, z);
  parent.add(object);
  return object;
}
function group(parent, x = 0, y = 0, z = 0) {
  const object = new THREE.Group();
  object.position.set(x, y, z);
  parent.add(object);
  return object;
}
function ellipsoid(parent, mat, x, y, z, sx, sy, sz) {
  const object = mesh(parent, new THREE.SphereGeometry(1, 36, 28), mat, x, y, z);
  object.scale.set(sx, sy, sz);
  return object;
}
function curve(parent, mat, points, radius = 0.012) {
  return mesh(
    parent,
    new THREE.TubeGeometry(
      new THREE.CatmullRomCurve3(points.map((p) => new THREE.Vector3(...p))),
      24,
      radius,
      6,
      false
    ),
    mat
  );
}
function leaf(parent, mat, x, y, z, width, length, angle = 0) {
  const shape = new THREE.Shape();
  shape.moveTo(-width / 2, 0);
  shape.bezierCurveTo(-width * 0.5, length * 0.45, width * 0.32, length * 0.6, 0, length);
  shape.bezierCurveTo(width * 0.18, length * 0.68, width * 0.65, length * 0.22, width / 2, 0);
  shape.closePath();
  const geometry = new THREE.ExtrudeGeometry(shape, {
    depth: 0.035,
    bevelEnabled: true,
    bevelThickness: 0.028,
    bevelSize: 0.028,
    bevelSegments: 2,
    steps: 1,
    curveSegments: 10
  });
  const object = mesh(parent, geometry, mat, x, y, z);
  object.rotation.z = angle;
  return object;
}
function paw(parent, fur, pads, side, fingers = true) {
  ellipsoid(parent, fur, side * 0.04, -0.4, 0.04, 0.19, 0.31, 0.2).rotation.z = side * -0.18;
  ellipsoid(parent, pads, side * 0.07, -0.64, 0.09, 0.18, 0.17, 0.16);
  if (fingers)
    for (let i = 0; i < 3; i++) {
      ellipsoid(parent, pads, -0.08 + i * 0.085, -0.68, 0.2, 0.044, 0.09, 0.049);
    }
}

function addSquintMorph(pupil) {
  // Deform the solid pupil into a curved closed-eye stroke. No alpha blending.
  const geometry = pupil.geometry;
  const positions = geometry.attributes.position;
  const closed = new Float32Array(positions.array.length);
  for (let i = 0; i < positions.count; i++) {
    const x = positions.getX(i);
    closed[i * 3] = (x * 0.23) / pupil.scale.x;
    closed[i * 3 + 1] = (0.1 * (1 - x * x) + positions.getY(i) * 0.026) / pupil.scale.y;
    closed[i * 3 + 2] = (positions.getZ(i) * 0.023) / pupil.scale.z;
  }
  const target = new THREE.Float32BufferAttribute(closed, 3);
  const closedGeometry = geometry.clone();
  closedGeometry.setAttribute("position", target);
  closedGeometry.computeVertexNormals();
  geometry.morphAttributes.position = [target];
  geometry.morphAttributes.normal = [closedGeometry.attributes.normal.clone()];
  closedGeometry.dispose();
  pupil.updateMorphTargets();
}

function buildKeys(parent, silver, darkSilver) {
  const keys = group(parent, 0.05, -0.44, 0.3);
  keys.name = "silver-keyring";
  mesh(keys, new THREE.TorusGeometry(0.28, 0.032, 12, 64), silver, 0, -0.1, 0);
  mesh(keys, new THREE.TorusGeometry(0.26, 0.013, 8, 64), darkSilver, 0.014, -0.11, -0.012);
  const pieces = [-0.62, -0.25, 0.18, 0.56].map((angle, i) => {
    const key = group(keys, (i - 1.5) * 0.032, -0.31, 0.055 + i * 0.028);
    key.rotation.z = angle;
    mesh(key, new THREE.TorusGeometry(0.105, 0.034, 10, 28), silver);
    mesh(key, new THREE.BoxGeometry(0.073, 0.69 + i * 0.04, 0.055), silver, 0, -0.44, 0);
    mesh(key, new THREE.BoxGeometry(0.02, 0.48, 0.012), darkSilver, -0.009, -0.45, 0.034);
    for (let tooth = 0; tooth < 3; tooth++) {
      mesh(key, new THREE.BoxGeometry(0.13 + (tooth % 2) * 0.04, 0.066, 0.057), silver, 0.043, -0.67 + tooth * 0.1, 0);
    }
    key.userData.rest = angle;
    return key;
  });
  // A tiny gopher medallion hangs inside the ring, as in the keyholder reference.
  ellipsoid(keys, silver, 0, -0.11, 0.065, 0.115, 0.145, 0.025);
  for (const side of [-1, 1]) {
    ellipsoid(keys, silver, side * 0.076, -0.015, 0.06, 0.04, 0.04, 0.025);
    ellipsoid(keys, darkSilver, side * 0.041, -0.081, 0.09, 0.013, 0.02, 0.009);
  }
  return { keys, pieces };
}

function buildStaff(parent, wood, stone, leather) {
  // Leather grip midpoint is the wrist socket, shared by the staff and curled fingers.
  const staff = group(parent, -0.013, -0.04, 0);
  staff.name = "crooked-wizard-staff";
  curve(
    staff,
    wood,
    [
      [0, -0.86, 0],
      [0.06, -0.48, 0.025],
      [0.015, 0.05, 0],
      [-0.025, 0.72, -0.025],
      [0.07, 1.48, 0],
      [0.02, 1.86, 0.015]
    ],
    0.073
  );
  const grain = material("#b28154");
  for (const side of [-1, 1])
    curve(
      staff,
      grain,
      [
        [side * 0.037, -0.82, 0.053],
        [0.06 + side * 0.035, -0.47, 0.074],
        [side * 0.038, 0.06, 0.065],
        [-0.02 + side * 0.037, 0.73, 0.033],
        [0.07 + side * 0.036, 1.49, 0.06]
      ],
      0.008
    );
  for (const [x, y] of [
    [0.055, -0.48],
    [-0.025, 0.77],
    [0.06, 1.45]
  ])
    ellipsoid(staff, wood, x, y, 0, 0.1, 0.15, 0.09);
  curve(
    staff,
    wood,
    [
      [0.04, 1.56, 0],
      [-0.2, 1.81, 0.025],
      [-0.2, 2.12, 0.02],
      [-0.12, 2.29, 0.01]
    ],
    0.065
  );
  curve(
    staff,
    wood,
    [
      [0.04, 1.59, -0.02],
      [0.24, 1.87, -0.035],
      [0.26, 2.19, -0.03],
      [0.17, 2.4, -0.02]
    ],
    0.06
  );
  const crystal = mesh(staff, new THREE.OctahedronGeometry(0.25), stone, 0.035, 2.09, 0.01);
  crystal.scale.set(0.72, 1.65, 0.8);
  crystal.rotation.z = -0.1;
  crystal.name = "staff-crystal";
  curve(
    staff,
    new THREE.MeshBasicMaterial({ color: "#e0fff5" }),
    [
      [-0.045, 1.94, 0.15],
      [-0.025, 2.12, 0.17],
      [0.035, 2.41, 0.04]
    ],
    0.011
  );
  const brass = material("#bba578");
  for (const y of [-0.14, 0.23, 1.57]) {
    const collar = mesh(staff, new THREE.TorusGeometry(0.077, 0.018, 8, 32), brass, 0.02, y, 0);
    collar.rotation.x = Math.PI / 2;
  }
  for (let i = 0; i < 9; i++) {
    const wrap = mesh(staff, new THREE.TorusGeometry(0.079, 0.014, 6, 20), leather, 0.013, -0.11 + i * 0.037, 0);
    wrap.rotation.x = Math.PI / 2;
  }
  return staff;
}

function buildStaffGrip(arm, fur, pads) {
  // A bent forearm leads into a vertical fist, rather than an open downward-facing paw.
  const forearm = ellipsoid(arm, fur, -0.1, -0.37, 0.12, 0.19, 0.25, 0.2);
  forearm.rotation.z = -0.42;
  const wrist = group(arm, -0.18, -0.52, 0.21);
  wrist.name = "staff-gripping-wrist";
  ellipsoid(wrist, pads, 0.105, -0.005, -0.04, 0.16, 0.19, 0.13).name = "gripping-palm";
  for (let i = 0; i < 3; i++) {
    const y = 0.095 - i * 0.087;
    // Curl around the front and outer side of the leather-wrapped shaft.
    curve(
      wrist,
      pads,
      [
        [0.16, y, 0.055],
        [0.07, y, 0.108],
        [-0.035, y, 0.11],
        [-0.105, y, 0.06],
        [-0.1, y, -0.025]
      ],
      0.046
    ).name = "curled-finger";
  }
  const thumb = ellipsoid(wrist, pads, 0.105, 0.125, 0.115, 0.065, 0.125, 0.062);
  thumb.rotation.z = -0.7;
  thumb.name = "opposing-thumb";
  return wrist;
}

function buildWizardHat(head) {
  const hat = group(head);
  hat.name = "bent-wizard-hat";
  const felt = material("#536c7b");
  const brim = ellipsoid(hat, felt, 0, 0.91, -0.06, 1.32, 0.075, 0.86);
  brim.rotation.z = -0.055;
  // The crown bends backward in depth, then flops to one side with a drooping tip.
  const spine = new THREE.CatmullRomCurve3([
    new THREE.Vector3(0, 0.94, -0.08),
    new THREE.Vector3(-0.05, 1.27, -0.18),
    new THREE.Vector3(0.01, 1.65, -0.38),
    new THREE.Vector3(0.23, 1.95, -0.65),
    new THREE.Vector3(0.54, 2.02, -0.97),
    new THREE.Vector3(0.85, 1.83, -1.24)
  ]);
  const segments = 32;
  const frames = spine.computeFrenetFrames(segments, false);
  const crown = new THREE.ConeGeometry(1, 1, 48, segments);
  const positions = crown.attributes.position;
  for (let i = 0; i < positions.count; i++) {
    const t = THREE.MathUtils.clamp(positions.getY(i) + 0.5, 0, 1);
    const ring = Math.round(t * segments);
    const radius = t < 1 ? 0.84 * Math.pow(1 - t, -0.08) : 0;
    const point = spine
      .getPointAt(t)
      .addScaledVector(frames.normals[ring], positions.getX(i) * radius)
      .addScaledVector(frames.binormals[ring], -positions.getZ(i) * radius * 0.83);
    positions.setXYZ(i, point.x, point.y, point.z);
  }
  crown.computeVertexNormals();
  mesh(hat, crown, felt).name = "folded-hat-crown";
  const band = mesh(hat, new THREE.TorusGeometry(0.765, 0.045, 10, 64), material("#344957"), 0, 1.02, -0.1);
  band.rotation.x = Math.PI / 2;
  band.scale.y = 0.81;
  return hat;
}

export function applyCelStyle(root, scene) {
  // A single key light and nearest-sampled ramp give crisp, genuinely stepped lighting.
  const ramp = new THREE.DataTexture(new Uint8Array([58, 123, 195, 255]), 4, 1, THREE.RedFormat);
  ramp.minFilter = ramp.magFilter = THREE.NearestFilter;
  ramp.generateMipmaps = false;
  ramp.needsUpdate = true;
  let keyFound = false;
  scene.traverse((object) => {
    if (object.isHemisphereLight) object.intensity = 0;
    if (object.isDirectionalLight) {
      object.intensity = keyFound ? 0 : Math.PI;
      keyFound = true;
    }
  });
  const surfaces = [];
  root.traverse((object) => {
    if (object.isMesh) surfaces.push(object);
  });
  const materials = new Map();
  surfaces.forEach((object) => {
    const previous = object.material;
    if (previous.isMeshBasicMaterial) return; // Preserve eye glints, the robot display, and its logo.
    if (!materials.has(previous))
      materials.set(
        previous,
        new THREE.MeshToonMaterial({
          color: previous.color,
          emissive: previous.color.clone().multiplyScalar(0.065),
          gradientMap: ramp,
          toneMapped: false,
          transparent: previous.transparent,
          opacity: previous.opacity,
          depthWrite: previous.depthWrite
        })
      );
    object.material = materials.get(previous);
    // Ink existing strokes only once; dense contour shells muddy tiny facial details.
    if (
      object.geometry.type === "TubeGeometry" ||
      previous.color.r + previous.color.g + previous.color.b < 0.16 ||
      Math.max(object.scale.x, object.scale.y, object.scale.z) < 0.17 ||
      previous.color.getHex() === 0xefaea7
    )
      return;
    const box = object.geometry.parameters;
    if (
      object.geometry.type === "BoxGeometry" &&
      box.widthSegments === 1 &&
      box.heightSegments === 1 &&
      box.depthSegments === 1
    ) {
      const edges = new THREE.LineSegments(
        new THREE.EdgesGeometry(object.geometry),
        new THREE.LineBasicMaterial({ color: "#465564", transparent: true, opacity: 0.75, toneMapped: false })
      );
      edges.name = "cel-hard-edges";
      object.add(edges);
      return;
    }
    const outline = new THREE.Mesh(
      object.geometry,
      new THREE.ShaderMaterial({
        uniforms: {
          ink: { value: new THREE.Color("#302735") },
          opacity: { value: previous.opacity },
          thickness: { value: object.geometry.type === "TorusGeometry" ? 0.005 : 0.012 }
        },
        side: THREE.BackSide,
        transparent: true,
        depthWrite: false,
        toneMapped: false,
        vertexShader: `uniform float thickness;
        void main() {
          vec4 p = modelViewMatrix * vec4(position, 1.0);
          p.xyz += normalize(normalMatrix * normal) * thickness;
          gl_Position = projectionMatrix * p;
        }`,
        fragmentShader: `uniform vec3 ink; uniform float opacity;
        void main() {
          gl_FragColor = vec4(ink, opacity);
          #include <colorspace_fragment>
        }`
      })
    );
    outline.name = "cel-ink";
    // View-space expansion keeps line weight consistent on flattened eyes and tiny keys.
    outline.onBeforeRender = () => {
      outline.material.uniforms.opacity.value = object.material.opacity;
    };
    object.add(outline);
  });
  materials.forEach((replacement, previous) => previous.dispose());
  return ramp;
}

function plushBody(width, height, depth, roundness, taper) {
  // A rounded barrel profile fills the flanks and flattens the base, unlike an ellipsoid.
  const geometry = new THREE.SphereGeometry(1, 56, 48);
  const positions = geometry.attributes.position;
  for (let i = 0; i < positions.count; i++) {
    const y = positions.getY(i);
    const radius = Math.sqrt(Math.max(0, 1 - y * y));
    const profile = Math.pow(Math.max(0, 1 - Math.abs(y) ** roundness), 1 / roundness);
    const factor = radius > 0.00001 ? profile / radius : 0;
    positions.setXYZ(
      i,
      positions.getX(i) * factor * (width - y * taper),
      y * height,
      positions.getZ(i) * factor * depth
    );
  }
  geometry.computeVertexNormals();
  return geometry;
}

export function buildGopher(avatar, character) {
  const wizard = character === "wizard";
  const root = avatar._robot;
  avatar._torso = group(root);
  avatar._head = group(root, 0, wizard ? 0.77 : 0.4, 0);
  avatar._face = group(avatar._head);
  const dark = material(wizard ? "#271c18" : "#372334", 0.58);
  const fur = material(wizard ? "#65412c" : "#73c4d0");
  const light = material(wizard ? "#bd9263" : "#b4e9ef");
  const cream = material(wizard ? "#dfbf8b" : "#f3debc");
  const white = material("#fffdf7", 0.28);
  const eyeBlack = material(wizard ? "#100d0c" : "#332030", wizard ? 0.08 : 0.4);
  const shine = new THREE.MeshBasicMaterial({ color: "#ffffff" });
  const pupilGroups = [];
  const eyeGroups = [];
  const pupilMeshes = [];
  const glints = [];
  const brows = [];
  let keys, pieces, staff, wrist, beard, cape;
  avatar._arms = [-1, 1].map((side) =>
    group(avatar._torso, side * (wizard ? 0.74 : 1.02), wizard ? -0.24 : -0.27, wizard ? 0.02 : 0.21)
  );
  if (!wizard) {
    // Short front paws belong to the barrel body and follow its gentle gaze/breathing.
    avatar._arms.forEach((arm) => avatar._head.add(arm));
    avatar._arms[0].position.set(-0.72, -0.5, 0.72);
    avatar._arms[1].position.set(0.59, -0.41, 0.8);
  } else avatar._arms[0].position.x = -0.86;
  avatar._feet = [-1, 1].map((side) => {
    const foot = group(avatar._torso, side * (wizard ? 0.49 : 0.65), -1.32, 0.12);
    if (!wizard) ellipsoid(foot, fur, 0, 0.16, -0.04, 0.18, 0.25, 0.22);
    ellipsoid(foot, wizard ? fur : dark, 0, -0.08, 0.1, wizard ? 0.27 : 0.28, 0.12, 0.35);
    if (wizard) for (let i = 0; i < 3; i++) ellipsoid(foot, light, -0.14 + i * 0.13, -0.075, 0.36, 0.075, 0.065, 0.12);
    return foot;
  });
  if (!wizard) {
    mesh(avatar._head, plushBody(1.23, 1.45, 0.74, 3.2, 0.09), fur, 0, -0.28, 0).name = "keyholder-body";
    mesh(avatar._head, plushBody(1.1, 1.19, 0.41, 2.8, 0.06), light, 0, -0.49, 0.34).name = "keyholder-belly";
    // Dark band behind the two big white eyes is a defining feature of this gopher.
    ellipsoid(avatar._head, dark, 0, 0.57, 0.53, 0.98, 0.28, 0.25);
    for (const side of [-1, 1]) {
      ellipsoid(avatar._head, dark, side * 0.87, 1.12, -0.01, 0.29, 0.29, 0.19);
      ellipsoid(avatar._head, fur, side * 0.87, 1.13, 0.07, 0.265, 0.26, 0.17);
      const arm = avatar._arms[side === -1 ? 0 : 1];
      ellipsoid(arm, fur, 0, -0.07, 0.02, 0.15, 0.17, 0.115);
      ellipsoid(arm, fur, -side * 0.025, -0.21, 0.105, 0.17, 0.135, 0.12).name = "little-front-paw";
      for (let i = 0; i < 2; i++)
        curve(
          arm,
          dark,
          [
            [-0.055 + i * 0.065, -0.26, 0.214],
            [-0.049 + i * 0.065, -0.29, 0.208]
          ],
          0.006
        );
      ellipsoid(avatar._head, material("#efaea7"), side * 0.7, 0.12, 0.78, 0.12, 0.052, 0.023);
    }
    ellipsoid(avatar._head, cream, 0, 0.49, 0.88, 0.105, 0.066, 0.055);
    curve(
      avatar._head,
      dark,
      [
        [0, 0.44, 0.87],
        [0, 0.35, 0.88],
        [-0.08, 0.31, 0.875]
      ],
      0.014
    );
    for (const side of [-1, 1]) {
      const tooth = group(avatar._head, side === -1 ? -0.065 : 0.063, side === -1 ? 0.298 : 0.302, 0.86);
      tooth.rotation.z = side === -1 ? -0.045 : 0.035;
      const width = side === -1 ? 0.054 : 0.055;
      const height = side === -1 ? 0.103 : 0.099;
      ellipsoid(tooth, dark, 0, -0.006, -0.012, width + 0.018, height + 0.018, 0.039);
      ellipsoid(tooth, white, 0, 0.003, 0.02, width, height, 0.035);
    }
    // Small asymmetric ink marks echo a hand-inked game character.
    for (let i = 0; i < 3; i++)
      curve(
        avatar._head,
        dark,
        [
          [-0.99 + i * 0.035, -0.69 - i * 0.08, 0.56],
          [-0.94 + i * 0.035, -0.75 - i * 0.08, 0.59]
        ],
        0.008
      );
    const tail = curve(
      avatar._torso,
      fur,
      [
        [0.7, -0.91, -0.2],
        [1.14, -0.73, -0.18],
        [1.28, -0.48, -0.14]
      ],
      0.1
    );
    tail.name = "tiny-gopher-tail";
    ({ keys, pieces } = buildKeys(avatar._arms[1], material("#c5d6de", 0.23, 0.76), material("#5c7c86", 0.4, 0.58)));
    keys.position.set(0.02, -0.29, 0.22);
    keys.scale.setScalar(0.85);
  } else {
    const tunic = material("#526c70", 0.95);
    const cloak = material("#392d2a", 1);
    ellipsoid(avatar._torso, fur, 0, -0.68, 0, 0.74, 0.69, 0.47);
    cape = ellipsoid(avatar._torso, cloak, -0.12, -0.51, -0.31, 0.91, 0.9, 0.3);
    cape.rotation.z = -0.14;
    ellipsoid(avatar._torso, tunic, 0, -0.53, 0.24, 0.68, 0.7, 0.38);
    // Visible folds give the cloth its own material language, distinct from fur.
    for (let i = 0; i < 4; i++)
      curve(
        avatar._torso,
        material(i % 2 ? "#647d7e" : "#42595d"),
        [
          [-0.47 + i * 0.29, -0.23, 0.52],
          [-0.4 + i * 0.24, -0.57, 0.61],
          [-0.34 + i * 0.24, -0.95, 0.47]
        ],
        0.012
      );
    ellipsoid(avatar._torso, cloak, 0, -1.0, 0.15, 0.65, 0.1, 0.45);
    mesh(avatar._torso, new THREE.BoxGeometry(0.18, 0.12, 0.05), material("#a4926b", 0.5, 0.35), 0.04, -1.0, 0.6);
    for (const side of [-1, 1]) {
      ellipsoid(avatar._torso, fur, side * 0.45, -1.13, -0.02, 0.27, 0.3, 0.29);
      const arm = avatar._arms[side === -1 ? 0 : 1];
      // Overlapping shoulder, sleeve, and upper arm keep the paw attached in every pose.
      ellipsoid(arm, cloak, 0, -0.035, 0.02, 0.26, 0.24, 0.25).name = "wizard-shoulder";
      ellipsoid(arm, fur, side * 0.025, -0.22, 0.03, 0.205, 0.31, 0.22).name = "wizard-upper-arm";
      if (side === -1) wrist = buildStaffGrip(arm, fur, light);
      else paw(arm, fur, light, side);
      ellipsoid(avatar._head, fur, side * 0.93, 0.15, 0, 0.23, 0.3, 0.19);
      ellipsoid(avatar._head, light, side * 0.95, 0.17, 0.13, 0.14, 0.2, 0.09);
    }
    ellipsoid(avatar._head, fur, 0, -0.06, 0, 1.12, 0.91, 0.71);
    ellipsoid(avatar._head, light, -0.08, 0.2, 0.18, 1.02, 0.76, 0.59);
    // Shaped tufts around the silhouette and brow, not a texture pasted on a sphere.
    const furColors = [light, cream, material("#b99770"), fur];
    for (let i = 0; i < 25; i++) {
      const angle = (i / 25) * Math.PI * 2;
      const x = Math.sin(angle),
        y = Math.cos(angle);
      leaf(
        avatar._head,
        furColors[i % 4],
        x * 0.93,
        y * 0.7,
        0.23 + (i % 3) * 0.025,
        0.15 + (i % 3) * 0.025,
        0.17 + (i % 4) * 0.04,
        -angle + 0.44
      );
    }
    const fringe = group(avatar._head);
    fringe.name = "shaggy-forehead-and-cheeks";
    for (let i = 0; i < 11; i++) {
      const x = -0.85 + i * 0.17;
      leaf(
        fringe,
        furColors[i % 3],
        x,
        0.83 - Math.abs(x) * 0.08,
        0.64 - Math.abs(x) * 0.1,
        0.23,
        0.2 + (i % 3) * 0.045,
        Math.PI + x * 0.42
      );
    }
    for (const side of [-1, 1])
      for (let i = 0; i < 6; i++) {
        leaf(
          fringe,
          furColors[i % 3],
          side * (0.86 - i * 0.012),
          0.08 - i * 0.1,
          0.57 + i * 0.026,
          0.18,
          0.29 + (i % 2) * 0.07,
          -side * (1.12 + i * 0.13)
        );
      }
    buildWizardHat(avatar._head);
    beard = group(avatar._head, 0, -0.31, 0.47);
    ellipsoid(beard, material("#c5a078"), 0, -0.16, 0.1, 0.74, 0.52, 0.38);
    for (const side of [-1, 1]) {
      ellipsoid(beard, light, side * 0.37, 0.02, 0.17, 0.49, 0.34, 0.34);
      for (let i = 0; i < 7; i++) {
        leaf(
          beard,
          furColors[i % 3],
          side * (0.33 + i * 0.061),
          0.12 - i * 0.065,
          0.37 - i * 0.018,
          0.12,
          0.3 + (i % 3) * 0.06,
          side * (1.8 + i * 0.11)
        );
      }
    }
    for (let i = 0; i < 9; i++)
      leaf(
        beard,
        furColors[i % 3],
        -0.45 + i * 0.11,
        -0.42 + Math.abs(i - 4) * 0.04,
        0.27,
        0.14,
        0.24 + (i % 3) * 0.03,
        Math.PI + (i - 4) * -0.13
      );
    // Fine, swept locks across the brow and cheeks soften the sculpted silhouette.
    for (let i = 0; i < 15; i++) {
      const x = -0.75 + (i % 8) * 0.21;
      const y = 0.62 + Math.floor(i / 8) * 0.12;
      curve(
        avatar._head,
        furColors[i % 3],
        [
          [x, y - 0.035, 0.68 - Math.abs(x) * 0.19],
          [x + 0.024, y + 0.015, 0.68 - Math.abs(x) * 0.19],
          [x + 0.035, y + 0.09, 0.64 - Math.abs(x) * 0.19]
        ],
        0.01
      );
    }
    for (const side of [-1, 1])
      for (let i = 0; i < 3; i++)
        curve(
          avatar._head,
          dark,
          [
            [side * 0.46, -0.17 - i * 0.05, 1.0],
            [side * 0.96, -0.09 - i * 0.1, 0.93],
            [side * (1.37 + (i % 2) * 0.1), 0.04 - i * 0.17, 0.79]
          ],
          0.008
        );
    const nose = ellipsoid(avatar._head, eyeBlack, 0, -0.065, 1.035, 0.19, 0.105, 0.12);
    nose.rotation.z = -0.025;
    ellipsoid(avatar._head, shine, -0.025, -0.029, 1.136, 0.08, 0.014, 0.012);
    curve(
      avatar._head,
      dark,
      [
        [0, -0.15, 1.075],
        [0, -0.31, 1.058],
        [-0.04, -0.37, 1.019]
      ],
      0.022
    );
    staff = buildStaff(wrist, material("#67422a"), material("#829c91", 0.52, 0.35), material("#34241d"));
  }
  const eyeY = wizard ? 0.31 : 0.62;
  const eyeZ = wizard ? 0.71 : 0.74;
  const eyeX = wizard ? 0.49 : 0.6;
  for (const side of [-1, 1]) {
    if (wizard) ellipsoid(avatar._head, material("#94704c"), side * eyeX, eyeY, eyeZ - 0.09, 0.405, 0.48, 0.2);
    const eye = group(avatar._head, side * eyeX, eyeY, eyeZ);
    ellipsoid(eye, dark, 0, 0, -0.02, wizard ? 0.37 : 0.45, wizard ? 0.435 : 0.45, 0.22);
    ellipsoid(eye, white, 0, 0.004, 0.021, wizard ? 0.324 : 0.432, wizard ? 0.383 : 0.432, 0.216);
    const pupil = group(eye, wizard ? side * -0.015 : side * 0.095, wizard ? -0.026 : -0.02, wizard ? 0.19 : 0.232);
    pupil.userData.gaze = { x: pupil.position.x, y: pupil.position.y };
    const pupilMesh = ellipsoid(
      pupil,
      eyeBlack,
      0,
      0,
      0,
      wizard ? 0.265 : 0.13,
      wizard ? 0.33 : 0.13,
      wizard ? 0.126 : 0.043
    );
    addSquintMorph(pupilMesh);
    pupilMeshes.push(pupilMesh);
    const highlights = [];
    if (wizard) {
      highlights.push(ellipsoid(pupil, shine, -0.079, 0.123, 0.106, 0.081, 0.089, 0.026));
      highlights.push(ellipsoid(pupil, shine, 0.07, -0.18, 0.111, 0.055, 0.025, 0.016));
    } else {
      highlights.push(ellipsoid(pupil, shine, -0.032, 0.04, 0.039, 0.025, 0.028, 0.01));
    }
    highlights.forEach((highlight) => {
      highlight.userData.openScale = highlight.scale.clone();
    });
    glints.push(highlights);
    eye.traverse((object) => {
      if (!object.isMesh) return;
      object.material = object.material.clone();
      object.material.transparent = false;
    });
    eyeGroups.push(eye);
    pupilGroups.push(pupil);
    if (wizard) {
      const brow = group(avatar._head, side * eyeX, 0.73, 0.65);
      for (let i = 0; i < 4; i++) leaf(brow, cream, -0.21 + i * 0.13, 0, 0, 0.12, 0.14, side * -0.5);
      brows.push(brow);
    }
  }
  const mouth = ellipsoid(
    avatar._head,
    dark,
    0,
    wizard ? -0.49 : 0.21,
    wizard ? 0.971 : 0.825,
    wizard ? 0.14 : 0.11,
    wizard ? 0.14 : 0.06,
    0.029
  );
  const tongue = ellipsoid(mouth, material("#c1847b"), 0, -0.38, 0.72, 0.55, 0.22, 0.42);
  tongue.name = "little-tongue";
  const glow = new THREE.MeshBasicMaterial({
    color: "#38a7b4",
    transparent: true,
    opacity: 0,
    side: THREE.DoubleSide,
    depthWrite: false,
    toneMapped: false
  });
  const halo = mesh(avatar._scene, new THREE.RingGeometry(0.73, 0.77, 64), glow, -1.23, -1.57, 0.15);
  halo.rotation.x = -Math.PI / 2;
  halo.name = "staff-ward";
  halo.visible = false;
  const shockwaves = wizard
    ? [0, 1].map(() => {
        const ring = mesh(avatar._scene, new THREE.RingGeometry(0.94, 1, 64), glow.clone(), 0, -1.51, 0.1);
        ring.rotation.x = -Math.PI / 2;
        ring.visible = false;
        return ring;
      })
    : [];
  const sparks = [];
  let shield;
  if (wizard) {
    const shieldMaterial = glow.clone();
    shield = mesh(avatar._scene, new THREE.RingGeometry(1.43, 1.455, 64), shieldMaterial, 0, 0.12, -0.7);
    shield.name = "runic-ward";
    shield.visible = false;
    for (let i = 0; i < 12; i++) {
      const angle = (i / 12) * Math.PI * 2;
      const rune = mesh(
        shield,
        new THREE.BoxGeometry(0.035, 0.12, 0.012),
        shieldMaterial,
        Math.sin(angle) * 1.36,
        Math.cos(angle) * 1.36,
        0
      );
      rune.rotation.z = -angle;
    }
    const shard = new THREE.OctahedronGeometry(0.035);
    const sparkMaterial = new THREE.MeshBasicMaterial({
      color: "#e3ad46",
      transparent: true,
      opacity: 0,
      depthWrite: false,
      toneMapped: false
    });
    for (let i = 0; i < 12; i++) {
      const spark = mesh(avatar._scene, shard, sparkMaterial);
      spark.visible = false;
      sparks.push(spark);
    }
  }
  avatar._rig = {
    wizard,
    eyes: eyeGroups,
    pupils: pupilGroups,
    pupilMeshes,
    glints,
    brows,
    mouth,
    keys,
    pieces,
    staff,
    wrist,
    beard,
    cape,
    halo,
    shockwaves,
    sparks,
    shield,
    contact: new THREE.Vector3(),
    joy: 0
  };
  if (!wizard) {
    avatar._rig.quirk = {
      nextAt: avatar._time + 8 + animationRandom() * 6,
      start: -100,
      type: null,
      eye: 0,
      driftDuration: 0.72,
      returnAt: 0.96,
      dx: 0,
      dy: 0,
      x: 0,
      y: 0,
      wink: 0
    };
    // Put the bean's pivot at its base so gaze and breathing never lift its body off its feet.
    avatar._head.children.forEach((child) => {
      child.position.y += 1.72;
    });
    avatar._head.position.y = -1.32;
  }
  avatar.dataset.character = character;
}

function updateEyeQuirk(avatar, blend, staticPose, quietIdle, blink) {
  const quirk = avatar._rig.quirk;
  if (!quirk) return null;
  const now = avatar._time;
  if (staticPose) {
    quirk.type = null;
    quirk.x = quirk.y = quirk.wink = 0;
    quirk.nextAt = Math.max(quirk.nextAt, now + 18);
  } else if (blend > 0) {
    const finished = quirk.type && now - quirk.start >= (quirk.type === "drift" ? quirk.returnAt + 0.36 : 0.48);
    if (quirk.type && (!quietIdle || finished)) {
      quirk.type = null;
      quirk.nextAt = now + 18 + animationRandom() * 16;
      avatar._nextBlink = Math.max(avatar._nextBlink, now + 0.5);
    }
    if (!quietIdle) quirk.nextAt = Math.max(quirk.nextAt, now + 4);
    if (quietIdle && !quirk.type && now >= quirk.nextAt && blink === 1) {
      quirk.type = animationRandom() < 0.65 ? "drift" : "wink";
      quirk.eye = animationRandom() < 0.5 ? 0 : 1;
      quirk.start = now;
      quirk.dx = (quirk.eye === 0 ? -1 : 1) * (0.035 + animationRandom() * 0.025);
      quirk.dy = (animationRandom() - 0.5) * 0.024;
      quirk.driftDuration = 0.72 + Math.pow(animationRandom(), 2) * 3.2;
      quirk.returnAt = quirk.driftDuration + 0.24;
    }
  }
  const age = now - quirk.start;
  const drift =
    quirk.type === "drift"
      ? THREE.MathUtils.smoothstep(age, 0, quirk.driftDuration) *
        (1 - THREE.MathUtils.smoothstep(age, quirk.returnAt, quirk.returnAt + 0.12))
      : 0;
  const wink =
    quirk.type === "wink"
      ? THREE.MathUtils.smoothstep(age, 0, 0.09) * (1 - THREE.MathUtils.smoothstep(age, 0.14, 0.31))
      : 0;
  // Return quickly without changing the underlying pointer-tracking gaze.
  const quickBlend = 1 - Math.pow(1 - blend, 36 / 11);
  quirk.x = mix(quirk.x, drift * quirk.dx, quickBlend);
  quirk.y = mix(quirk.y, drift * quirk.dy, quickBlend);
  quirk.wink = mix(quirk.wink, wink, quickBlend);
  avatar.dataset.quirk = quirk.type || "none";
  return quirk;
}

export function animateGopher(
  avatar,
  { blend, t, blink, happy, thinking, wave, joy, action, staticPose, windup, brace, actionAge, quietIdle }
) {
  const rig = avatar._rig;
  const quirk = updateEyeQuirk(avatar, blend, staticPose, quietIdle, blink);
  rig.joy = mix(rig.joy, happy ? 1 - (rig.wizard ? action : 0) : wave * 0.24, blend);
  rig.eyes.forEach((eye, i) => {
    const winking = quirk && quirk.eye === i ? quirk.wink : 0;
    eye.scale.y = (1 - rig.joy * 0.22) * (quirk?.type ? 1 : blink) * (1 - winking * 0.94);
    rig.pupilMeshes[i].morphTargetInfluences[0] = rig.joy;
    rig.glints[i].forEach((highlight) => {
      highlight.scale.copy(highlight.userData.openScale).multiplyScalar(1 - rig.joy);
    });
    const pupil = rig.pupils[i];
    // The wizard's open pupils protrude farther; keep the flattened stroke in front of its sclera.
    pupil.position.z = rig.wizard ? 0.19 + rig.joy * 0.085 : 0.232;
    const side = i === 0 ? -1 : 1;
    const gaze = pupil.userData.gaze;
    gaze.x = mix(
      gaze.x,
      (rig.wizard ? side * -0.015 : side * 0.095) + avatar._look.x * (rig.wizard ? 0.035 : 0.075),
      blend
    );
    gaze.y = mix(gaze.y, -0.02 - avatar._look.y * (rig.wizard ? 0.028 : 0.1), blend);
    pupil.position.x = gaze.x + (quirk && quirk.eye === i ? quirk.x : 0);
    pupil.position.y = gaze.y + (quirk && quirk.eye === i ? quirk.y : 0);
  });
  const mouth = rig.mouth;
  mouth.scale.y = mix(mouth.scale.y, rig.wizard ? 0.14 + rig.joy * 0.1 + action * 0.09 : 0.04 + rig.joy * 0.1, blend);
  mouth.scale.x = mix(mouth.scale.x, rig.wizard ? 0.14 + rig.joy * 0.13 : 0.1 + rig.joy * 0.07, blend);
  rig.brows.forEach((brow, i) => {
    brow.rotation.z = mix(brow.rotation.z, (i === 0 ? 1 : -1) * (action * -0.22 + (thinking ? 0.12 : 0)), blend);
  });
  if (rig.keys) {
    const breath = 1 + Math.sin(t * 1.7) * 0.006;
    avatar._head.scale.set(
      mix(avatar._head.scale.x, 1 / Math.sqrt(breath), blend),
      mix(avatar._head.scale.y, breath, blend),
      1
    );
    // Counter-rotation lets the ring hang from the paw rather than rotating rigidly with it.
    rig.keys.rotation.z = mix(
      rig.keys.rotation.z,
      -avatar._arms[1].rotation.z + Math.sin(t * 2) * 0.04 + Math.sin(t * 22) * action * 0.22,
      blend
    );
    rig.pieces.forEach((key, i) => {
      key.rotation.z = mix(
        key.rotation.z,
        key.userData.rest + Math.sin(t * 2.3 + i * 0.6) * 0.035 + Math.sin(t * 19 + i * 0.7) * action * 0.18,
        blend
      );
    });
  }
  if (rig.staff) {
    // Hand and staff rotate together around the grip; all lifting comes from the arm.
    rig.wrist.rotation.z = mix(
      rig.wrist.rotation.z,
      -avatar._arms[0].rotation.z + 0.035 - windup * 0.18 - brace * 0.035,
      blend
    );
    rig.wrist.rotation.x = mix(rig.wrist.rotation.x, -avatar._arms[0].rotation.x - windup * 0.18, blend);
    rig.cape.rotation.x = mix(rig.cape.rotation.x, -brace * 0.21 - windup * 0.1, blend);
    rig.cape.rotation.z = mix(rig.cape.rotation.z, -0.14 - brace * 0.12, blend);
    avatar._robot.updateMatrixWorld(true);
    rig.staff.localToWorld(rig.contact.set(0, -0.86, 0));
    const impactAge = actionAge - 0.78;
    const pulse = !staticPose && impactAge >= 0 && impactAge < 1.2 ? Math.exp(-impactAge * 2.8) : 0;
    rig.halo.position.set(rig.contact.x, -1.5, rig.contact.z);
    rig.halo.material.opacity = mix(rig.halo.material.opacity, staticPose ? 0 : brace * 0.5 + pulse * 0.45, blend);
    rig.halo.visible = rig.halo.material.opacity > 0.005;
    rig.halo.scale.setScalar(1 + brace * 0.4);
    rig.shield.material.opacity = mix(rig.shield.material.opacity, staticPose ? 0 : brace * 0.48 + pulse * 0.4, blend);
    rig.shield.visible = rig.shield.material.opacity > 0.005;
    rig.shield.scale.setScalar(0.9 + brace * 0.2);
    rig.shield.rotation.z = -brace * 0.18;
    rig.shockwaves.forEach((ring, i) => {
      const age = impactAge - i * 0.13;
      const active = !staticPose && age >= 0 && age < 1.15;
      ring.material.opacity = active ? Math.sin((Math.PI * Math.min(age / 0.12, 1)) / 2) * (1 - age / 1.15) * 0.8 : 0;
      ring.visible = ring.material.opacity > 0.005;
      ring.position.x = rig.contact.x;
      ring.position.z = rig.contact.z;
      ring.scale.setScalar(0.24 + Math.max(0, age) * 2.8);
    });
    rig.sparks.forEach((spark, i) => {
      const age = Math.max(0, impactAge);
      const angle = (i / rig.sparks.length) * Math.PI * 2;
      spark.material.opacity = pulse;
      spark.visible = pulse > 0.005;
      spark.position.set(
        rig.contact.x + Math.cos(angle) * age * 1.7,
        -1.47 + age * (1.7 + (i % 3) * 0.25) - age * age * 1.9,
        rig.contact.z + Math.sin(angle) * age * 1.7
      );
      spark.rotation.z = angle + age * 4;
      spark.scale.setScalar(1 + pulse);
    });
    rig.beard.rotation.x = mix(rig.beard.rotation.x, Math.sin(t * 1.8) * 0.018 + joy * 0.04, blend);
  }
}

export function gopherFallback(character) {
  const wizard = character === "wizard";
  return `<svg class="fallback" viewBox="${wizard ? "0 -45 400 485" : "0 0 400 440"}" role="img" aria-label="${wizard ? "Furry wizard gopher with a backward-bent hat and crooked staff" : "Turquoise keyholder gopher with silver keys"}">
    <ellipse cx="200" cy="394" rx="112" ry="13" fill="#2f99a4" opacity=".12"/>
    ${wizard ? '<path d="M105 362Q70 226 120 204H275Q321 279 290 365Z" fill="#392d2a"/><ellipse cx="200" cy="294" rx="79" ry="92" fill="#526c70"/><path d="M62 388L72 271L68 210L78 103M78 112Q47 87 62 47M78 112Q105 84 94 44" fill="none" stroke="#67422a" stroke-width="9" stroke-linecap="round"/><path d="M79 31L92 73L78 101L65 73Z" fill="#829c91" stroke="#302735" stroke-width="2"/><path d="M79 31L78 101L65 73Z" fill="#b9d4c9"/><path d="M70 243L78 245M69 252L77 254M68 261L76 263" stroke="#34241d" stroke-width="4"/>' : ""}
    ${wizard ? '<ellipse cx="200" cy="175" rx="121" ry="113" fill="#dfc297"/>' : '<path d="M200 95C295 95 314 123 322 211C332 326 334 381 258 384H142C66 381 68 326 78 211C86 123 105 95 200 95Z" fill="#73c4d0" stroke="#302735" stroke-width="3"/>'}
    ${wizard ? '<path d="M83 209L64 258L104 246L95 284L149 269L176 297L213 278L245 292L269 262L304 267L290 228" fill="#c5a078"/>' : '<path d="M200 174C282 174 302 229 306 296C310 361 282 379 244 379H156C118 379 90 361 94 296C98 229 118 174 200 174Z" fill="#b2e8f1"/><circle cx="117" cy="109" r="26" fill="#69b6c4"/><circle cx="283" cy="109" r="26" fill="#69b6c4"/><path d="M99 166H302" stroke="#372334" stroke-width="30"/>'}
    ${wizard ? '<g fill="#dfc297" stroke="#76513d" stroke-width="2"><path d="M106 199L66 212L98 225L58 238L108 245M294 199L334 212L302 225L342 238L292 245"/><path d="M92 94L116 127L131 105L147 133L168 106L190 137L209 105L231 132L248 104L269 127L285 99Z"/></g><path d="M109 93Q120 2 191 -22Q258 -38 282 25Q252 5 247 37L280 98Z" fill="#536c7b" stroke="#302735" stroke-width="3"/><path d="M240 -10Q255 3 247 37L280 98L246 95Q205 28 240 -10Z" fill="#3d5363"/><path d="M112 82L275 86L280 100L108 97Z" fill="#344957"/><path d="M68 94Q174 76 316 101Q340 117 287 116L107 108Q59 108 68 94Z" fill="#536c7b" stroke="#302735" stroke-width="3"/>' : ""}
    <g fill="white" stroke="#372334" stroke-width="3"><circle cx="145" cy="166" r="43"/><circle cx="255" cy="166" r="43"/></g>
    <g fill="#271c23"><ellipse cx="${wizard ? 145 : 138}" cy="170" rx="${wizard ? 30 : 13}" ry="${wizard ? 34 : 13}"/><ellipse cx="${wizard ? 255 : 262}" cy="170" rx="${wizard ? 30 : 13}" ry="${wizard ? 34 : 13}"/></g>
    ${wizard ? "" : '<g fill="white"><circle cx="134" cy="166" r="3"/><circle cx="258" cy="166" r="3"/></g><g fill="#efaea7"><ellipse cx="132" cy="219" rx="12" ry="5"/><ellipse cx="268" cy="219" rx="12" ry="5"/></g>'}
    ${wizard ? '<g fill="white"><circle cx="134" cy="153" r="10"/><circle cx="244" cy="153" r="10"/></g><path d="M185 209Q200 200 215 209L200 222Z" fill="#271c23"/><path d="M165 221L64 208M167 229L58 232M236 221L334 208M235 229L338 234" stroke="#493529" stroke-width="2"/>' : '<g fill="#73c4d0" stroke="#302735" stroke-width="2"><path d="M117 251Q106 251 109 267Q103 280 117 282Q134 281 129 267Q131 251 117 251Z"/><path d="M273 235Q261 235 262 250Q255 263 270 266Q288 264 283 250Q289 237 273 235Z"/></g><ellipse cx="200" cy="180" rx="10" ry="7" fill="#f3debc"/><g fill="white" stroke="#372334" stroke-width="2"><ellipse cx="194" cy="199.5" rx="6" ry="10.3" transform="rotate(-3 194 199.5)"/><ellipse cx="207" cy="199" rx="6.1" ry="10" transform="rotate(2 207 199)"/></g><g stroke="#a0bac6" fill="none" stroke-width="8"><circle cx="279" cy="275" r="25"/><path d="M274 300L249 375L264 380M279 300L284 389L301 389M288 295L318 370L332 364"/></g>'}
    <ellipse cx="141" cy="382" rx="32" ry="13" fill="${wizard ? "#76513d" : "#372334"}"/><ellipse cx="259" cy="382" rx="32" ry="13" fill="${wizard ? "#76513d" : "#372334"}"/>
  </svg>`;
}
