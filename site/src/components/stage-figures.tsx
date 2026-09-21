import type { ReactNode } from "react";

// Line-art figures for the landing page "System plan" cards.
// Monochrome (stroke currentColor, hairline weight) so they inherit the
// archive theme; the parent card supplies `group` + text color for the
// subtle hover shift. Decorative reinforcement of adjacent copy: aria-hidden.
function Frame({ children }: { children: ReactNode }) {
  return (
    <svg
      viewBox="0 0 120 72"
      className="h-full w-full"
      fill="none"
      stroke="currentColor"
      strokeWidth={1}
      aria-hidden="true"
      focusable="false"
    >
      {children}
    </svg>
  );
}

export function StageInput() {
  return (
    <Frame>
      <rect x={14} y={14} width={82} height={32} rx={2} />
      <line x1={23} y1={22} x2={23} y2={32} strokeWidth={2} />
      <line x1={31} y1={25} x2={79} y2={25} />
      <line x1={31} y1={32} x2={63} y2={32} />
      <line x1={96} y1={30} x2={108} y2={30} />
      <polyline points="104,26 108,30 104,34" />
    </Frame>
  );
}

export function StageRetrieve() {
  return (
    <Frame>
      <rect x={14} y={12} width={92} height={9} fill="currentColor" fillOpacity={0.14} />
      <rect x={14} y={29} width={70} height={9} />
      <rect x={14} y={46} width={82} height={9} />
      <line x1={106} y1={12} x2={106} y2={21} strokeWidth={2} />
    </Frame>
  );
}

export function StageInfer() {
  return (
    <Frame>
      <rect x={40} y={6} width={40} height={34} rx={2} />
      <line x1={48} y1={16} x2={72} y2={16} />
      <line x1={48} y1={23} x2={64} y2={23} />
      <line x1={60} y1={40} x2={60} y2={48} />
      {[28, 43, 58, 73, 88].map((x, i) => (
        <rect
          key={x}
          x={x}
          y={48}
          width={9}
          height={9}
          fill={i < 3 ? "currentColor" : "none"}
          fillOpacity={i < 3 ? 0.45 - i * 0.1 : undefined}
        />
      ))}
    </Frame>
  );
}

export function StageRender() {
  return (
    <Frame>
      <rect x={26} y={8} width={68} height={56} />
      <line x1={46} y1={26} x2={74} y2={26} />
      <line x1={40} y1={36} x2={80} y2={36} />
      <line x1={50} y1={46} x2={70} y2={46} />
      <path d="M26 8 h8 M26 8 v8 M94 64 h-8 M94 64 v-8" strokeWidth={2} />
    </Frame>
  );
}
