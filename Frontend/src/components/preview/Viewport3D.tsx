import { useEffect, useRef } from "react";
import * as THREE from "three";
import { OrbitControls } from "three/addons/controls/OrbitControls.js";

interface CameraDef {
  topic: string;
  label: string;
  /** Position in vehicle frame (meters) */
  position: [number, number, number];
  /** Look direction (normalized) */
  lookAt: [number, number, number];
  /** Horizontal FOV in degrees */
  fovDeg: number;
  color: string;
}

interface Viewport3DProps {
  cameras?: CameraDef[];
}

const DEFAULT_CAMERAS: CameraDef[] = [
  {
    topic: "/camera/front/image_raw/compressed",
    label: "front",
    position: [0, 0.3, 1.5],
    lookAt: [0, -0.1, 3.5],
    fovDeg: 90,
    color: "#00b4d8",
  },
  {
    topic: "/camera/side/image_raw/compressed",
    label: "side",
    position: [0, 1.0, 1.2],
    lookAt: [3, 0.5, 1.0],
    fovDeg: 90,
    color: "#ff6b6b",
  },
  {
    topic: "/camera/down/image_raw/compressed",
    label: "down",
    position: [0, 0.5, 1.6],
    lookAt: [0, -2, 0.5],
    fovDeg: 90,
    color: "#51cf66",
  },
];

/** Build a wireframe frustum (4-sided pyramid) for a camera */
function createFrustum(
  pos: [number, number, number],
  lookAt: [number, number, number],
  fovDeg: number,
  color: string,
  nearDist = 0.2,
  farDist = 2.0,
): THREE.Group {
  const group = new THREE.Group();

  const origin = new THREE.Vector3(...pos);
  const target = new THREE.Vector3(...lookAt);
  const dir = target.clone().sub(origin).normalize();
  const nearCenter = origin.clone().add(dir.clone().multiplyScalar(nearDist));
  const farCenter = origin.clone().add(dir.clone().multiplyScalar(farDist));

  // Compute right/up axes for the near/far planes
  const up = new THREE.Vector3(0, 1, 0);
  if (Math.abs(dir.dot(up)) > 0.99) up.set(1, 0, 0);
  const right = dir.clone().cross(up).normalize();
  const camUp = right.clone().cross(dir).normalize();

  const fovRad = (fovDeg * Math.PI) / 180;
  const nearH = Math.tan(fovRad / 2) * nearDist;
  const farH = Math.tan(fovRad / 2) * farDist;
  // Aspect ratio ~1 for a square-ish frustum
  const nearW = nearH;
  const farW = farH;

  const ntl = nearCenter.clone().add(camUp.clone().multiplyScalar(nearH)).add(right.clone().multiplyScalar(-nearW));
  const ntr = nearCenter.clone().add(camUp.clone().multiplyScalar(nearH)).add(right.clone().multiplyScalar(nearW));
  const nbl = nearCenter.clone().add(camUp.clone().multiplyScalar(-nearH)).add(right.clone().multiplyScalar(-nearW));
  const nbr = nearCenter.clone().add(camUp.clone().multiplyScalar(-nearH)).add(right.clone().multiplyScalar(nearW));

  const ftl = farCenter.clone().add(camUp.clone().multiplyScalar(farH)).add(right.clone().multiplyScalar(-farW));
  const ftr = farCenter.clone().add(camUp.clone().multiplyScalar(farH)).add(right.clone().multiplyScalar(farW));
  const fbl = farCenter.clone().add(camUp.clone().multiplyScalar(-farH)).add(right.clone().multiplyScalar(-farW));
  const fbr = farCenter.clone().add(camUp.clone().multiplyScalar(-farH)).add(right.clone().multiplyScalar(farW));

  const lineMat = new THREE.LineBasicMaterial({ color, linewidth: 1, transparent: true, opacity: 0.7 });

  // Near rectangle
  group.add(new THREE.Line(
    new THREE.BufferGeometry().setFromPoints([ntl, ntr, nbr, nbl, ntl]),
    lineMat,
  ));
  // Far rectangle
  group.add(new THREE.Line(
    new THREE.BufferGeometry().setFromPoints([ftl, ftr, fbr, fbl, ftl]),
    lineMat,
  ));
  // Connecting lines
  [ntl, ntr, nbl, nbr].forEach((n, i) => {
    const f = [ftl, ftr, fbl, fbr][i];
    group.add(new THREE.Line(
      new THREE.BufferGeometry().setFromPoints([n, f]),
      lineMat,
    ));
  });

  // Semi-transparent planes
  const planeMat = new THREE.MeshBasicMaterial({
    color,
    transparent: true,
    opacity: 0.08,
    side: THREE.DoubleSide,
    depthWrite: false,
  });

  // Near plane
  const nearGeo = new THREE.BufferGeometry();
  const nearVerts = new Float32Array([
    ntl.x, ntl.y, ntl.z, ntr.x, ntr.y, ntr.z, nbl.x, nbl.y, nbl.z,
    ntr.x, ntr.y, ntr.z, nbr.x, nbr.y, nbr.z, nbl.x, nbl.y, nbl.z,
  ]);
  nearGeo.setAttribute("position", new THREE.BufferAttribute(nearVerts, 3));
  group.add(new THREE.Mesh(nearGeo, planeMat));

  // Far plane
  const farGeo = new THREE.BufferGeometry();
  const farVerts = new Float32Array([
    ftl.x, ftl.y, ftl.z, ftr.x, ftr.y, ftr.z, fbl.x, fbl.y, fbl.z,
    ftr.x, ftr.y, ftr.z, fbr.x, fbr.y, fbr.z, fbl.x, fbl.y, fbl.z,
  ]);
  farGeo.setAttribute("position", new THREE.BufferAttribute(farVerts, 3));
  group.add(new THREE.Mesh(farGeo, planeMat));

  return group;
}

