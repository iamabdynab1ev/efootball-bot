"use client";

import { useMemo, useRef } from "react";
import { Canvas, useFrame } from "@react-three/fiber";
import * as THREE from "three";
import type { MotionValue } from "framer-motion";

// Настоящая 3D-сцена интро «Финал. 90-я минута» (React Three Fiber).
// Всё процедурное — ни одного внешнего ассета: стадион-чаша с толпой и
// вспышками телефонов, поле с разметкой, прожекторы, мяч, ворота с сеткой.
// Скролл (MotionValue 0..1) двигает мяч по траектории и камеру по кривой —
// то самое «камера летит за мячом». Телефоны без WebGL получают CSS-фолбэк
// (решается на уровне страницы).

// ── Тайминги (синхронизированы с DOM-оверлеями страницы) ──────────────
const T = {
  runStart: 0.16,  // мяч начинает движение
  strike: 0.45,    // момент удара
  goal: 0.62,      // мяч в сетке
  celebrate: 0.8,  // отъезд камеры
} as const;

// ── Процедурные текстуры ──────────────────────────────────────────────

function makePitchTexture(): THREE.CanvasTexture {
  const c = document.createElement("canvas");
  c.width = c.height = 1024;
  const g = c.getContext("2d")!;
  // Полосы газона
  for (let i = 0; i < 8; i++) {
    g.fillStyle = i % 2 ? "#14401f" : "#0e3117";
    g.fillRect((i * 1024) / 8, 0, 1024 / 8, 1024);
  }
  // Разметка (правая половина поля: штрафная + центральный круг слева)
  g.strokeStyle = "rgba(255,255,255,0.55)";
  g.lineWidth = 6;
  g.strokeRect(20, 20, 984, 984);                    // границы
  g.beginPath(); g.arc(20, 512, 130, -Math.PI / 2, Math.PI / 2); g.stroke(); // центр. круг
  g.strokeRect(824, 312, 180, 400);                  // штрафная
  g.strokeRect(944, 412, 60, 200);                   // вратарская
  return new THREE.CanvasTexture(c);
}

function makeBallTexture(): THREE.CanvasTexture {
  const c = document.createElement("canvas");
  c.width = 512; c.height = 256;
  const g = c.getContext("2d")!;
  g.fillStyle = "#f4f4f6";
  g.fillRect(0, 0, 512, 256);
  // Пятиугольники + швы (равномерная сетка — на вращающейся сфере читается как классический мяч)
  const pent = (x: number, y: number, r: number) => {
    g.beginPath();
    for (let i = 0; i < 5; i++) {
      const a = (i / 5) * Math.PI * 2 - Math.PI / 2;
      g[i ? "lineTo" : "moveTo"](x + r * Math.cos(a), y + r * Math.sin(a));
    }
    g.closePath(); g.fill();
  };
  g.fillStyle = "#17181d";
  for (let row = 0; row < 3; row++) {
    for (let col = 0; col < 5; col++) {
      pent(col * 102 + (row % 2 ? 51 : 0) + 26, row * 86 + 42, 26);
    }
  }
  const tex = new THREE.CanvasTexture(c);
  tex.wrapS = tex.wrapT = THREE.RepeatWrapping;
  return tex;
}

function makeGlowTexture(): THREE.CanvasTexture {
  const c = document.createElement("canvas");
  c.width = c.height = 128;
  const g = c.getContext("2d")!;
  const grad = g.createRadialGradient(64, 64, 0, 64, 64, 64);
  grad.addColorStop(0, "rgba(255,255,255,1)");
  grad.addColorStop(0.35, "rgba(255,250,220,0.55)");
  grad.addColorStop(1, "rgba(255,255,255,0)");
  g.fillStyle = grad;
  g.fillRect(0, 0, 128, 128);
  return new THREE.CanvasTexture(c);
}

// ── Толпа на трибунах: два слоя точек (тёмная масса + вспышки телефонов) ──

