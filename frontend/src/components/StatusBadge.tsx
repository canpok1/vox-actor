interface StatusBadgeProps {
  connected: boolean;
}

export function StatusBadge({ connected }: StatusBadgeProps) {
  const color = connected
    ? "bg-ctp-green text-ctp-base"
    : "bg-ctp-red text-ctp-base";
  const label = connected ? "● 接続中" : "● 切断";
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-[0.1rem] text-xs font-semibold ${color}`}
    >
      {label}
    </span>
  );
}
