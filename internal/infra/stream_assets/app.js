(() => {
  const statusEl = document.getElementById("status");
  const volumeEl = document.getElementById("volume");
  const volumeIcon = document.getElementById("volume-icon");
  const timelineEl = document.getElementById("timeline");
  const player = document.getElementById("player");

  const parsedSize = parseInt(document.body.dataset.historySize, 10);
  const historySize = Number.isFinite(parsedSize) && parsedSize > 0 ? parsedSize : 10;
  // スクロール追従判定のしきい値（px）。最下部からこの範囲内なら新着時に自動スクロールする。
  const autoScrollThresholdPx = 32;

  const queue = [];
  let playingItem = null;

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

  const isNearBottom = () => {
    const { scrollTop, scrollHeight, clientHeight } = timelineEl;
    return scrollHeight - scrollTop - clientHeight <= autoScrollThresholdPx;
  };

  const appendTimelineItem = (clip) => {
    // 新着追加前の位置で「既に最下部近辺だったか」を判定し、追従するかを決める。
    const shouldAutoScroll = isNearBottom();

    const li = document.createElement("li");
    li.className = "timeline-item";
    li.dataset.clipId = String(clip.id);
    const body = document.createElement("span");
    body.className = "timeline-text";
    body.textContent = clip.text || "";
    li.appendChild(body);
    timelineEl.appendChild(li);

    // 上限を超えた分を先頭から削除（再生中は残す）。
    while (timelineEl.children.length > historySize) {
      const first = timelineEl.firstElementChild;
      if (!first || first === playingItem) break;
      timelineEl.removeChild(first);
    }

    if (shouldAutoScroll) {
      timelineEl.scrollTop = timelineEl.scrollHeight;
    }
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