function crowdPositions(count: number, seed: number): Float32Array {
  const arr = new Float32Array(count * 3);
  let s = seed;
  const rnd = () => { s = (s * 16807) % 2147483647; return s / 2147483647; };
  for (let i = 0; i < count; i++) {
    const angle = rnd() * Math.PI * 2;
    const tier = rnd();                    // 0..1 вверх по чаше
    const radius = 52 + tier * 26;
    arr[i * 3] = Math.cos(angle) * radius;
    arr[i * 3 + 1] = 4 + tier * 17 + rnd() * 0.8;
    arr[i * 3 + 2] = Math.sin(angle) * radius;
  }
  return arr;
}

function Crowd() {
  const flashes = useRef<THREE.PointsMaterial>(null);
  const base = useMemo(() => crowdPositions(3500, 42), []);
  const flash = useMemo(() => crowdPositions(320, 1337), []);
  useFrame(({ clock }) => {
    if (flashes.current) flashes.current.opacity = 0.45 + 0.45 * Math.abs(Math.sin(clock.elapsedTime * 2.3));
  });
  return (
    <>
      <points>
        <bufferGeometry>
          <bufferAttribute attach="attributes-position" args={[base, 3]} />
        </bufferGeometry>
        <pointsMaterial color="#3a4152" size={0.5} sizeAttenuation transparent opacity={0.85} />
      </points>
      <points>
        <bufferGeometry>
          <bufferAttribute attach="attributes-position" args={[flash, 3]} />
        </bufferGeometry>
        <pointsMaterial ref={flashes} color="#ffffff" size={0.65} sizeAttenuation transparent />
      </points>
    </>
  );
}

// ── Ворота с «живой» сеткой (рябь после гола) ─────────────────────────

const GOAL_X = 36;
const GOAL_W = 7.32, GOAL_H = 2.44;

function Goal({ progress }: { progress: MotionValue<number> }) {
  const netRef = useRef<THREE.Mesh>(null);
  const baseZ = useMemo(() => {
    // Запоминаем исходные вершины сетки, чтобы рябь была смещением от базы.
    const geo = new THREE.PlaneGeometry(GOAL_W, GOAL_H, 22, 10);
    return geo;
  }, []);

  useFrame(({ clock }) => {
    const p = progress.get();
    const net = netRef.current;
    if (!net) return;
    const geo = net.geometry as THREE.PlaneGeometry;
    const pos = geo.attributes.position;
    // Рябь по сетке в течение ~0.06 прогресса после гола.
    const k = p >= T.goal && p < T.goal + 0.08 ? 1 - (p - T.goal) / 0.08 : 0;
    for (let i = 0; i < pos.count; i++) {
      const x = pos.getX(i), y = pos.getY(i);
      const wave = k * 0.35 * Math.sin(x * 2.4 + y * 3 + clock.elapsedTime * 22);
      pos.setZ(i, wave);
    }
    pos.needsUpdate = true;
  });

  const post = new THREE.MeshStandardMaterial({ color: "#f5f5f7", roughness: 0.35 });
  return (
    <group position={[GOAL_X, 0, 0]}>
      {/* Штанги и перекладина */}
      <mesh material={post} position={[0, GOAL_H / 2, -GOAL_W / 2]}>
        <cylinderGeometry args={[0.09, 0.09, GOAL_H, 12]} />
      </mesh>
      <mesh material={post} position={[0, GOAL_H / 2, GOAL_W / 2]}>
        <cylinderGeometry args={[0.09, 0.09, GOAL_H, 12]} />
      </mesh>
      <mesh material={post} position={[0, GOAL_H, 0]} rotation={[Math.PI / 2, 0, 0]}>
        <cylinderGeometry args={[0.09, 0.09, GOAL_W + 0.18, 12]} />
      </mesh>
      {/* Сетка: задняя плоскость (с рябью) + верх */}
      <mesh ref={netRef} geometry={baseZ} position={[1.3, GOAL_H / 2, 0]} rotation={[0, -Math.PI / 2, 0]}>
        <meshBasicMaterial color="#cfd4dd" wireframe transparent opacity={0.4} />
      </mesh>
      <mesh position={[0.65, GOAL_H - 0.02, 0]} rotation={[0, -Math.PI / 2, 0]}>
        <planeGeometry args={[GOAL_W, 1.35, 22, 5]} />
        <meshBasicMaterial color="#cfd4dd" wireframe transparent opacity={0.32} side={THREE.DoubleSide} />
      </mesh>
    </group>
  );
}

