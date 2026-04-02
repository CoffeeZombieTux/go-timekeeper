const STORAGE_KEY = "gotimekeeper.localClient.state.v1";

function detectTimezone() {
  try {
    const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
    return tz || "UTC";
  } catch {
    return "UTC";
  }
}

function toDateInputValue(date) {
  return date.toISOString().slice(0, 10);
}

function toDateTimeLocalValue(date) {
  const pad = (value) => String(value).padStart(2, "0");
  return [
    date.getFullYear(),
    pad(date.getMonth() + 1),
    pad(date.getDate()),
  ].join("-") + "T" + [pad(date.getHours()), pad(date.getMinutes())].join(":");
}

function parseJSON(text) {
  try {
    return JSON.parse(text);
  } catch {
    return null;
  }
}

function formatJSON(payload) {
  return JSON.stringify(payload, null, 2);
}

function parseLocalDateTime(value) {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    throw new Error(`Invalid datetime value: ${value}`);
  }
  return parsed;
}

function getWorkDateFromISO(isoValue, timezone) {
  const date = new Date(isoValue);
  if (Number.isNaN(date.getTime())) {
    throw new Error("Failed to calculate workDate from start time.");
  }
  const parts = new Intl.DateTimeFormat("en-CA", {
    timeZone: timezone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).formatToParts(date);

  const year = parts.find((part) => part.type === "year")?.value;
  const month = parts.find((part) => part.type === "month")?.value;
  const day = parts.find((part) => part.type === "day")?.value;

  if (!year || !month || !day) {
    throw new Error("Unable to resolve workDate parts.");
  }

  return `${year}-${month}-${day}T00:00:00Z`;
}

function getEl(id) {
  return document.getElementById(id);
}

const now = new Date();
const oneHourBack = new Date(now.getTime() - 60 * 60 * 1000);
const firstOfMonth = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), 1));

const DEFAULT_STATE = {
  baseUrl: "http://localhost:8080",
  timezone: detectTimezone(),
  email: "demo.user@example.com",
  password: "Demo@12345",
  newPassword: "Demo@12345New",
  accessToken: "",
  refreshToken: "",
  projectName: "",
  taskName: "",
  selectedProjectId: "",
  selectedTaskId: "",
  taskListIsActive: "",
  taskListLimit: "100",
  taskListOffset: "0",
  timeRecordId: "",
  startTimeLocal: toDateTimeLocalValue(oneHourBack),
  endTimeLocal: toDateTimeLocalValue(now),
  reportFromDate: toDateInputValue(firstOfMonth),
  reportToDate: toDateInputValue(now),
};

const PERSISTED_INPUT_IDS = Object.keys(DEFAULT_STATE);

const runtime = {
  projects: [],
  tasks: [],
  reportKind: "",
  reportData: null,
  currentUser: null,
  lastRequest: null,
  lastResponse: null,
  busy: false,
};

function loadState() {
  const raw = localStorage.getItem(STORAGE_KEY);
  const parsed = raw ? parseJSON(raw) : null;
  return { ...DEFAULT_STATE, ...(parsed || {}) };
}

function readState() {
  const state = loadState();
  for (const id of PERSISTED_INPUT_IDS) {
    const element = getEl(id);
    if (element) {
      state[id] = String(element.value || "").trim();
    }
  }
  return state;
}

function writeState(state) {
  const merged = { ...DEFAULT_STATE, ...(state || {}) };
  for (const id of PERSISTED_INPUT_IDS) {
    const element = getEl(id);
    if (!element) {
      continue;
    }
    element.value = merged[id] ?? "";
  }
}

function persistState(state) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
}

function saveStateFromInputs() {
  persistState(readState());
}

function setStatus(message, isError = false) {
  const element = getEl("statusLine");
  const timestamp = new Date().toLocaleTimeString();
  element.textContent = `[${timestamp}] ${message}`;
  element.classList.toggle("status-error", isError);
}

function setBusy(isBusy) {
  runtime.busy = isBusy;
  for (const button of document.querySelectorAll("[data-busy='1']")) {
    button.disabled = isBusy;
  }
}

function endpointUrl(baseUrl, path) {
  return `${baseUrl.replace(/\/+$/, "")}${path}`;
}

function formatShortID(value) {
  if (!value) {
    return "";
  }
  return `${value.slice(0, 8)}...${value.slice(-4)}`;
}

