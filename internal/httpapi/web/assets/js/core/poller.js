export function createPoller(task, intervalMs = 5000) {
  let timer = null;
  let busy = false;

  async function tick() {
    if (busy) return;
    busy = true;
    try { await task(); } finally { busy = false; }
  }

  return {
    start() { if (!timer) timer = setInterval(tick, intervalMs); },
    stop() { if (timer) { clearInterval(timer); timer = null; } },
    get running() { return timer !== null; },
  };
}