// ── Мяч: качение → удар → полёт в девятку ─────────────────────────────

function useBallPosition() {
  // Дорожка качения (лёгкая «змейка» дриблинга) и дуга полёта.
  const roll = useMemo(() => new THREE.CatmullRomCurve3([
    new THREE.Vector3(-34, 0.42, 7),
    new THREE.Vector3(-22, 0.42, 3.5),
    new THREE.Vector3(-12, 0.42, 5),
    new THREE.Vector3(-4, 0.42, 1.5),
    new THREE.Vector3(0, 0.42, 0),
  ]), []);
  const flight = useMemo(() => new THREE.QuadraticBezierCurve3(
    new THREE.Vector3(0, 0.42, 0),
    new THREE.Vector3(19, 7.5, -1.6),
    new THREE.Vector3(GOAL_X + 0.9, GOAL_H - 0.35, -GOAL_W / 2 + 0.75), // девятка
  ), []);
  return (p: number, out: THREE.Vector3) => {
    if (p <= T.runStart) { out.copy(roll.getPoint(0)); return; }
    if (p < T.strike) { roll.getPoint((p - T.runStart) / (T.strike - T.runStart), out); return; }
    if (p < T.goal) { flight.getPoint((p - T.strike) / (T.goal - T.strike), out); return; }
    out.copy(flight.getPoint(1));
    out.y = Math.max(0.42, out.y - (p - T.goal) * 6); // мяч опускается в сетке
  };
}

function BallAndCamera({ progress }: { progress: MotionValue<number> }) {
  const ball = useRef<THREE.Mesh>(null);
  const shadow = useRef<THREE.Mesh>(null);
  const getBallPos = useBallPosition();
  const ballTex = useMemo(makeBallTexture, []);
  const v = useMemo(() => new THREE.Vector3(), []);
  const camTarget = useMemo(() => new THREE.Vector3(), []);
  const lookTarget = useMemo(() => new THREE.Vector3(-6, 1.2, 0), []);

  useFrame(({ camera }) => {
    const p = progress.get();
    getBallPos(p, v);
    if (ball.current) {
      ball.current.position.copy(v);
      ball.current.rotation.z = -p * 60; // вращение по ходу
      ball.current.rotation.x = p * 14;
    }
    if (shadow.current) {
      shadow.current.position.set(v.x, 0.02, v.z);
      const h = Math.min(v.y / 8, 1);
      shadow.current.scale.setScalar(1 - h * 0.55);
      (shadow.current.material as THREE.MeshBasicMaterial).opacity = 0.45 * (1 - h * 0.8);
    }

    // ── Камера: этапы полёта ──
    if (p < T.runStart) {
      camTarget.set(-30, 2.6, 16);
      lookTarget.lerp(new THREE.Vector3(-14, 1.4, 0), 0.08);
    } else if (p < T.strike) {
      // Низкий проезд за мячом над газоном
      camTarget.set(v.x - 7, 1.7 + p * 2, v.z + 8.5);
      lookTarget.lerp(v, 0.16);
    } else if (p < T.goal) {
      // Отлетаем вбок и провожаем мяч в полёте
      camTarget.set(10, 3.2, 13);
      lookTarget.lerp(v, 0.2);
    } else if (p < T.celebrate) {
      // Камера у ворот: мяч в сетке
      camTarget.set(24, 2.2, 9);
      lookTarget.lerp(new THREE.Vector3(GOAL_X, 1.6, -1.5), 0.1);
    } else {
      // Финал: отъезд на общий план под CTA
      camTarget.set(4, 7, 24);
      lookTarget.lerp(new THREE.Vector3(8, 1.5, 0), 0.06);
    }
    camera.position.lerp(camTarget, 0.09);
    camera.lookAt(lookTarget);
  });

  return (
    <>
      <mesh ref={ball}>
        <sphereGeometry args={[0.42, 40, 28]} />
        <meshStandardMaterial map={ballTex} roughness={0.38} metalness={0.05} />
      </mesh>
      <mesh ref={shadow} rotation={[-Math.PI / 2, 0, 0]}>
        <circleGeometry args={[0.5, 24]} />
        <meshBasicMaterial color="#000000" transparent opacity={0.45} depthWrite={false} />
      </mesh>
    </>
  );
}