function formatDateTime(value) {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return "N/A";
  }
  return parsed.toLocaleString();
}

function formatMinutes(minutes) {
  const hrs = Math.floor(minutes / 60);
  const mins = minutes % 60;
  if (hrs === 0) {
    return `${mins}m`;
  }
  return `${hrs}h ${mins}m`;
}

function renderTrace() {
  getEl("lastRequest").textContent = runtime.lastRequest
    ? formatJSON(runtime.lastRequest)
    : "No requests yet.";
  getEl("lastResponse").textContent = runtime.lastResponse
    ? formatJSON(runtime.lastResponse)
    : "No responses yet.";
}

function renderAuthBadge() {
  const badge = getEl("authBadge");
  const state = readState();
  badge.className = "badge";

  if (!state.accessToken) {
    badge.classList.add("badge-muted");
    badge.textContent = "Not authenticated";
    return;
  }

  const userEmail = runtime.currentUser?.email || state.email || "Authenticated";
  const userId = runtime.currentUser?.id ? ` · ${formatShortID(runtime.currentUser.id)}` : "";
  badge.classList.add("badge-ok");
  badge.textContent = `${userEmail}${userId}`;
}

function getSelectedProject() {
  const state = readState();
  return runtime.projects.find((project) => project.id === state.selectedProjectId) || null;
}

function getSelectedTask() {
  const state = readState();
  return runtime.tasks.find((task) => task.id === state.selectedTaskId) || null;
}

function renderSelectionBadge() {
  const project = getSelectedProject();
  const task = getSelectedTask();
  const badge = getEl("selectedBadge");
  badge.className = "badge";

  if (!project) {
    badge.classList.add("badge-muted");
    badge.textContent = "No project selected";
    return;
  }

  badge.classList.add("badge-ok");
  if (!task) {
    badge.textContent = `Project: ${project.name}`;
    return;
  }

  badge.textContent = `Project: ${project.name} · Task: ${task.name}`;
}

function syncProjectAndTaskTextFields() {
  const state = readState();
  const selectedProject = runtime.projects.find((project) => project.id === state.selectedProjectId);
  const selectedTask = runtime.tasks.find((task) => task.id === state.selectedTaskId);

  if (selectedProject) {
    state.projectName = selectedProject.name;
  }
  if (selectedTask) {
    state.taskName = selectedTask.name;
  }

  writeState(state);
  persistState(state);
}

function renderProjects() {
  const state = readState();
  const list = getEl("projectList");
  getEl("projectCount").textContent = String(runtime.projects.length);

  if (runtime.projects.length === 0) {
    list.innerHTML = `<li class="entity-item entity-empty">No projects loaded.</li>`;
    renderSelectionBadge();
    return;
  }

  list.innerHTML = runtime.projects
    .map((project) => {
      const selectedClass = project.id === state.selectedProjectId ? "is-selected" : "";
      return `
      <li class="entity-item ${selectedClass}">
        <button class="entity-btn" data-project-id="${project.id}">
          <span>${project.name}</span>
          <span class="mono">${formatShortID(project.id)}</span>
        </button>
      </li>`;
    })
    .join("");

  renderSelectionBadge();
}

function getStatusClass(status) {
  switch (status) {
    case "WORKING_ON":
      return "status status-active";
    case "CLOSED":
      return "status status-closed";
    default:
      return "status status-idle";
  }
}

function renderTasks() {
  const state = readState();
  const body = getEl("taskTableBody");
  getEl("taskCount").textContent = String(runtime.tasks.length);

  if (!state.selectedProjectId) {
    body.innerHTML = `<tr><td colspan="4" class="empty-cell">Select a project first.</td></tr>`;
    renderSelectionBadge();
    return;
  }

  if (runtime.tasks.length === 0) {
    body.innerHTML = `<tr><td colspan="4" class="empty-cell">No tasks for this project.</td></tr>`;
    renderSelectionBadge();
    return;
  }

  body.innerHTML = runtime.tasks
    .map((task) => {
      const selectedClass = task.id === state.selectedTaskId ? "row-selected" : "";
      return `
      <tr class="${selectedClass}">
        <td>${task.name}</td>
        <td><span class="${getStatusClass(task.status)}">${task.status}</span></td>
        <td>${formatDateTime(task.updatedAt)}</td>
        <td class="table-actions">
          <button data-task-action="select" data-task-id="${task.id}" class="btn-xs">Use</button>
          <button data-task-action="start" data-task-id="${task.id}" class="btn-xs">Start</button>
          <button data-task-action="stop" data-task-id="${task.id}" class="btn-xs btn-secondary">Stop</button>
          <button data-task-action="close" data-task-id="${task.id}" class="btn-xs btn-ghost">Close</button>
        </td>
      </tr>`;
    })
    .join("");

  renderSelectionBadge();
}

