import "./app.css";

(() => {
  const statusBadge = document.getElementById("status-badge");
  const volumeEl = document.getElementById("volume");
  const volumeIcon = document.getElementById("volume-icon");
  const muteEl = document.getElementById("mute");
  const timelineEl = document.getElementById("timeline");
  const historySizeEl = document.getElementById("history-size");
  const showSpeakerNameEl = document.getElementById("show-speaker-name");
  const showStyleNameEl = document.getElementById("show-style-name");
  const showTimestampEl = document.getElementById("show-timestamp");
  const player = document.getElementById("player");
  const tabStreamEl = document.getElementById("tab-stream");
  const tabTestEl = document.getElementById("tab-test");
  const panelStreamEl = document.getElementById("panel-stream");
  const panelTestEl = document.getElementById("panel-test");
  const testSpeakerEl = document.getElementById("test-speaker");
  const testPlayEl = document.getElementById("test-play");
  const testErrorEl = document.getElementById("test-error");

  const historySizeStorageKey = "vox-actor.stream.historySize";
  const defaultHistorySize = 20;
  const volumeStorageKey = "vox-actor.stream.volume";
  const defaultVolume = 50;
  const activeTabStorageKey = "vox-actor.stream.activeTab";
  const testSpeakerStorageKey = "vox-actor.stream.testSpeakerId";
  const defaultActiveTab = "stream";
  const testErrorDisplayMs = 4000;

  const toggles = [
    { el: showSpeakerNameEl, storageKey: "vox-actor.stream.showSpeakerName", bodyClass: "hide-speaker-name" },
    { el: showStyleNameEl, storageKey: "vox-actor.stream.showStyleName", bodyClass: "hide-style-name" },
    { el: showTimestampEl, storageKey: "vox-actor.stream.showTimestamp", bodyClass: "hide-timestamp" },
  ];

  const queue = [];
  let playingItem = null;
  let historySize = initHistorySize();
  let activeTab = defaultActiveTab;
  let testErrorTimer = null;

  function initHistorySize() {
    const stored = parseInt(localStorage.getItem(historySizeStorageKey), 10);
    const allowed = Array.from(historySizeEl.options).map((o) => parseInt(o.value, 10));
    const value = allowed.includes(stored) ? stored : defaultHistorySize;
    historySizeEl.value = String(value);
    return value;
  }

  function initVolume() {
    const stored = parseInt(localStorage.getItem(volumeStorageKey), 10);
    const value = Number.isInteger(stored) && stored >= 0 && stored <= 100 ? stored : defaultVolume;
    volumeEl.value = String(value);
    player.volume = value / 100;
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

  const formatTimestamp = (ms) => {
    if (typeof ms !== "number" || !Number.isFinite(ms) || ms <= 0) return "";
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

  const stopPlayback = () => {
    player.pause();
    player.removeAttribute("src");
    player.load();
    queue.length = 0;
    clearPlayingHighlight();
  };

  const playNext = () => {
    if (activeTab !== "stream") return;
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

  const applyTab = (tab) => {
    activeTab = tab === "test" ? "test" : "stream";
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

  const setActiveTab = (tab) => {
    applyTab(tab);
    localStorage.setItem(activeTabStorageKey, activeTab);
  };

  tabStreamEl.addEventListener("click", () => setActiveTab("stream"));
  tabTestEl.addEventListener("click", () => setActiveTab("test"));

  const showTestError = (msg) => {
    testErrorEl.textContent = msg;
    if (testErrorTimer) {
      clearTimeout(testErrorTimer);
    }
    testErrorTimer = setTimeout(() => {
      testErrorEl.textContent = "";
      testErrorTimer = null;
    }, testErrorDisplayMs);
  };

  const loadSpeakers = async () => {
    try {
      const resp = await fetch("/speakers.json");
      if (!resp.ok) throw new Error(`status ${resp.status}`);
      const speakers = await resp.json();
      testSpeakerEl.innerHTML = "";
      speakers.forEach((s) => {
        const opt = document.createElement("option");
        opt.value = String(s.id);
        const style = s.styleName ? `(${s.styleName})` : "";
        opt.textContent = `${s.speakerName}${style}`;
        testSpeakerEl.appendChild(opt);
      });
      const stored = localStorage.getItem(testSpeakerStorageKey);
      if (stored && Array.from(testSpeakerEl.options).some((o) => o.value === stored)) {
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
    player.play().catch((err) => {
      console.error("test play failed", err);
      showTestError("合成に失敗しました");
    });
  });

  const setBadge = (connected) => {
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

  const connect = () => {
    const es = new EventSource("/events");
    es.addEventListener("open", () => {
      setBadge(true);
    });
    es.addEventListener("clip", (event) => {
      try {
        const clip = JSON.parse(event.data);
        const item = appendTimelineItem(clip);
        if (activeTab === "stream") {
          queue.push({ clip, item });
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
  loadSpeakers();
  connect();
})();
