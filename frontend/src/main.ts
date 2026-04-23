import "./app.css";
import { getElement } from "./dom";
import {
  isClipEvent,
  isSpeakerArray,
  type ClipEvent,
  type Speaker,
} from "./types/api";

interface ToggleConfig {
  el: HTMLInputElement;
  storageKey: string;
  bodyClass: string;
}

interface QueueItem {
  clip: ClipEvent;
  item: HTMLLIElement;
}

type TabName = "stream" | "test";

(() => {
  const statusBadge = getElement<HTMLSpanElement>("status-badge");
  const volumeEl = getElement<HTMLInputElement>("volume");
  const volumeIcon = getElement<HTMLSpanElement>("volume-icon");
  const muteEl = getElement<HTMLInputElement>("mute");
  const timelineEl = getElement<HTMLOListElement>("timeline");
  const historySizeEl = getElement<HTMLSelectElement>("history-size");
  const showSpeakerNameEl = getElement<HTMLInputElement>("show-speaker-name");
  const showStyleNameEl = getElement<HTMLInputElement>("show-style-name");
  const showTimestampEl = getElement<HTMLInputElement>("show-timestamp");
  const player = getElement<HTMLAudioElement>("player");
  const tabStreamEl = getElement<HTMLButtonElement>("tab-stream");
  const tabTestEl = getElement<HTMLButtonElement>("tab-test");
  const panelStreamEl = getElement<HTMLElement>("panel-stream");
  const panelTestEl = getElement<HTMLElement>("panel-test");
  const testSpeakerEl = getElement<HTMLSelectElement>("test-speaker");
  const testPlayEl = getElement<HTMLButtonElement>("test-play");
  const testErrorEl = getElement<HTMLSpanElement>("test-error");

  const historySizeStorageKey = "vox-actor.stream.historySize";
  const defaultHistorySize = 20;
  const volumeStorageKey = "vox-actor.stream.volume";
  const defaultVolume = 50;
  const activeTabStorageKey = "vox-actor.stream.activeTab";
  const testSpeakerStorageKey = "vox-actor.stream.testSpeakerId";
  const defaultActiveTab: TabName = "stream";
  const testErrorDisplayMs = 4000;

  const toggles: ToggleConfig[] = [
    { el: showSpeakerNameEl, storageKey: "vox-actor.stream.showSpeakerName", bodyClass: "hide-speaker-name" },
    { el: showStyleNameEl, storageKey: "vox-actor.stream.showStyleName", bodyClass: "hide-style-name" },
    { el: showTimestampEl, storageKey: "vox-actor.stream.showTimestamp", bodyClass: "hide-timestamp" },
  ];

  const queue: QueueItem[] = [];
  let playingItem: HTMLLIElement | null = null;
  let historySize = initHistorySize();
  let activeTab: TabName = defaultActiveTab;
  let testErrorTimer: ReturnType<typeof setTimeout> | null = null;

  function initHistorySize(): number {
    const stored = parseInt(localStorage.getItem(historySizeStorageKey) ?? "", 10);
    const allowed = Array.from(historySizeEl.options).map((o) => parseInt(o.value, 10));
    const value = allowed.includes(stored) ? stored : defaultHistorySize;
    historySizeEl.value = String(value);
    return value;
  }

  function initVolume(): void {
    const stored = parseInt(localStorage.getItem(volumeStorageKey) ?? "", 10);
    const value = Number.isInteger(stored) && stored >= 0 && stored <= 100 ? stored : defaultVolume;
    volumeEl.value = String(value);
    player.volume = value / 100;
  }

  function initToggles(): void {
    toggles.forEach((t) => {
      const stored = localStorage.getItem(t.storageKey);
      const checked = stored === null ? true : stored === "true";
      t.el.checked = checked;
      applyToggleState(t, checked);
      t.el.addEventListener("change", () => {
        applyToggleState(t, t.el.checked);
        localStorage.setItem(t.storageKey, String(t.el.checked));
      });
    });
  }

  function applyToggleState(t: ToggleConfig, checked: boolean): void {
    if (checked) {
      document.body.classList.remove(t.bodyClass);
    } else {
      document.body.classList.add(t.bodyClass);
    }
  }

  const setVolumeIcon = (): void => {
    volumeIcon.textContent = player.muted || player.volume === 0 ? "🔇" : "🔊";
  };

  volumeEl.addEventListener("input", () => {
    const value = Number(volumeEl.value);
    player.volume = value / 100;
    localStorage.setItem(volumeStorageKey, String(value));
    setVolumeIcon();
  });

  muteEl.addEventListener("change", () => {
    player.muted = muteEl.checked;
    setVolumeIcon();
  });

  const trimTimeline = (): void => {
    while (timelineEl.children.length > historySize) {
      const first = timelineEl.firstElementChild;
      if (!first || first === playingItem) break;
      timelineEl.removeChild(first);
    }
  };

  historySizeEl.addEventListener("change", () => {
    const next = parseInt(historySizeEl.value, 10);
    if (!Number.isFinite(next) || next <= 0) return;
    historySize = next;
    localStorage.setItem(historySizeStorageKey, String(next));
    trimTimeline();
  });

  const formatTimestamp = (ms: number): string => {
    if (!Number.isFinite(ms) || ms <= 0) return "";
    return new Date(ms).toLocaleTimeString("ja-JP", { hour12: false });
  };

  const appendTimelineItem = (clip: ClipEvent): HTMLLIElement => {
    const li = document.createElement("li");
    li.className = "timeline-item";
    li.dataset.clipId = String(clip.id);

    const meta = document.createElement("div");
    meta.className = "timeline-meta";

    const timestamp = document.createElement("span");
    timestamp.className = "timeline-timestamp";
    timestamp.textContent = formatTimestamp(clip.timestamp);
    meta.appendChild(timestamp);

    const speakerName = document.createElement("span");
    speakerName.className = "timeline-speaker-name";
    speakerName.textContent = clip.speakerName;
    meta.appendChild(speakerName);

    const styleName = document.createElement("span");
    styleName.className = "timeline-style-name";
    styleName.textContent = clip.styleName ? `[${clip.styleName}]` : "";
    meta.appendChild(styleName);

    li.appendChild(meta);

    const body = document.createElement("div");
    body.className = "timeline-body";
    body.textContent = clip.text;
    li.appendChild(body);

    timelineEl.appendChild(li);

    trimTimeline();
    return li;
  };

  const clearPlayingHighlight = (): void => {
    if (playingItem) {
      playingItem.classList.remove("playing");
      playingItem = null;
    }
  };

  const stopPlayback = (): void => {
    player.pause();
    player.removeAttribute("src");
    player.load();
    queue.length = 0;
    clearPlayingHighlight();
  };

  const playNext = (): void => {
    if (activeTab !== "stream") return;
    if (playingItem) return;
    const next = queue.shift();
    if (!next) return;
    const { clip, item } = next;
    playingItem = item;
    item.classList.add("playing");
    item.scrollIntoView({ block: "nearest" });
    player.src = clip.url;
    player.play().catch((err: unknown) => {
      console.error("play failed", err);
      clearPlayingHighlight();
      playNext();
    });
  };

  player.addEventListener("ended", () => {
    clearPlayingHighlight();
    playNext();
  });

  const applyTab = (tab: TabName): void => {
    activeTab = tab;
    const isStream = activeTab === "stream";
    tabStreamEl.classList.toggle("active", isStream);
    tabStreamEl.setAttribute("aria-selected", String(isStream));
    tabTestEl.classList.toggle("active", !isStream);
    tabTestEl.setAttribute("aria-selected", String(!isStream));
    panelStreamEl.classList.toggle("hidden", !isStream);
    panelTestEl.classList.toggle("hidden", isStream);
    stopPlayback();
    if (isStream) {
      playNext();
    }
  };

  const setActiveTab = (tab: TabName): void => {
    applyTab(tab);
    localStorage.setItem(activeTabStorageKey, activeTab);
  };

  tabStreamEl.addEventListener("click", () => setActiveTab("stream"));
  tabTestEl.addEventListener("click", () => setActiveTab("test"));

  const showTestError = (msg: string): void => {
    testErrorEl.textContent = msg;
    if (testErrorTimer !== null) {
      clearTimeout(testErrorTimer);
    }
    testErrorTimer = setTimeout(() => {
      testErrorEl.textContent = "";
      testErrorTimer = null;
    }, testErrorDisplayMs);
  };

  const loadSpeakers = async (): Promise<void> => {
    try {
      const resp = await fetch("/speakers.json");
      if (!resp.ok) throw new Error(`status ${resp.status}`);
      const data: unknown = await resp.json();
      if (!isSpeakerArray(data)) {
        throw new Error("invalid speakers payload");
      }
      const speakers: Speaker[] = data;
      testSpeakerEl.innerHTML = "";
      speakers.forEach((s) => {
        const opt = document.createElement("option");
        opt.value = String(s.id);
        const style = s.styleName ? `(${s.styleName})` : "";
        opt.textContent = `${s.speakerName}${style}`;
        testSpeakerEl.appendChild(opt);
      });
      const stored = localStorage.getItem(testSpeakerStorageKey);
      if (stored !== null && Array.from(testSpeakerEl.options).some((o) => o.value === stored)) {
        testSpeakerEl.value = stored;
      }
    } catch (err) {
      console.error("failed to load speakers", err);
      showTestError("話者一覧の取得に失敗しました");
    }
  };

  testSpeakerEl.addEventListener("change", () => {
    localStorage.setItem(testSpeakerStorageKey, testSpeakerEl.value);
  });

  testPlayEl.addEventListener("click", () => {
    const speakerId = testSpeakerEl.value;
    if (!speakerId) {
      showTestError("話者が選択されていません");
      return;
    }
    testErrorEl.textContent = "";
    player.pause();
    player.src = `/test-clip?speaker=${encodeURIComponent(speakerId)}`;
    player.play().catch((err: unknown) => {
      console.error("test play failed", err);
      showTestError("合成に失敗しました");
    });
  });

  const setBadge = (connected: boolean): void => {
    if (connected) {
      statusBadge.textContent = "● 接続中";
      statusBadge.classList.add("badge-connected");
      statusBadge.classList.remove("badge-disconnected");
    } else {
      statusBadge.textContent = "● 切断";
      statusBadge.classList.add("badge-disconnected");
      statusBadge.classList.remove("badge-connected");
    }
  };

  const connect = (): void => {
    const es = new EventSource("/events");
    es.addEventListener("open", () => {
      setBadge(true);
    });
    es.addEventListener("clip", (event) => {
      try {
        const data: unknown = JSON.parse(event.data);
        if (!isClipEvent(data)) {
          console.error("invalid clip payload", data);
          return;
        }
        const item = appendTimelineItem(data);
        if (activeTab === "stream") {
          queue.push({ clip: data, item });
          playNext();
        }
      } catch (err) {
        console.error("invalid clip payload", err);
      }
    });
    es.addEventListener("error", () => {
      setBadge(false);
      es.close();
      setTimeout(connect, 2000);
    });
  };

  initToggles();
  initVolume();
  player.muted = muteEl.checked;
  setVolumeIcon();
  const storedTab = localStorage.getItem(activeTabStorageKey);
  applyTab(storedTab === "test" ? "test" : "stream");
  void loadSpeakers();
  connect();
})();
