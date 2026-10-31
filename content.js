(function () {
  'use strict';

  const SCROLL_DELAY = 500;
  const CHECK_INTERVAL = 100;
  const FEED_FILTER_INTERVAL = 1000;
  const CONTEXT_CHECK_INTERVAL = 5000;
  const RELOAD_COOLDOWN_MS = 10000;

  let currentVideoId = null;
  let watchCount = 0;
  let isEnabled = true;
  let maxWatchCount = 1;
  let blockedKeywords = [];
  let blockedAudios = [];
  let videoData = {};
  let trackingSession = 0;

  async function init() {
    await loadSettings();
    await loadVideoData();
    startObserver();
    startContextWatchdog();
  }

  function isInvalidContextError(error) {
    return !!error && /Extension context invalidated/i.test(error.message || '');
  }

  function reloadPage() {
    const key = 'scLastAutoReload';
    const last = Number(sessionStorage.getItem(key) || 0);
    if (Date.now() - last < RELOAD_COOLDOWN_MS) return;
    sessionStorage.setItem(key, String(Date.now()));
    window.location.reload();
  }

  function startContextWatchdog() {
    setInterval(async () => {
      try {
        await chrome.storage.local.get('enabled');
      } catch (error) {
        if (isInvalidContextError(error)) {
          reloadPage();
        }
      }
    }, CONTEXT_CHECK_INTERVAL);
  }

  async function loadSettings() {
    try {
      const result = await chrome.storage.local.get([
        'enabled',
        'maxWatchCount',
        'blockedKeywords',
        'blockedAudios'
      ]);

      isEnabled = result.enabled !== false;
      maxWatchCount = result.maxWatchCount || 1;
      blockedKeywords = result.blockedKeywords || [];
      blockedAudios = result.blockedAudios || [];
    } catch (error) {
      if (isInvalidContextError(error)) reloadPage();
    }
  }

  async function loadVideoData() {
    try {
      const result = await chrome.storage.local.get('videoData');
      videoData = result.videoData || {};
    } catch (error) {
      if (isInvalidContextError(error)) reloadPage();
    }
  }

  async function saveVideoData() {
    try {
      await chrome.storage.local.set({ videoData });
    } catch (error) {
      if (isInvalidContextError(error)) reloadPage();
    }
  }

  function getVideoId() {
    const shortsMatch = window.location.pathname.match(/\/shorts\/([a-zA-Z0-9_-]+)/);
    if (shortsMatch) {
      return shortsMatch[1];
    }

    const urlParams = new URLSearchParams(window.location.search);
    return urlParams.get('v');
  }

  function getVideoElement() {
    return document.querySelector('video');
  }

  function getTitle() {
    const titleElement = document.querySelector('h1.ytd-watch-metadata yt-formatted-string, h1.ytd-reel-player-header-renderer, ytd-reel-player-header-renderer h2');
    return titleElement ? titleElement.textContent.trim() : '';
  }

  function extractAudioName() {
    const title = getTitle().trim().toLowerCase();
    const isValidCandidate = (text) => {
      const trimmed = (text || '').trim();
      return !!trimmed && trimmed.toLowerCase() !== title;
    };

    const description = document.querySelector('#description-text');
    if (description) {
      const match = (description.textContent || '').match(/(?:song|music|audio)[:\s]+([^\n]+)/i);
      if (match && isValidCandidate(match[1])) {
        return match[1].trim();
      }
    }

    const soundButtons = document.querySelectorAll('.ytSpecButtonShapeNextButtonTextContent');
    const withArtistSeparator = Array.from(soundButtons).find(el => el.textContent.includes('·'));
    if (withArtistSeparator && isValidCandidate(withArtistSeparator.textContent)) {
      return withArtistSeparator.textContent.trim();
    }

    const selectors = [
      'a[href*="/source/"]',
      '[aria-label*="original audio" i]',
      '[aria-label*="song" i]',
      '[aria-label*="music" i]',
      '.ytSpecButtonShapeNextButtonTextContent',
      'ytd-reel-player-header-renderer a'
    ];

    for (const selector of selectors) {
      const element = document.querySelector(selector);
      if (element) {
        const text = element.textContent || element.getAttribute('aria-label') || '';
        if (isValidCandidate(text)) {
          return text.trim();
        }
      }
    }

    return null;
  }

  function getDescription() {
    const descElement = document.querySelector('#description-text, ytd-reel-player-overlay-renderer #description-text');
    return descElement ? descElement.textContent.trim() : '';
  }

  function getHashtags() {
    const hashtags = [];
    const hashtagElements = document.querySelectorAll('a[href*="/hashtag/"], ytd-reel-player-header-renderer a[href*="/hashtag/"]');
    hashtagElements.forEach(el => {
      const text = el.textContent.trim();
      if (text.startsWith('#')) {
        hashtags.push(text);
      }
    });
    return hashtags;
  }

  function shouldBlockContent() {
    if (!isEnabled || blockedKeywords.length === 0) {
      return false;
    }

    const title = getTitle().toLowerCase();
    const description = getDescription().toLowerCase();
    const hashtags = getHashtags().join(' ').toLowerCase();

    const allText = `${title} ${description} ${hashtags}`;

    for (const keyword of blockedKeywords) {
      if (allText.includes(keyword.toLowerCase())) {
        return true;
      }
    }

    return false;
  }

  function shouldBlockAudio(audioName) {
    if (!isEnabled || !audioName || blockedAudios.length === 0) {
      return false;
    }

    const audioLower = audioName.toLowerCase();
    return blockedAudios.some(blocked => audioLower.includes(blocked.toLowerCase()));
  }

  const FEED_ITEM_SELECTORS = [
    'ytd-rich-item-renderer',
    'ytd-video-renderer',
    'ytd-compact-video-renderer',
    'ytd-grid-video-renderer',
    'ytd-reel-item-renderer',
    'ytm-shorts-lockup-view-model-v2',
    'yt-lockup-view-model'
  ].join(',');

  function filterFeedItems() {
    if (!isEnabled || blockedKeywords.length === 0) return;

    const items = document.querySelectorAll(FEED_ITEM_SELECTORS);
    items.forEach(item => {
      if (item.dataset.scFiltered) return;

      const text = (item.innerText || '').toLowerCase();
      const isBlocked = blockedKeywords.some(keyword => text.includes(keyword.toLowerCase()));

      item.dataset.scFiltered = '1';
      if (isBlocked) {
        item.style.display = 'none';
        item.dataset.scBlocked = '1';
      }
    });
  }

  function resetFeedFilterCache() {
    document.querySelectorAll('[data-sc-filtered]').forEach(item => {
      delete item.dataset.scFiltered;
      if (item.dataset.scBlocked) {
        item.style.display = '';
        delete item.dataset.scBlocked;
      }
    });
  }

  function scrollToNext() {
    if (!isEnabled) return;

    const container = document.querySelector('#shorts-container, ytd-shorts #shorts-inner-container, ytd-shorts')
      || document.body;
    const video = getVideoElement();

    const wheelEvent = new WheelEvent('wheel', {
      deltaY: 300,
      deltaX: 0,
      deltaMode: 0,
      bubbles: true,
      cancelable: true,
      view: window
    });
    container.dispatchEvent(wheelEvent);

    setTimeout(() => {
      const arrowDownEvent = new KeyboardEvent('keydown', {
        key: 'ArrowDown',
        code: 'ArrowDown',
        keyCode: 40,
        which: 40,
        bubbles: true,
        cancelable: true,
        view: window
      });
      (video || document.activeElement || document).dispatchEvent(arrowDownEvent);
      document.dispatchEvent(arrowDownEvent);
    }, 200);
  }

  async function handleVideoChange(videoId) {
    if (!videoId || !isEnabled) {
      return;
    }

    currentVideoId = videoId;

    if (shouldBlockContent()) {
      setTimeout(() => scrollToNext(), 100);
      return;
    }

    if (!videoData[videoId]) {
      videoData[videoId] = {
        watchCount: 0,
        audio: null
      };
    }

    watchCount = videoData[videoId].watchCount || 0;

    const audioName = extractAudioName();
    if (audioName && videoData[videoId].audio !== audioName) {
      videoData[videoId].audio = audioName;
      await saveVideoData();
    }

    updateWatchCountIndicator();

    if (audioName && shouldBlockAudio(audioName)) {
      setTimeout(() => scrollToNext(), 100);
      return;
    }

    if (watchCount > maxWatchCount) {
      setTimeout(() => scrollToNext(), 1000);
      return;
    }

    startTracking(videoId);
  }

  function createWatchCountIndicator() {
    const existing = document.getElementById('shorts-controller-indicator');
    if (existing) existing.remove();

    const indicator = document.createElement('div');
    indicator.id = 'shorts-controller-indicator';
    indicator.style.cssText = `
      position: fixed;
      top: 80px;
      right: 20px;
      background: rgba(0, 0, 0, 0.8);
      color: white;
      padding: 8px 12px;
      border-radius: 8px;
      font-size: 13px;
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      z-index: 10000;
      pointer-events: none;
      backdrop-filter: blur(10px);
      box-shadow: 0 2px 8px rgba(0,0,0,0.3);
    `;
    document.body.appendChild(indicator);
    return indicator;
  }

  function updateWatchCountIndicator() {
    if (!window.location.pathname.includes('/shorts') && !getVideoId()) {
      const indicator = document.getElementById('shorts-controller-indicator');
      if (indicator) indicator.style.display = 'none';
      return;
    }

    let indicator = document.getElementById('shorts-controller-indicator');
    if (!indicator && currentVideoId && isEnabled) {
      indicator = createWatchCountIndicator();
    }

    if (!indicator) return;

    if (currentVideoId && isEnabled) {
      const count = videoData[currentVideoId]?.watchCount || 0;
      const max = maxWatchCount;
      const percentage = Math.min((count / max) * 100, 100);
      indicator.innerHTML = `
        <div style="font-weight: 600; margin-bottom: 4px;">Watch Count</div>
        <div style="font-size: 18px; font-weight: bold; color: ${count >= max ? '#ff4444' : '#4CAF50'}">
          ${count} / ${max}
        </div>
        <div style="width: 100px; height: 4px; background: rgba(255,255,255,0.2); border-radius: 2px; margin-top: 6px; overflow: hidden;">
          <div style="width: ${percentage}%; height: 100%; background: ${count >= max ? '#ff4444' : '#4CAF50'}; transition: width 0.3s;"></div>
        </div>
      `;
      indicator.style.display = 'block';
    } else {
      indicator.style.display = 'none';
    }
  }

  function startTracking(videoId) {
    const video = getVideoElement();
    if (!video) {
      return;
    }

    const session = ++trackingSession;
    let maxTimeReached = 0;
    let lastCountedAt = 0;

    const isActive = () => isEnabled && session === trackingSession && video.isConnected;

    const countCompletedWatch = () => {
      const now = Date.now();
      if (now - lastCountedAt < 1000) return;
      lastCountedAt = now;
      incrementWatchCount(videoId);
    };

    const onTimeUpdate = () => {
      if (!isActive()) return;

      const currentTime = video.currentTime;
      const duration = video.duration;

      if (currentTime > maxTimeReached) {
        maxTimeReached = currentTime;
      }

      if (currentTime < maxTimeReached - 1) {
        if (duration && maxTimeReached >= duration - 1.5) {
          countCompletedWatch();
        }
        maxTimeReached = currentTime;
      }
    };

    const onEnded = () => {
      if (!isActive()) return;
      countCompletedWatch();
    };

    video.addEventListener('timeupdate', onTimeUpdate);
    video.addEventListener('ended', onEnded);

    const cleanupInterval = setInterval(() => {
      if (!isActive()) {
        video.removeEventListener('timeupdate', onTimeUpdate);
        video.removeEventListener('ended', onEnded);
        clearInterval(cleanupInterval);
      }
    }, CHECK_INTERVAL);

    updateWatchCountIndicator();
  }

  async function incrementWatchCount(videoId) {
    if (!videoId) return;

    if (!videoData[videoId]) {
      videoData[videoId] = { watchCount: 0, audio: null };
    }

    videoData[videoId].watchCount++;
    const newCount = videoData[videoId].watchCount;
    if (videoId === currentVideoId) {
      watchCount = newCount;
    }

    await saveVideoData();
    updateWatchCountIndicator();

    if (newCount >= maxWatchCount) {
      setTimeout(() => {
        if (videoId === currentVideoId && videoData[videoId] && videoData[videoId].watchCount >= maxWatchCount) {
          scrollToNext();
        }
      }, 1500);
    }
  }

  function startObserver() {
    let lastVideoId = getVideoId();
    let lastVideoSrc = null;

    const checkVideoChange = () => {
      const currentVideoId = getVideoId();
      const video = getVideoElement();
      const currentVideoSrc = video ? video.src : null;

      if (currentVideoId && currentVideoId !== lastVideoId) {
        lastVideoId = currentVideoId;
        handleVideoChange(currentVideoId);
      }
      else if (currentVideoSrc && currentVideoSrc !== lastVideoSrc && currentVideoId) {
        lastVideoSrc = currentVideoSrc;
        handleVideoChange(currentVideoId);
      }
    };

    setInterval(() => {
      if (document.hidden) return;
      checkVideoChange();
    }, 500);

    setInterval(() => {
      if (document.hidden) return;
      filterFeedItems();
    }, FEED_FILTER_INTERVAL);

    window.addEventListener('popstate', () => {
      setTimeout(checkVideoChange, 100);
    });

    let mutationCheckPending = false;
    const scheduleMutationCheck = () => {
      if (mutationCheckPending || document.hidden) return;
      mutationCheckPending = true;
      setTimeout(() => {
        mutationCheckPending = false;
        checkVideoChange();
        filterFeedItems();
      }, 200);
    };

    const observer = new MutationObserver(scheduleMutationCheck);

    if (document.body) {
      observer.observe(document.body, {
        childList: true,
        subtree: true
      });
    }

    document.addEventListener('visibilitychange', () => {
      if (!document.hidden) {
        checkVideoChange();
        filterFeedItems();
      }
    });

    const SETTINGS_KEYS = ['enabled', 'maxWatchCount', 'blockedKeywords', 'blockedAudios'];

    chrome.storage.onChanged.addListener((changes, area) => {
      if (area !== 'local') return;
      if (!SETTINGS_KEYS.some(key => key in changes)) return;

      loadSettings().then(() => {
        const videoId = getVideoId();
        if (videoId) {
          handleVideoChange(videoId);
        }
        updateWatchCountIndicator();

        resetFeedFilterCache();
        filterFeedItems();
      });
    });

    chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
      if (request.action === 'getCurrentVideo') {
        const videoId = getVideoId();
        const videoInfo = videoId && videoData[videoId] ? videoData[videoId] : null;
        sendResponse({
          videoId: videoId,
          watchCount: videoInfo ? videoInfo.watchCount : 0,
          audio: videoInfo ? videoInfo.audio : null
        });
        return true;
      }
      if (request.action === 'refreshIndicator') {
        loadVideoData().then(() => {
          updateWatchCountIndicator();
        });
        sendResponse({ success: true });
        return true;
      }
      if (request.action === 'scrollToNext') {
        scrollToNext();
        sendResponse({ success: true });
        return true;
      }
    });

    setTimeout(() => {
      if (lastVideoId) {
        handleVideoChange(lastVideoId);
      }
      filterFeedItems();
    }, 1000);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