function renderReport() {
  const summary = getEl("reportSummary");
  const json = getEl("reportJson");

  if (!runtime.reportData) {
    summary.innerHTML = `<p class="helper-text">No report loaded.</p>`;
    json.textContent = "No report loaded.";
    return;
  }

  const data = runtime.reportData;
  json.textContent = formatJSON(data);

  if (runtime.reportKind === "general") {
    const projects = Array.isArray(data.projects) ? data.projects : [];
    const projectItems = projects
      .map((project) => `<li>${project.projectName}: <strong>${formatMinutes(project.totalMinutes || 0)}</strong></li>`)
      .join("");
    summary.innerHTML = `
      <div class="metric-row">
        <div class="metric"><span>Total</span><strong>${formatMinutes(data.totalMinutes || 0)}</strong></div>
        <div class="metric"><span>Projects</span><strong>${projects.length}</strong></div>
      </div>
      <ul class="summary-list">${projectItems || "<li>No project rows.</li>"}</ul>
    `;
    return;
  }

  if (runtime.reportKind === "project") {
    const tasks = Array.isArray(data.tasks) ? data.tasks : [];
    const taskItems = tasks
      .map((task) => `<li>${task.taskName}: <strong>${formatMinutes(task.totalMinutes || 0)}</strong></li>`)
      .join("");
    summary.innerHTML = `
      <div class="metric-row">
        <div class="metric"><span>Project</span><strong>${data.projectName || "N/A"}</strong></div>
        <div class="metric"><span>Total</span><strong>${formatMinutes(data.totalMinutes || 0)}</strong></div>
      </div>
      <ul class="summary-list">${taskItems || "<li>No task rows.</li>"}</ul>
    `;
    return;
  }

  const dayReports = Array.isArray(data.dayReports) ? data.dayReports : [];
  const dayItems = dayReports
    .map((day) => `<li>${day.workingDate}: <strong>${formatMinutes(day.totalMinutes || 0)}</strong></li>`)
    .join("");
  summary.innerHTML = `
    <div class="metric-row">
      <div class="metric"><span>Task</span><strong>${data.taskName || "N/A"}</strong></div>
      <div class="metric"><span>Total</span><strong>${formatMinutes(data.totalMinutes || 0)}</strong></div>
    </div>
    <ul class="summary-list">${dayItems || "<li>No day rows.</li>"}</ul>
  `;
}

function getErrorMessage(label, responseStatus, payload) {
  const message = payload?.message || `${label} failed with HTTP ${responseStatus}`;
  const details = payload?.error?.details;
  if (!Array.isArray(details) || details.length === 0) {
    return message;
  }
  const mapped = details
    .map((detail) => detail.reason || detail.field)
    .filter(Boolean)
    .join("; ");
  return mapped ? `${message}: ${mapped}` : message;
}

