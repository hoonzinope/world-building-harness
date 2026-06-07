package harness

// storyRoomAssetJS is the only story-room client implementation.
// It is served from /assets/story-room.js and loaded by the template via a same-origin script tag.
const storyRoomAssetJS = `(() => {
  const root = document.querySelector('[data-story-room]');
  if (!root) return;
  const progress = root.querySelector('[data-story-progress]');
  if (!progress) return;
  const refreshButton = progress.querySelector('[data-story-refresh]');
  const statusLabel = progress.querySelector('[data-story-progress-label]');
  const statusBadge = progress.querySelector('[data-story-progress-status]');
  const messageNode = progress.querySelector('[data-story-progress-message]');
  const metaNode = progress.querySelector('[data-story-progress-meta]');
  const jobIdNode = progress.querySelector('[data-story-progress-job-id]');
  const jobTypeNode = progress.querySelector('[data-story-progress-job-type]');
  const jobStatusNode = progress.querySelector('[data-story-progress-job-status]');
  const turnNode = progress.querySelector('[data-story-progress-turn]');
  const pendingNode = progress.querySelector('[data-story-progress-pending-count]');
  const stepNodes = Array.from(progress.querySelectorAll('[data-story-step]'));
  const forms = Array.from(root.querySelectorAll('form[data-story-submit]'));
  const inputPanel = root.querySelector('[data-story-input-panel]');
  const storyTurn = Number(root.dataset.currentTurn || 0);
  const initialControlState = new WeakMap();
  let pollTimer = null;
  let activeTask = null;
  let reloadTimer = null;

  function captureInitialControlState(control) {
    if (!control || control.type === 'hidden') return;
    if (!initialControlState.has(control)) {
      initialControlState.set(control, {
        disabled: control.disabled,
        ariaDisabled: control.getAttribute('aria-disabled'),
      });
    }
  }

  function restoreInitialControlState(control) {
    if (!control || control.type === 'hidden') return;
    const initial = initialControlState.get(control);
    if (!initial) return;
    control.disabled = initial.disabled;
    if (initial.disabled) {
      control.setAttribute('aria-disabled', initial.ariaDisabled ?? 'true');
    } else if (initial.ariaDisabled === null) {
      control.removeAttribute('aria-disabled');
    } else {
      control.setAttribute('aria-disabled', initial.ariaDisabled);
    }
    if (control.tagName === 'BUTTON' && control.dataset.storyOriginalHtml) {
      control.innerHTML = control.dataset.storyOriginalHtml;
      delete control.dataset.storyOriginalHtml;
    }
  }

  forms.forEach((form) => {
    form.querySelectorAll('button, input, select, textarea').forEach(captureInitialControlState);
  });

  function setStep(stepLabel) {
    const stepOrder = ['queued', 'generating', 'applying', 'ready'];
    const activeIndex = stepOrder.indexOf(stepLabel);
    stepNodes.forEach((node) => {
      const active = node.dataset.storyStep === stepLabel;
      const nodeIndex = stepOrder.indexOf(node.dataset.storyStep);
      if (active) {
        node.setAttribute('aria-current', 'step');
      } else {
        node.removeAttribute('aria-current');
      }
      node.classList.toggle('is-active', active);
      node.classList.toggle('is-complete', activeIndex >= 0 && nodeIndex >= 0 && nodeIndex < activeIndex);
    });
  }

  function friendlyStepLabel(stepLabel) {
    switch (stepLabel) {
      case 'queued':
        return '대기열';
      case 'generating':
        return '생성 중';
      case 'applying':
        return '반영 중';
      case 'ready':
        return '입력 가능';
      case 'failed':
        return '실패';
      default:
        return stepLabel || '';
    }
  }

  function setBusy(busy) {
    root.setAttribute('aria-busy', busy ? 'true' : 'false');
    progress.setAttribute('aria-busy', busy ? 'true' : 'false');
    if (inputPanel) {
      inputPanel.setAttribute('aria-busy', busy ? 'true' : 'false');
    }
    forms.forEach((form) => {
      form.querySelectorAll('button, input, select, textarea').forEach((control) => {
        if (control.type === 'hidden') return;
        captureInitialControlState(control);
        if (busy) {
          control.disabled = true;
          control.setAttribute('aria-disabled', 'true');
          if (control.tagName === 'BUTTON' && !control.hasAttribute('data-story-choice-button')) {
            if (!control.dataset.storyOriginalHtml) control.dataset.storyOriginalHtml = control.innerHTML;
            control.innerHTML = '처리 중...';
          }
        } else {
          restoreInitialControlState(control);
        }
      });
    });
  }

  function showRefresh(visible) {
    if (!refreshButton) return;
    refreshButton.hidden = !visible;
  }

  function showMeta(visible) {
    if (!metaNode) return;
    metaNode.hidden = !visible;
  }

  function getReloadTarget(payload, task) {
    const payloadTurn = Number(payload && payload.current_turn ? payload.current_turn : 0);
    const activeTurn = Number(payload && payload.active_job_turn_id ? payload.active_job_turn_id : 0);
    const completedTurn = Number(payload && payload.last_completed_job_turn_id ? payload.last_completed_job_turn_id : 0);
    const submittedTurn = Number(task && task.turn_id ? task.turn_id : 0);
    const nextTurn = Math.max(storyTurn, payloadTurn, activeTurn, completedTurn, submittedTurn);
    const completedType = (payload && payload.last_completed_job_type) || (task && task.job_type) || '';
    if (completedType === 'question_answer' || nextTurn <= storyTurn) {
      return '#qa';
    }
    return '#turn-' + nextTurn;
  }

  function scheduleStoryReload(payload, task) {
    if (reloadTimer) {
      window.clearTimeout(reloadTimer);
      reloadTimer = null;
    }
    const target = getReloadTarget(payload, task);
    if (window.location.hash !== target) {
      window.history.replaceState(null, '', target);
    }
    reloadTimer = window.setTimeout(() => window.location.reload(), 420);
  }

  async function readErrorMessage(response) {
    const contentType = (response.headers.get('content-type') || '').toLowerCase();
    if (contentType.includes('application/json')) {
      const payload = await response.json().catch(() => null);
      if (payload && payload.error) {
        return payload.error;
      }
      return 'HTTP ' + response.status + ' - 제출 응답을 JSON으로 받지 못했습니다';
    }
    const text = await response.text().catch(() => '');
    const snippet = text.trim().replace(/\s+/g, ' ').slice(0, 160);
    return 'HTTP ' + response.status + ' - 제출 응답을 JSON으로 받지 못했습니다' + (snippet ? ': ' + snippet : '');
  }

  function renderStatus(payload) {
    progress.dataset.stepIndex = String(payload.step_index ?? 3);
    progress.dataset.stepLabel = payload.step_label || 'ready';
    progress.dataset.activeJobId = payload.active_job_id || '';
    progress.dataset.activeJobStatus = payload.active_job_status || '';
    progress.dataset.activeJobType = payload.active_job_type || '';
    progress.dataset.nextPollMs = String(payload.next_poll_ms || 0);
    if (statusLabel) statusLabel.textContent = friendlyStepLabel(payload.step_label || (payload.is_processing ? 'generating' : 'ready'));
    if (statusBadge) statusBadge.textContent = payload.status_label || '';
    if (messageNode) messageNode.textContent = payload.progress_message || '';
    if (jobIdNode) jobIdNode.textContent = payload.active_job_id || '';
    if (jobTypeNode) jobTypeNode.textContent = payload.active_job_type || '';
    if (jobStatusNode) jobStatusNode.textContent = payload.active_job_status || '';
    if (turnNode) turnNode.textContent = payload.active_job_turn_id ? String(payload.active_job_turn_id) : '';
    if (pendingNode) pendingNode.textContent = payload.pending_questions ? String(payload.pending_questions.length) : '';
    showMeta(false);
    setStep(payload.step_label || (payload.is_processing ? 'generating' : 'ready'));
    setBusy(Boolean(payload.is_processing));
  }

  async function pollStatus() {
    if (!activeTask || !activeTask.status_url) return;
    try {
      const response = await fetch(activeTask.status_url, {
        headers: {
          Accept: 'application/json',
          'X-Requested-With': 'XMLHttpRequest',
        },
        credentials: 'include',
      });
      if (!response.ok) {
        throw new Error(await readErrorMessage(response));
      }
      const payload = await response.json();
      renderStatus(payload);
      const nextPoll = Number(payload.next_poll_ms || activeTask.next_poll_ms || 2500);
      if (payload.is_processing) {
        pollTimer = window.setTimeout(pollStatus, Math.max(1000, nextPoll));
        return;
      }
      showRefresh(true);
      const completedType = payload.last_completed_job_type || activeTask.job_type || payload.active_job_type || '';
      const completedStoryTurn = completedType === 'story_turn' && (
        Number(payload.current_turn || 0) > storyTurn ||
        Number(payload.last_completed_job_turn_id || 0) > storyTurn
      );
      const completedQuestion = completedType === 'question_answer';
      activeTask = null;
      if (payload.active_job_status !== 'failed' && (completedStoryTurn || completedQuestion || Number(payload.current_turn || 0) > storyTurn)) {
        if (messageNode) messageNode.textContent = payload.progress_message || '새 내용이 준비되었습니다. 자동으로 최신 화면을 불러옵니다.';
        scheduleStoryReload(payload, {
          job_type: completedType,
          turn_id: completedStoryTurn ? Number(payload.last_completed_job_turn_id || payload.current_turn || 0) : Number(payload.last_completed_job_turn_id || 0),
        });
      }
    } catch (error) {
      if (messageNode) messageNode.textContent = '상태를 다시 불러오지 못했습니다. 잠시 후 다시 시도해 주세요.';
      pollTimer = window.setTimeout(pollStatus, 2500);
    }
  }

  async function submitForm(form, submitter) {
    if (pollTimer) {
      window.clearTimeout(pollTimer);
      pollTimer = null;
    }
    const data = new FormData(form);
    const requestPayload = Object.fromEntries(data.entries());
    const actionURL = new URL(form.action, window.location.href);
    const requestURL = actionURL.origin === window.location.origin ? actionURL.pathname + actionURL.search : form.action;
    setBusy(true);
    progress.dataset.stepIndex = '0';
    progress.dataset.stepLabel = 'queued';
    if (statusLabel) statusLabel.textContent = friendlyStepLabel('queued');
    setStep('queued');
    if (messageNode) messageNode.textContent = '제출을 보냈습니다. 서버 응답을 기다립니다.';
    showRefresh(false);
    try {
      const response = await fetch(requestURL, {
        method: (form.method || 'post').toUpperCase(),
        body: JSON.stringify(requestPayload),
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          Accept: 'application/json',
          'X-Requested-With': 'XMLHttpRequest',
          'X-CSRF-Token': form.querySelector('input[name="csrf_token"]')?.value || '',
        },
      });
      const contentType = (response.headers.get('content-type') || '').toLowerCase();
      const rawBody = await response.text();
      let responsePayload = null;
      if (contentType.includes('application/json')) {
        try {
          responsePayload = JSON.parse(rawBody);
        } catch (parseError) {
          responsePayload = null;
        }
      }
      if (!response.ok || !responsePayload) {
        if (responsePayload && responsePayload.error) {
          throw new Error(responsePayload.error);
        }
        const snippet = rawBody.trim().replace(/\s+/g, ' ').slice(0, 160);
        throw new Error('HTTP ' + response.status + ' - 제출 응답을 JSON으로 받지 못했습니다' + (snippet ? ': ' + snippet : ''));
      }
      activeTask = {
        status_url: responsePayload.status_url,
        next_poll_ms: responsePayload.next_poll_ms || 2500,
        turn_id: responsePayload.turn_id || 0,
        job_id: responsePayload.job_id || '',
        job_type: responsePayload.job_type || '',
      };
      renderStatus(responsePayload);
      pollTimer = window.setTimeout(pollStatus, Math.max(1000, activeTask.next_poll_ms || 2500));
    } catch (error) {
      setBusy(false);
      showRefresh(false);
      if (messageNode) messageNode.textContent = error.message || '제출 처리에 실패했습니다.';
    }
  }

  root.addEventListener('submit', (event) => {
    const form = event.target.closest('form[data-story-submit]');
    if (!form) return;
    event.preventDefault();
    submitForm(form, event.submitter || null);
  });

  if (refreshButton) {
    refreshButton.addEventListener('click', () => window.location.reload());
  }

  if (statusLabel) statusLabel.textContent = friendlyStepLabel(progress.dataset.stepLabel || (root.dataset.initialProcessing === 'true' ? 'generating' : 'ready'));
  setStep(progress.dataset.stepLabel || (root.dataset.initialProcessing === 'true' ? 'generating' : 'ready'));
  setBusy(root.dataset.initialProcessing === 'true');
  if (root.dataset.initialProcessing === 'true') {
    activeTask = {
      status_url: progress.dataset.statusUrl || root.dataset.statusUrl,
      next_poll_ms: Number(progress.dataset.nextPollMs || 2500),
    };
    pollTimer = window.setTimeout(pollStatus, Math.max(1000, activeTask.next_poll_ms || 2500));
  }
})();`
