const STORAGE_KEY = "gotimekeeper.demoClient.state";

const DEFAULT_STATE = {
  baseUrl: "http://localhost:8080",
  timezone: "Europe/Prague",
  email: "demo.user@example.com",
  password: "Demo@12345",
  newPassword: "Demo@12345New",
  accessToken: "",
  refreshToken: "",
  userId: "",
  projectId: "",
  projectName: "Demo Project",
  taskId: "",
  taskName: "Demo Task",
  timeRecordId: "",
  workDate: "2026-03-31T00:00:00Z",
  startTime: "2026-03-31T09:00:00+01:00",
  endTime: "2026-03-31T10:00:00+01:00",
  reportFromDate: "2026-03-01T00:00:00Z",
  reportToDate: "2026-03-31T23:59:59Z",
  projectIdsCsv: "",
  taskIdsCsv: "",
  taskListLimit: "20",
  taskListOffset: "0",
  taskListIsActive: "",
};

const INPUT_IDS = Object.keys(DEFAULT_STATE);

function parseJSON(text) {
  try {
    return JSON.parse(text);
  } catch {
    return null;
  }
}

function getEl(id) {
  return document.getElementById(id);
}

function loadState() {
  const raw = localStorage.getItem(STORAGE_KEY);
  const saved = raw ? parseJSON(raw) : null;
  return { ...DEFAULT_STATE, ...(saved || {}) };
}

function writeStateToInputs(state) {
  for (const id of INPUT_IDS) {
    const el = getEl(id);
    if (!el) {
      continue;
    }
    el.value = state[id] ?? "";
  }
}

function readStateFromInputs() {
  const state = {};
  for (const id of INPUT_IDS) {
    const el = getEl(id);
    state[id] = el ? String(el.value || "").trim() : "";
  }
  return state;
}

function persistState(state) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
}

function resetState() {
  const state = { ...DEFAULT_STATE };
  writeStateToInputs(state);
  persistState(state);
  setStatus("Demo values reset.");
}

function csvToArray(value) {
  if (!value) {
    return null;
  }
  const parts = value
    .split(",")
    .map((v) => v.trim())
    .filter((v) => v.length > 0);
  return parts.length > 0 ? parts : null;
}

function setStatus(message, isError = false) {
  const el = getEl("statusLine");
  el.textContent = message;
  el.style.color = isError ? "#b91c1c" : "#065f46";
}

function setRequestLog(payload) {
  getEl("lastRequest").textContent = JSON.stringify(payload, null, 2);
}

function setResponseLog(payload) {
  getEl("lastResponse").textContent = JSON.stringify(payload, null, 2);
}

function endpointUrl(baseUrl, path) {
  return `${baseUrl.replace(/\/+$/, "")}${path}`;
}