async function apiRequest({ label, method, path, body, authRequired = true, headers = {} }) {
  const state = readState();
  const url = endpointUrl(state.baseUrl, path);
  const requestHeaders = { ...headers };

  if (body !== undefined) {
    requestHeaders["Content-Type"] = "application/json";
  }

  if (authRequired) {
    if (!state.accessToken) {
      throw new Error("Access token is empty. Login or register first.");
    }
    requestHeaders.Authorization = `Bearer ${state.accessToken}`;
  }

  runtime.lastRequest = {
    at: new Date().toISOString(),
    label,
    method,
    url,
    headers: requestHeaders,
    body: body ?? null,
  };
  renderTrace();

  const response = await fetch(url, {
    method,
    headers: requestHeaders,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  const text = await response.text();
  const payload = parseJSON(text) || { raw: text };
  runtime.lastResponse = {
    at: new Date().toISOString(),
    label,
    status: response.status,
    ok: response.ok,
    payload,
  };
  renderTrace();

  if (!response.ok) {
    throw new Error(getErrorMessage(label, response.status, payload));
  }

  if (payload?.success === false) {
    throw new Error(getErrorMessage(label, response.status, payload));
  }

  return payload?.data;
}

function updateTokensAndUser(authData) {
  const state = readState();
  if (authData?.accessToken) {
    state.accessToken = authData.accessToken;
  }
  if (authData?.refreshToken) {
    state.refreshToken = authData.refreshToken;
  }
  if (authData?.user) {
    runtime.currentUser = authData.user;
  }
  writeState(state);
  persistState(state);
  renderAuthBadge();
}

function clearAuthLocal() {
  const state = readState();
  state.accessToken = "";
  state.refreshToken = "";
  runtime.currentUser = null;
  runtime.projects = [];
  runtime.tasks = [];
  writeState(state);
  persistState(state);
  renderAuthBadge();
  renderProjects();
  renderTasks();
}

function ensureProjectSelected() {
  const selected = getSelectedProject();
  if (!selected) {
    throw new Error("Select a project first.");
  }
  return selected;
}

function ensureTaskSelected() {
  const selected = getSelectedTask();
  if (!selected) {
    throw new Error("Select a task first.");
  }
  return selected;
}

function setSelectedProject(projectId) {
  const state = readState();
  state.selectedProjectId = projectId || "";
  state.selectedTaskId = "";
  writeState(state);
  persistState(state);
  syncProjectAndTaskTextFields();
  renderProjects();
  renderTasks();
}

function setSelectedTask(taskId) {
  const state = readState();
  state.selectedTaskId = taskId || "";
  writeState(state);
  persistState(state);
  syncProjectAndTaskTextFields();
  renderTasks();
}

async function loadCurrentUser() {
  const me = await apiRequest({
    label: "Get Me",
    method: "GET",
    path: "/api/user/me",
  });
  runtime.currentUser = me || null;
  renderAuthBadge();
}

async function loadProjects() {
  const data = await apiRequest({
    label: "List Projects",
    method: "GET",
    path: "/api/project/list",
  });

  runtime.projects = Array.isArray(data) ? data : [];
  const state = readState();
  const selectedExists = runtime.projects.some((project) => project.id === state.selectedProjectId);

  if (!selectedExists) {
    state.selectedProjectId = runtime.projects[0]?.id || "";
    state.selectedTaskId = "";
  }

  writeState(state);
  persistState(state);
  syncProjectAndTaskTextFields();
  renderProjects();
}

async function loadTasks() {
  const state = readState();
  if (!state.selectedProjectId) {
    runtime.tasks = [];
    renderTasks();
    return;
  }

  const query = new URLSearchParams();
  if (state.taskListLimit) {
    query.set("limit", state.taskListLimit);
  }
  if (state.taskListOffset) {
    query.set("offset", state.taskListOffset);
  }
  if (state.taskListIsActive !== "") {
    query.set("isActive", state.taskListIsActive);
  }
  const suffix = query.toString() ? `?${query.toString()}` : "";

  const data = await apiRequest({
    label: "List Project Tasks",
    method: "GET",
    path: `/api/task/list/project/${state.selectedProjectId}${suffix}`,
  });

  runtime.tasks = Array.isArray(data?.tasks) ? data.tasks : [];
  const selectedExists = runtime.tasks.some((task) => task.id === state.selectedTaskId);
  if (!selectedExists) {
    state.selectedTaskId = runtime.tasks[0]?.id || "";
  }

  writeState(state);
  persistState(state);
  syncProjectAndTaskTextFields();
  renderTasks();
}

async function syncWorkspace() {
  const state = readState();
  if (!state.accessToken) {
    runtime.projects = [];
    runtime.tasks = [];
    runtime.currentUser = null;
    renderAuthBadge();
    renderProjects();
    renderTasks();
    return;
  }

  await loadCurrentUser();
  await loadProjects();
  await loadTasks();
}

async function runBusy(label, fn) {
  if (runtime.busy) {
    return;
  }
  setBusy(true);
  try {
    await fn();
    setStatus(`${label} succeeded.`);
  } catch (error) {
    setStatus(error instanceof Error ? error.message : String(error), true);
  } finally {
    setBusy(false);
  }
}

function getTimeRangeBody() {
  const state = readState();
  if (!state.reportFromDate || !state.reportToDate) {
    throw new Error("Both report dates are required.");
  }
  return {
    fromDate: `${state.reportFromDate}T00:00:00Z`,
    toDate: `${state.reportToDate}T23:59:59Z`,
  };
}

async function onPing() {
  await apiRequest({
    label: "Ping",
    method: "GET",
    path: "/ping",
    authRequired: false,
  });
}

async function onRegister() {
  const state = readState();
  const data = await apiRequest({
    label: "Register",
    method: "POST",
    path: "/api/auth/register",
    authRequired: false,
    body: { email: state.email, password: state.password },
  });
  updateTokensAndUser(data);
  await syncWorkspace();
}

async function onLogin() {
  const state = readState();
  const data = await apiRequest({
    label: "Login",
    method: "POST",
    path: "/api/auth/login",
    authRequired: false,
    body: { email: state.email, password: state.password },
  });
  updateTokensAndUser(data);
  await syncWorkspace();
}

async function onRefresh() {
  const state = readState();
  const data = await apiRequest({
    label: "Refresh Tokens",
    method: "POST",
    path: "/api/auth/refresh",
    authRequired: false,
    body: { refreshToken: state.refreshToken },
  });
  updateTokensAndUser(data);
}

async function onLogout() {
  const state = readState();
  await apiRequest({
    label: "Logout",
    method: "POST",
    path: "/api/auth/logout",
    body: { refreshToken: state.refreshToken },
  });
  clearAuthLocal();
  runtime.reportData = null;
  runtime.reportKind = "";
  renderReport();
}

async function onGetMe() {
  await loadCurrentUser();
}

async function onChangePassword() {
  const state = readState();
  await apiRequest({
    label: "Change Password",
    method: "POST",
    path: "/api/auth/change-password",
    body: {
      currentPassword: state.password,
      newPassword: state.newPassword,
    },
  });
}

async function onCreateProject() {
  const state = readState();
  if (!state.projectName) {
    throw new Error("Project name is required.");
  }
  const project = await apiRequest({
    label: "Create Project",
    method: "POST",
    path: "/api/project",
    body: { name: state.projectName },
  });
  if (project?.id) {
    state.selectedProjectId = project.id;
    writeState(state);
    persistState(state);
  }
  await loadProjects();
  await loadTasks();
}

async function onUpdateProject() {
  const project = ensureProjectSelected();
  const state = readState();
  if (!state.projectName) {
    throw new Error("Project name is required.");
  }
  await apiRequest({
    label: "Update Project",
    method: "PATCH",
    path: "/api/project",
    body: { id: project.id, name: state.projectName },
  });
  await loadProjects();
}

async function onDeleteProject() {
  const project = ensureProjectSelected();
  if (!window.confirm(`Delete project "${project.name}"?`)) {
    return;
  }
  await apiRequest({
    label: "Delete Project",
    method: "DELETE",
    path: `/api/project/${project.id}`,
  });
  await loadProjects();
  await loadTasks();
}

async function onCreateTask() {
  const project = ensureProjectSelected();
  const state = readState();
  if (!state.taskName) {
    throw new Error("Task name is required.");
  }
  const task = await apiRequest({
    label: "Create Task",
    method: "POST",
    path: "/api/task",
    body: { name: state.taskName, projectId: project.id },
  });
  if (task?.id) {
    state.selectedTaskId = task.id;
    writeState(state);
    persistState(state);
  }
  await loadTasks();
}

async function onUpdateTask() {
  const project = ensureProjectSelected();
  const task = ensureTaskSelected();
  const state = readState();
  if (!state.taskName) {
    throw new Error("Task name is required.");
  }
  await apiRequest({
    label: "Update Task",
    method: "PATCH",
    path: "/api/task",
    body: {
      id: task.id,
      projectId: project.id,
      name: state.taskName,
    },
  });
  await loadTasks();
}

async function onDeleteTask() {
  const task = ensureTaskSelected();
  if (!window.confirm(`Delete task "${task.name}"?`)) {
    return;
  }
  await apiRequest({
    label: "Delete Task",
    method: "DELETE",
    path: `/api/task/${task.id}`,
  });
  await loadTasks();
}

async function onStartTask() {
  const task = ensureTaskSelected();
  const state = readState();
  await apiRequest({
    label: "Start Task",
    method: "PATCH",
    path: `/api/task/${task.id}/start`,
    headers: { "X-Timezone": state.timezone },
  });
  await loadTasks();
}

async function onStopTask() {
  const task = ensureTaskSelected();
  await apiRequest({
    label: "Stop Task",
    method: "PATCH",
    path: `/api/task/${task.id}/stop`,
  });
  await loadTasks();
}

async function onCloseTask() {
  const task = ensureTaskSelected();
  await apiRequest({
    label: "Close Task",
    method: "PATCH",
    path: `/api/task/${task.id}/close`,
  });
  await loadTasks();
}

function buildManualSessionPayload(isUpdate = false) {
  const state = readState();
  const project = ensureProjectSelected();
  const task = ensureTaskSelected();
  const startTime = parseLocalDateTime(state.startTimeLocal).toISOString();
  const endTime = parseLocalDateTime(state.endTimeLocal).toISOString();

  if (new Date(endTime) <= new Date(startTime)) {
    throw new Error("End time must be after start time.");
  }

  const payload = {
    projectId: project.id,
    taskId: task.id,
    workDate: getWorkDateFromISO(startTime, state.timezone),
    workTimezone: state.timezone,
    startTime,
    endTime,
  };

  if (isUpdate) {
    if (!state.timeRecordId) {
      throw new Error("Time Record ID is required for update.");
    }
    payload.id = state.timeRecordId;
  }

  return payload;
}

async function onCreateTimeRecord() {
  const record = await apiRequest({
    label: "Create Time Record",
    method: "POST",
    path: "/api/task/session",
    body: buildManualSessionPayload(false),
  });
  if (record?.id) {
    const state = readState();
    state.timeRecordId = record.id;
    writeState(state);
    persistState(state);
  }
}

async function onUpdateTimeRecord() {
  await apiRequest({
    label: "Update Time Record",
    method: "PATCH",
    path: "/api/task/session",
    body: buildManualSessionPayload(true),
  });
}

async function onDeleteTimeRecord() {
  const state = readState();
  if (!state.timeRecordId) {
    throw new Error("Time Record ID is required for delete.");
  }
  await apiRequest({
    label: "Delete Time Record",
    method: "DELETE",
    path: `/api/task/session/${state.timeRecordId}`,
  });
  state.timeRecordId = "";
  writeState(state);
  persistState(state);
}

async function onGeneralReport() {
  const state = readState();
  const body = { timeRange: getTimeRangeBody() };
  if (state.selectedProjectId) {
    body.projects = [state.selectedProjectId];
  }
  runtime.reportData = await apiRequest({
    label: "General Report",
    method: "POST",
    path: "/api/report/general",
    body,
  });
  runtime.reportKind = "general";
  renderReport();
}

async function onProjectReport() {
  const project = ensureProjectSelected();
  runtime.reportData = await apiRequest({
    label: "Project Report",
    method: "POST",
    path: "/api/report/project",
    body: {
      projectId: project.id,
      timeRange: getTimeRangeBody(),
    },
  });
  runtime.reportKind = "project";
  renderReport();
}

async function onTaskReport() {
  const task = ensureTaskSelected();
  runtime.reportData = await apiRequest({
    label: "Task Report",
    method: "POST",
    path: "/api/report/task",
    body: {
      taskId: task.id,
      timeRange: getTimeRangeBody(),
    },
  });
  runtime.reportKind = "task";
  renderReport();
}

function bindButtonHandlers() {
  getEl("pingBtn").addEventListener("click", () => runBusy("Ping", onPing));
  getEl("syncBtn").addEventListener("click", () => runBusy("Sync Data", syncWorkspace));
  getEl("resetBtn").addEventListener("click", () =>
    runBusy("Reset Local State", async () => {
      const reset = { ...DEFAULT_STATE };
      writeState(reset);
      persistState(reset);
      runtime.projects = [];
      runtime.tasks = [];
      runtime.reportData = null;
      runtime.reportKind = "";
      runtime.currentUser = null;
      runtime.lastRequest = null;
      runtime.lastResponse = null;
      renderAuthBadge();
      renderProjects();
      renderTasks();
      renderReport();
      renderTrace();
    }),
  );

  getEl("registerBtn").addEventListener("click", () => runBusy("Register", onRegister));
  getEl("loginBtn").addEventListener("click", () => runBusy("Login", onLogin));
  getEl("refreshBtn").addEventListener("click", () => runBusy("Refresh", onRefresh));
  getEl("logoutBtn").addEventListener("click", () => runBusy("Logout", onLogout));
  getEl("meBtn").addEventListener("click", () => runBusy("Get Me", onGetMe));
  getEl("changePasswordBtn").addEventListener("click", () => runBusy("Change Password", onChangePassword));

  getEl("createProjectBtn").addEventListener("click", () => runBusy("Create Project", onCreateProject));
  getEl("updateProjectBtn").addEventListener("click", () => runBusy("Update Project", onUpdateProject));
  getEl("deleteProjectBtn").addEventListener("click", () => runBusy("Delete Project", onDeleteProject));
  getEl("refreshProjectsBtn").addEventListener("click", () =>
    runBusy("Refresh Projects", async () => {
      await loadProjects();
      await loadTasks();
    }),
  );

  getEl("createTaskBtn").addEventListener("click", () => runBusy("Create Task", onCreateTask));
  getEl("updateTaskBtn").addEventListener("click", () => runBusy("Update Task", onUpdateTask));
  getEl("deleteTaskBtn").addEventListener("click", () => runBusy("Delete Task", onDeleteTask));
  getEl("refreshTasksBtn").addEventListener("click", () => runBusy("Refresh Tasks", loadTasks));
  getEl("startTaskBtn").addEventListener("click", () => runBusy("Start Task", onStartTask));
  getEl("stopTaskBtn").addEventListener("click", () => runBusy("Stop Task", onStopTask));
  getEl("closeTaskBtn").addEventListener("click", () => runBusy("Close Task", onCloseTask));

  getEl("createTimeRecordBtn").addEventListener("click", () =>
    runBusy("Create Time Record", onCreateTimeRecord),
  );
  getEl("updateTimeRecordBtn").addEventListener("click", () =>
    runBusy("Update Time Record", onUpdateTimeRecord),
  );
  getEl("deleteTimeRecordBtn").addEventListener("click", () =>
    runBusy("Delete Time Record", onDeleteTimeRecord),
  );

  getEl("generalReportBtn").addEventListener("click", () => runBusy("General Report", onGeneralReport));
  getEl("projectReportBtn").addEventListener("click", () => runBusy("Project Report", onProjectReport));
  getEl("taskReportBtn").addEventListener("click", () => runBusy("Task Report", onTaskReport));
}

function bindListHandlers() {
  getEl("projectList").addEventListener("click", (event) => {
    const button = event.target.closest("button[data-project-id]");
    if (!button) {
      return;
    }
    setSelectedProject(button.dataset.projectId || "");
    runBusy("Load Tasks", loadTasks);
  });

  getEl("taskTableBody").addEventListener("click", (event) => {
    const button = event.target.closest("button[data-task-action]");
    if (!button) {
      return;
    }
    const taskId = button.dataset.taskId || "";
    setSelectedTask(taskId);

    const action = button.dataset.taskAction;
    if (action === "select") {
      setStatus("Task selected.");
      return;
    }

    if (action === "start") {
      runBusy("Start Task", onStartTask);
      return;
    }
    if (action === "stop") {
      runBusy("Stop Task", onStopTask);
      return;
    }
    if (action === "close") {
      runBusy("Close Task", onCloseTask);
    }
  });
}

function bindPersistence() {
  for (const id of PERSISTED_INPUT_IDS) {
    const element = getEl(id);
    if (!element) {
      continue;
    }
    element.addEventListener("input", saveStateFromInputs);
  }

  getEl("taskListIsActive").addEventListener("change", () => {
    saveStateFromInputs();
    runBusy("Refresh Tasks", loadTasks);
  });
}

async function init() {
  writeState(loadState());
  bindButtonHandlers();
  bindListHandlers();
  bindPersistence();
  renderAuthBadge();
  renderProjects();
  renderTasks();
  renderReport();
  renderTrace();
  renderSelectionBadge();

  const state = readState();
  if (!state.accessToken) {
    setStatus("Ready. Login to load workspace data.");
    return;
  }

  setBusy(true);
  try {
    await syncWorkspace();
    setStatus("Workspace restored from saved token.");
  } catch (error) {
    clearAuthLocal();
    setStatus(error instanceof Error ? error.message : String(error), true);
  } finally {
    setBusy(false);
  }
}

init();