// ── Прожекторные мачты ────────────────────────────────────────────────

function Floodlight({ x, z, glow }: { x: number; z: number; glow: THREE.Texture }) {
  return (
    <group position={[x, 0, z]}>
      <mesh position={[0, 11, 0]}>
        <cylinderGeometry args={[0.25, 0.4, 22, 8]} />
        <meshStandardMaterial color="#1a1f2b" roughness={0.8} />
      </mesh>
      <mesh position={[0, 22.5, 0]}>
        <boxGeometry args={[3.4, 1.6, 0.6]} />
        <meshStandardMaterial color="#0e1117" emissive="#f8f6e8" emissiveIntensity={1.6} />
      </mesh>
      <sprite position={[0, 22.5, 0]} scale={[10, 5, 1]}>
        <spriteMaterial map={glow} transparent opacity={0.8} blending={THREE.AdditiveBlending} depthWrite={false} />
      </sprite>
    </group>
  );
}

// ── Мир ───────────────────────────────────────────────────────────────

function World({ progress }: { progress: MotionValue<number> }) {
  const pitchTex = useMemo(makePitchTexture, []);
  const glowTex = useMemo(makeGlowTexture, []);
  const goalLight = useRef<THREE.PointLight>(null);

  useFrame(() => {
    const p = progress.get();
    // Вспышка света у ворот в момент гола.
    if (goalLight.current) {
      const k = p >= T.goal - 0.01 && p < T.goal + 0.06 ? 1 - Math.abs(p - T.goal - 0.02) / 0.05 : 0;
      goalLight.current.intensity = Math.max(0, k) * 60;
    }
  });

  return (
    <>
      <fog attach="fog" args={["#05070b", 65, 210]} />
      <ambientLight intensity={0.85} />
      <hemisphereLight args={["#cdd9ff", "#0a2013", 0.7]} />
      <directionalLight position={[20, 30, 15]} intensity={1.5} color="#eef2ff" />
      <directionalLight position={[-25, 25, -10]} intensity={0.7} color="#dfe8ff" />
      <pointLight ref={goalLight} position={[GOAL_X, 3, 0]} color="#ffffff" intensity={0} distance={30} />

      {/* Поле */}
      <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, 0, 0]}>
        <planeGeometry args={[110, 74]} />
        <meshStandardMaterial map={pitchTex} roughness={0.95} />
      </mesh>

      {/* Чаша трибун вокруг поля */}
      <mesh position={[0, 11, 0]}>
        <cylinderGeometry args={[85, 48, 26, 48, 1, true]} />
        <meshStandardMaterial color="#171d2b" side={THREE.BackSide} roughness={0.9} />
      </mesh>
      <Crowd />

      {/* Прожекторы по углам */}
      <Floodlight x={-36} z={-26} glow={glowTex} />
      <Floodlight x={-36} z={26} glow={glowTex} />
      <Floodlight x={36} z={-26} glow={glowTex} />
      <Floodlight x={36} z={26} glow={glowTex} />

      <Goal progress={progress} />
      <BallAndCamera progress={progress} />
    </>
  );
}

export default function Scene3D({ progress }: { progress: MotionValue<number> }) {
  return (
    <Canvas
      dpr={[1, 1.75]}
      gl={{ antialias: true, powerPreference: "high-performance" }}
      camera={{ position: [-30, 2.6, 16], fov: 55, near: 0.1, far: 260 }}
      style={{ position: "absolute", inset: 0 }}
    >
      <color attach="background" args={["#05070b"]} />
      <World progress={progress} />
    </Canvas>
  );
}
