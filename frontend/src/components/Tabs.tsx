export type TabName = "stream" | "test";

interface TabsProps {
  active: TabName;
  onChange: (tab: TabName) => void;
}

interface TabButtonProps {
  id: string;
  panelId: string;
  active: boolean;
  label: string;
  onClick: () => void;
}

function TabButton({ id, panelId, active, label, onClick }: TabButtonProps) {
  const base = "cursor-pointer rounded-t-md px-2 py-[0.4rem] font-inherit sm:px-[0.9rem] sm:py-[0.45rem]";
  const colors = active
    ? "border border-ctp-overlay border-b-ctp-base bg-ctp-base text-ctp-text"
    : "border-0 bg-transparent text-ctp-subtext hover:text-ctp-text";
  return (
    <button
      type="button"
      id={id}
      role="tab"
      aria-selected={active}
      aria-controls={panelId}
      onClick={onClick}
      className={`${base} ${colors}`}
    >
      {label}
    </button>
  );
}

export function Tabs({ active, onChange }: TabsProps) {
  return (
    <div role="tablist" className="mt-4 flex gap-[0.15rem] border-b border-ctp-overlay sm:gap-1">
      <TabButton
        id="tab-stream"
        panelId="panel-stream"
        active={active === "stream"}
        label="配信"
        onClick={() => onChange("stream")}
      />
      <TabButton
        id="tab-test"
        panelId="panel-test"
        active={active === "test"}
        label="音声テスト"
        onClick={() => onChange("test")}
      />
    </div>
  );
}
