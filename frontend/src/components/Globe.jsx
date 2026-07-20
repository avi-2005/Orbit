import { useMemo, useRef } from "react";
import { useFrame, useLoader, useThree } from "@react-three/fiber";
import * as THREE from "three";
import { latLonToVector3 } from "../utils/geo";
import { getSubsolarPoint } from "../utils/sun";

const EARTH_TEXTURE =
  "https://unpkg.com/three-globe@2.31.0/example/img/earth-blue-marble.jpg";

export const GLOBE_RADIUS = 2;

// Custom shader instead of a plain meshStandardMaterial so we can shade
// the globe based on real-time sun position: everywhere the surface
// normal points away from the sun gets dimmed toward "night", with a
// soft blended band at the terminator (the day/night boundary).
const VERTEX_SHADER = `
  varying vec2 vUv;
  varying vec3 vNormal;
  void main() {
    vUv = uv;
    // For a unit sphere centered at the origin, the un-transformed
    // vertex position IS the surface normal — cheaper and simpler than
    // computing it via the normal matrix, and exactly what we want since
    // this mesh never rotates (see note in App.jsx about why).
    vNormal = normalize(position);
    gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
  }
`;

const FRAGMENT_SHADER = `
  uniform sampler2D dayMap;
  uniform vec3 sunDirection;
  varying vec2 vUv;
  varying vec3 vNormal;

  void main() {
    vec3 dayColor = texture2D(dayMap, vUv).rgb;
    float sunFacing = dot(normalize(vNormal), normalize(sunDirection));
    float mixFactor = smoothstep(-0.15, 0.15, sunFacing);
    vec3 nightColor = dayColor * 0.12;
    vec3 color = mix(nightColor, dayColor, mixFactor);
    gl_FragColor = vec4(color, 1.0);
  }
`;

function initialSunDirection() {
  const { lat, lon } = getSubsolarPoint();
  return latLonToVector3(lat, lon, 1).normalize();
}

export default function Globe() {
  const { gl } = useThree();
  const colorMap = useLoader(THREE.TextureLoader, EARTH_TEXTURE);

  useMemo(() => {
    // Blurriness when zooming into a textured sphere is almost always a
    // missing-anisotropic-filtering problem, not a resolution problem —
    // this asks the GPU for its max supported quality instead of the
    // low default, which is the actual fix for "map goes blurry up close."
    colorMap.anisotropy = gl.capabilities.getMaxAnisotropy();
    colorMap.minFilter = THREE.LinearMipmapLinearFilter;
    colorMap.needsUpdate = true;
  }, [colorMap, gl]);

  const uniforms = useMemo(
    () => ({
      dayMap: { value: colorMap },
      sunDirection: { value: initialSunDirection() },
    }),
    [colorMap]
  );

  const sunUpdateAccum = useRef(0);

  useFrame((_, delta) => {
    sunUpdateAccum.current += delta;
    // The sun's position barely changes second to second — recomputing
    // every 30s (instead of every frame) is visually seamless and far
    // cheaper than doing this trig 60 times a second.
    if (sunUpdateAccum.current < 30) return;
    sunUpdateAccum.current = 0;

    const { lat, lon } = getSubsolarPoint();
    uniforms.sunDirection.value.copy(latLonToVector3(lat, lon, 1).normalize());
  });

  return (
    <group>
      <mesh>
        <sphereGeometry args={[GLOBE_RADIUS, 96, 96]} />
        <shaderMaterial
          vertexShader={VERTEX_SHADER}
          fragmentShader={FRAGMENT_SHADER}
          uniforms={uniforms}
        />
      </mesh>

      {/* Soft atmospheric glow shell */}
      <mesh>
        <sphereGeometry args={[GLOBE_RADIUS * 1.015, 64, 64]} />
        <meshBasicMaterial
          color="#4da6ff"
          transparent
          opacity={0.06}
          side={THREE.BackSide}
        />
      </mesh>
      <mesh>
        <sphereGeometry args={[GLOBE_RADIUS * 1.08, 64, 64]} />
        <meshBasicMaterial
          color="#4da6ff"
          transparent
          opacity={0.04}
          side={THREE.BackSide}
        />
      </mesh>
    </group>
  );
}