/** Label sprite for a camera */
function createCameraLabel(text: string, position: THREE.Vector3, color: string): THREE.Sprite {
  const canvas = document.createElement("canvas");
  canvas.width = 128;
  canvas.height = 32;
  const ctx = canvas.getContext("2d")!;
  ctx.fillStyle = color;
  ctx.font = "bold 16px monospace";
  ctx.textAlign = "center";
  ctx.fillText(text, 64, 22);

  const tex = new THREE.CanvasTexture(canvas);
  tex.minFilter = THREE.LinearFilter;
  const mat = new THREE.SpriteMaterial({ map: tex, depthTest: false, depthWrite: false });
  const sprite = new THREE.Sprite(mat);
  sprite.position.copy(position);
  sprite.scale.set(1.2, 0.3, 1);
  return sprite;
}

export default function Viewport3D({ cameras = DEFAULT_CAMERAS }: Viewport3DProps) {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!containerRef.current) return;

    const container = containerRef.current;
    const w = container.clientWidth || 640;
    const h = container.clientHeight || 480;

    const scene = new THREE.Scene();
    scene.background = new THREE.Color(0x1a1a2e);

    const camera3d = new THREE.PerspectiveCamera(50, w / h, 0.1, 100);
    camera3d.position.set(4, 5, 6);
    camera3d.lookAt(0, 0.3, 1.2);

    const renderer = new THREE.WebGLRenderer({ antialias: true });
    renderer.setSize(w, h);
    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
    container.appendChild(renderer.domElement);

    const controls = new OrbitControls(camera3d, renderer.domElement);
    controls.enableDamping = true;
    controls.dampingFactor = 0.1;
    controls.target.set(0, 0.3, 1.2);

    // Lighting
    scene.add(new THREE.AmbientLight(0x404060));
    const dirLight = new THREE.DirectionalLight(0xffffff, 1.2);
    dirLight.position.set(5, 10, 7);
    scene.add(dirLight);
    const dl2 = new THREE.DirectionalLight(0x8888ff, 0.3);
    dl2.position.set(-3, -1, -5);
    scene.add(dl2);

    // Grid
    scene.add(new THREE.GridHelper(10, 20, 0x444488, 0x333366));
    scene.add(new THREE.AxesHelper(3));

    // Camera frustums + labels
    cameras.forEach((cam) => {
      const frustum = createFrustum(cam.position, cam.lookAt, cam.fovDeg, cam.color);
      scene.add(frustum);
      const labelPos = new THREE.Vector3(...cam.position).add(new THREE.Vector3(0, 0.4, 0));
      scene.add(createCameraLabel(cam.label, labelPos, cam.color));
    });

    // Animation loop
    let running = true;
    const animate = () => {
      if (!running) return;
      controls.update();
      renderer.render(scene, camera3d);
      requestAnimationFrame(animate);
    };
    animate();

    const onResize = () => {
      const w2 = container.clientWidth;
      const h2 = container.clientHeight;
      if (w2 === 0 || h2 === 0) return;
      camera3d.aspect = w2 / h2;
      camera3d.updateProjectionMatrix();
      renderer.setSize(w2, h2);
    };
    window.addEventListener("resize", onResize);

    return () => {
      running = false;
      window.removeEventListener("resize", onResize);
      renderer.dispose();
      if (container.contains(renderer.domElement)) {
        container.removeChild(renderer.domElement);
      }
    };
  }, [cameras]);

  return (
    <div
      ref={containerRef}
      style={{ width: "100%", height: "100%", minHeight: 300, borderRadius: "var(--radius-md)", overflow: "hidden" }}
    />
  );
}
