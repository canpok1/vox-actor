(() => {
  const statusEl = document.getElementById("status");
  const volumeEl = document.getElementById("volume");
  const volumeIcon = document.getElementById("volume-icon");
  const timelineEl = document.getElementById("timeline");
  const historySizeEl = document.getElementById("history-size");
  const showSpeakerNameEl = document.getElementById("show-speaker-name");
  const showStyleNameEl = document.getElementById("show-style-name");
  const showTimestampEl = document.getElementById("show-timestamp");
  const player = document.getElementById("player");

  const historySizeStorageKey = "vox-actor.stream.historySize";
  const defaultHistorySize = 20;

  const toggles = [
    { el: showSpeakerNameEl, storageKey: "vox-actor.stream.showSpeakerName", bodyClass: "hide-speaker-name" },
    { el: showStyleNameEl, storageKey: "vox-actor.stream.showStyleName", bodyClass: "hide-style-name" },
    { el: showTimestampEl, storageKey: "vox-actor.stream.showTimestamp", bodyClass: "hide-timestamp" },
  ];

  const queue = [];
  let playingItem = null;
  let historySize = initHistorySize();

  function initHistorySize() {
    const stored = parseInt(localStorage.getItem(historySizeStorageKey), 10);
    const allowed = Array.from(historySizeEl.options).map((o) => parseInt(o.value, 10));
    const value = allowed.includes(stored) ? stored : defaultHistorySize;
    historySizeEl.value = String(value);
    return value;
  }

  function initToggles() {
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

  function applyToggleState(t, checked) {
    if (checked) {
      document.body.classList.remove(t.bodyClass);
    } else {
      document.body.classList.add(t.bodyClass);
    }
  }

  const setVolumeIcon = () => {
    volumeIcon.textContent = player.muted || player.volume === 0 ? "🔇" : "🔊";
  };

  const unlockAudio = () => {
    player.muted = false;
    setVolumeIcon();
  };

  volumeEl.addEventListener("input", () => {
    player.volume = Number(volumeEl.value) / 100;
    if (player.volume > 0) {
      unlockAudio();
    }
    setVolumeIcon();
  });

  document.body.addEventListener("click", unlockAudio, { once: true });

  const trimTimeline = () => {
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

  // formatTimestamp は Unix ms を 24 時間表記 HH:MM:SS に整形する。
  // サーバーが timestamp を送らない古いペイロードでも空文字を返すようにする。
  const formatTimestamp = (ms) => {
    if (typeof ms !== "number" || !Number.isFinite(ms)) return "";
    return new Date(ms).toLocaleTimeString("ja-JP", { hour12: false });
  };

  const appendTimelineItem = (clip) => {
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
    speakerName.textContent = clip.speakerName || "";
    meta.appendChild(speakerName);

    const styleName = document.createElement("span");
    styleName.className = "timeline-style-name";
    styleName.textContent = clip.styleName ? `[${clip.styleName}]` : "";
    meta.appendChild(styleName);

    li.appendChild(meta);

    const body = document.createElement("div");
    body.className = "timeline-body";
    body.textContent = clip.text || "";
    li.appendChild(body);

    timelineEl.appendChild(li);

    trimTimeline();
    return li;
  };

  const clearPlayingHighlight = () => {
    if (playingItem) {
      playingItem.classList.remove("playing");
      playingItem = null;
    }
  };

  const playNext = () => {
    if (playingItem || queue.length === 0) return;
    const { clip, item } = queue.shift();
    playingItem = item;
    item.classList.add("playing");
    item.scrollIntoView({ block: "nearest" });
    player.src = clip.url;
    player.play().catch((err) => {
      console.error("play failed", err);
      clearPlayingHighlight();
      playNext();
    });
  };

  player.addEventListener("ended", () => {
    clearPlayingHighlight();
    playNext();
  });

  const connect = () => {
    const es = new EventSource("/events");
    es.addEventListener("open", () => {
      statusEl.textContent = "接続中";
    });
    es.addEventListener("clip", (event) => {
      try {
        const clip = JSON.parse(event.data);
        const item = appendTimelineItem(clip);
        queue.push({ clip, item });
        playNext();
      } catch (err) {
        console.error("invalid clip payload", err);
      }
    });
    es.addEventListener("error", () => {
      statusEl.textContent = "切断";
      es.close();
      setTimeout(connect, 2000);
    });
  };

  initToggles();
  setVolumeIcon();
  connect();
})();
