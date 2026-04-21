(() => {
  const statusEl = document.getElementById("status");
  const volumeEl = document.getElementById("volume");
  const volumeIcon = document.getElementById("volume-icon");
  const timelineEl = document.getElementById("timeline");
  const historySizeEl = document.getElementById("history-size");
  const player = document.getElementById("player");

  const historySizeStorageKey = "vox-actor.stream.historySize";
  const defaultHistorySize = 20;

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

  const appendTimelineItem = (clip) => {
    const li = document.createElement("li");
    li.className = "timeline-item";
    li.dataset.clipId = String(clip.id);
    const body = document.createElement("span");
    body.className = "timeline-text";
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

  setVolumeIcon();
  connect();
})();
