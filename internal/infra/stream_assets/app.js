(() => {
  const statusEl = document.getElementById("status");
  const volumeEl = document.getElementById("volume");
  const volumeIcon = document.getElementById("volume-icon");
  const nowPlayingEl = document.getElementById("now-playing");
  const player = document.getElementById("player");

  const queue = [];
  let playing = false;

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

  const playNext = () => {
    if (playing || queue.length === 0) return;
    const clip = queue.shift();
    playing = true;
    nowPlayingEl.textContent = `クリップ #${clip.id}`;
    player.src = clip.url;
    player.play().catch((err) => {
      console.error("play failed", err);
      playing = false;
      playNext();
    });
  };

  player.addEventListener("ended", () => {
    playing = false;
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
        queue.push(clip);
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
