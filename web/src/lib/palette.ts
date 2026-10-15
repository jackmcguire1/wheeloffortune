export const PALETTE = [
  "#ff4f8b",
  "#ffb627",
  "#2ec4b6",
  "#7c4dff",
  "#3a86ff",
  "#ff6b35",
  "#06d6a0",
  "#e63946",
  "#8338ec",
  "#fb5607",
  "#118ab2",
  "#ffd166",
];

export function colorFor(index: number, color?: string): string {
  return color || PALETTE[index % PALETTE.length];
}

/** Picks black or white text for legibility on the given hex colour. */
export function contrastText(hex: string): string {
  const m = /^#?([\da-f]{2})([\da-f]{2})([\da-f]{2})/i.exec(hex);
  if (!m) return "#fff";
  const [r, g, b] = m.slice(1).map((h) => parseInt(h, 16) / 255);
  const lum = 0.2126 * r + 0.7152 * g + 0.0722 * b;
  return lum > 0.6 ? "#1a1033" : "#ffffff";
}