async function requestAPI({
  label,
  method,
  path,
  body,
  authRequired = true,
  headers = {},
}) {
  const state = readStateFromInputs();
  const url = endpointUrl(state.baseUrl, path);
  const requestHeaders = { ...headers };

  if (body !== undefined) {
    requestHeaders["Content-Type"] = "application/json";
  }

  if (authRequired && state.accessToken) {
    requestHeaders["Authorization"] = `Bearer ${state.accessToken}`;
  }

  if (authRequired && !state.accessToken) {
    throw new Error("Access token is empty. Login or register first.");
  }

  setRequestLog({
    label,
    method,
    url,
    headers: requestHeaders,
    body: body ?? null,
  });

  const response = await fetch(url, {
    method,
    headers: requestHeaders,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  const rawText = await response.text();
  const payload = parseJSON(rawText) || { raw: rawText };
  setResponseLog({
    label,
    status: response.status,
    ok: response.ok,
    payload,
  });

  if (!response.ok) {
    const msg = payload?.message || `HTTP ${response.status}`;
    throw new Error(`${label} failed: ${msg}`);
  }
  return payload;
}

function saveAndRender(state) {
  persistState(state);
  writeStateToInputs(state);
}

function updateStateFromResponse(action, payload) {
  const state = readStateFromInputs();
  const data = payload?.data;

  if (["register", "login", "refresh"].includes(action) && data) {
    state.accessToken = data.accessToken || state.accessToken;
    state.refreshToken = data.refreshToken || state.refreshToken;
    state.userId = data.user?.id || state.userId;
  }

  if (action === "me" && data) {
    state.userId = data.id || state.userId;
    state.email = data.email || state.email;
  }

  if (["logout", "deleteMe"].includes(action)) {
    state.accessToken = "";
    state.refreshToken = "";
    state.userId = "";
  }

  if (["createProject", "getProject", "updateProject"].includes(action) && data) {
    state.projectId = data.id || state.projectId;
    state.projectName = data.name || state.projectName;
  }

  if (action === "listProjects" && Array.isArray(data) && data.length > 0) {
    state.projectId = data[0].id || state.projectId;
    state.projectName = data[0].name || state.projectName;
  }

  if (["createTask", "getTask", "updateTask"].includes(action) && data) {
    state.taskId = data.id || state.taskId;
    state.taskName = data.name || state.taskName;
    state.projectId = data.projectId || state.projectId;
  }

  if (action === "listProjectTasks" && data?.tasks?.length > 0) {
    state.taskId = data.tasks[0].id || state.taskId;
    state.taskName = data.tasks[0].name || state.taskName;
  }

  if (["createTimeRecord", "updateTimeRecord"].includes(action) && data) {
    state.timeRecordId = data.id || state.timeRecordId;
    state.projectId = data.projectId || state.projectId;
    state.taskId = data.taskId || state.taskId;
  }

  if (action === "deleteTimeRecord") {
    state.timeRecordId = "";
  }

  saveAndRender(state);
}

const ACTIONS = {
  ping: () =>
    requestAPI({
      label: "Ping",
      method: "GET",
      path: "/ping",
      authRequired: false,
    }),

  register: () => {
    const s = readStateFromInputs();
    return requestAPI({
      label: "Register",
      method: "POST",
      path: "/api/auth/register",
      authRequired: false,
      body: { email: s.email, password: s.password },
    });
  },

  login: () => {
    const s = readStateFromInputs();
    return requestAPI({
      label: "Login",
      method: "POST",
      path: "/api/auth/login",
      authRequired: false,
      body: { email: s.email, password: s.password },
    });
  },

  refresh: () => {
    const s = readStateFromInputs();
    return requestAPI({
      label: "Refresh",
      method: "POST",
      path: "/api/auth/refresh",
      authRequired: false,
      body: { refreshToken: s.refreshToken },
    });
  },

  logout: () => {
    const s = readStateFromInputs();
    return requestAPI({
      label: "Logout",
      method: "POST",
      path: "/api/auth/logout",
      body: { refreshToken: s.refreshToken },
    });
  },

  me: () =>
    requestAPI({
      label: "Get Me",
      method: "GET",
      path: "/api/user/me",
    }),

  changePassword: () => {
    const s = readStateFromInputs();
    return requestAPI({
      label: "Change Password",
      method: "POST",
      path: "/api/auth/change-password",
      body: {
        currentPassword: s.password,
        newPassword: s.newPassword,
      },
    });
  },

  deleteMe: () =>
    requestAPI({
      label: "Delete Account",
      method: "DELETE",
      path: "/api/user/me",
    }),

  createProject: () => {
    const s = readStateFromInputs();
    return requestAPI({
      label: "Create Project",
      method: "POST",
      path: "/api/project",
      body: { name: s.projectName },
    });
  },

  getProject: () => {
    const s = readStateFromInputs();
    return requestAPI({
      label: "Get Project",
      method: "GET",
      path: `/api/project/${s.projectId}`,
    });
  },

  listProjects: () =>
    requestAPI({
      label: "List Projects",
      method: "GET",
      path: "/api/project/list",
    }),

  updateProject: () => {
    const s = readStateFromInputs();
    return requestAPI({
      label: "Update Project",
      method: "PATCH",
      path: "/api/project",
      body: { id: s.projectId, name: s.projectName },
    });
  },

  deleteProject: () => {
    const s = readStateFromInputs();
    return requestAPI({
      label: "Delete Project",
      method: "DELETE",
      path: `/api/project/${s.projectId}`,
    });
  },

  createTask: () => {
    const s = readStateFromInputs();
    return requestAPI({
      label: "Create Task",
      method: "POST",
      path: "/api/task",
      body: { name: s.taskName, projectId: s.projectId },
    });
  },

  getTask: () => {
    const s = readStateFromInputs();
    return requestAPI({
      label: "Get Task",
      method: "GET",
      path: `/api/task/${s.taskId}`,
    });
  },

  listProjectTasks: () => {
    const s = readStateFromInputs();
    const qs = new URLSearchParams();
    if (s.taskListLimit) {
      qs.set("limit", s.taskListLimit);
    }
    if (s.taskListOffset) {
      qs.set("offset", s.taskListOffset);
    }
    if (s.taskListIsActive !== "") {
      qs.set("isActive", s.taskListIsActive);
    }

    const suffix = qs.toString() ? `?${qs.toString()}` : "";
    return requestAPI({
      label: "List Project Tasks",
      method: "GET",
      path: `/api/task/list/project/${s.projectId}${suffix}`,
    });
  },

  updateTask: () => {
    const s = readStateFromInputs();
    return requestAPI({
      label: "Update Task",
      method: "PATCH",
      path: "/api/task",
      body: {
        id: s.taskId,
        name: s.taskName,
        projectId: s.projectId,
      },
    });
  },

  deleteTask: () => {
    const s = readStateFromInputs();
    return requestAPI({
      label: "Delete Task",
      method: "DELETE",
      path: `/api/task/${s.taskId}`,
    });
  },

  startTask: () => {
    const s = readStateFromInputs();
    return requestAPI({
      label: "Start Task",
      method: "PATCH",
      path: `/api/task/${s.taskId}/start`,
      headers: { "X-Timezone": s.timezone },
    });
  },

  stopTask: () => {
    const s = readStateFromInputs();
    return requestAPI({
      label: "Stop Task",
      method: "PATCH",
      path: `/api/task/${s.taskId}/stop`,
    });
  },

  closeTask: () => {
    const s = readStateFromInputs();
    return requestAPI({
      label: "Close Task",
      method: "PATCH",
      path: `/api/task/${s.taskId}/close`,
    });
  },

  createTimeRecord: () => {
    const s = readStateFromInputs();
    return requestAPI({
      label: "Create Time Record",
      method: "POST",
      path: "/api/task/session",
      body: {
        projectId: s.projectId,
        taskId: s.taskId,
        workDate: s.workDate,
        workTimezone: s.timezone,
        startTime: s.startTime,
        endTime: s.endTime,
      },
    });
  },

  updateTimeRecord: () => {
    const s = readStateFromInputs();
    return requestAPI({
      label: "Update Time Record",
      method: "PATCH",
      path: "/api/task/session",
      body: {
        id: s.timeRecordId,
        projectId: s.projectId,
        taskId: s.taskId,
        workDate: s.workDate,
        workTimezone: s.timezone,
        startTime: s.startTime,
        endTime: s.endTime,
      },
    });
  },

  deleteTimeRecord: () => {
    const s = readStateFromInputs();
    return requestAPI({
      label: "Delete Time Record",
      method: "DELETE",
      path: `/api/task/session/${s.timeRecordId}`,
    });
  },

  generalReport: () => {
    const s = readStateFromInputs();
    const projects = csvToArray(s.projectIdsCsv);
    const body = {
      timeRange: {
        fromDate: s.reportFromDate,
        toDate: s.reportToDate,
      },
    };
    if (projects) {
      body.projects = projects;
    }
    return requestAPI({
      label: "General Report",
      method: "POST",
      path: "/api/report/general",
      body,
    });
  },

  projectReport: () => {
    const s = readStateFromInputs();
    const tasks = csvToArray(s.taskIdsCsv);
    const body = {
      projectId: s.projectId,
      timeRange: {
        fromDate: s.reportFromDate,
        toDate: s.reportToDate,
      },
    };
    if (tasks) {
      body.tasks = tasks;
    }
    return requestAPI({
      label: "Project Report",
      method: "POST",
      path: "/api/report/project",
      body,
    });
  },

  taskReport: () => {
    const s = readStateFromInputs();
    return requestAPI({
      label: "Task Report",
      method: "POST",
      path: "/api/report/task",
      body: {
        taskId: s.taskId,
        timeRange: {
          fromDate: s.reportFromDate,
          toDate: s.reportToDate,
        },
      },
    });
  },
};

async function runAction(action) {
  const fn = ACTIONS[action];
  if (!fn) {
    setStatus(`Unknown action: ${action}`, true);
    return;
  }

  try {
    persistState(readStateFromInputs());
    const payload = await fn();
    updateStateFromResponse(action, payload);
    setStatus(`${action} succeeded.`);
  } catch (err) {
    setStatus(err instanceof Error ? err.message : String(err), true);
  }
}

function init() {
  writeStateToInputs(loadState());

  getEl("saveStateBtn").addEventListener("click", () => {
    persistState(readStateFromInputs());
    setStatus("State saved.");
  });

  getEl("resetStateBtn").addEventListener("click", () => {
    resetState();
  });

  for (const button of document.querySelectorAll("[data-action]")) {
    button.addEventListener("click", async () => {
      await runAction(button.dataset.action);
    });
  }

  setRequestLog({ info: "No requests yet." });
  setResponseLog({ info: "No responses yet." });
}

init();
