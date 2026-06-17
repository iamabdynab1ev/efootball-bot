import { ImageResponse } from "next/og";

export const size = { width: 512, height: 512 };
export const contentType = "image/png";

export default function Icon() {
  return new ImageResponse(
    (
      <div
        style={{
          width: 512,
          height: 512,
          background: "#09090b",
          borderRadius: 120,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          flexDirection: "column",
          gap: 0,
        }}
      >
        {/* Trophy icon */}
        <div style={{ display: "flex", flexDirection: "column", alignItems: "center" }}>
          <div
            style={{
              fontSize: 260,
              lineHeight: 1,
              filter: "drop-shadow(0 0 40px #eab308aa)",
            }}
          >
            🏆
          </div>
          {/* eFL text */}
          <div
            style={{
              fontSize: 90,
              fontWeight: 900,
              color: "#eab308",
              letterSpacing: -2,
              marginTop: -10,
              fontFamily: "sans-serif",
            }}
          >
            eFL
          </div>
        </div>
      </div>
    ),
    { width: 512, height: 512 }
  );
}
