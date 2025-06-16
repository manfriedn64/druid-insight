let sessionCountdownTimer = null;
let sessionExpired = false;

function forceLogoutSessionExpired() {
  localStorage.removeItem("jwt");
  sessionExpired = true;
  if (sessionCountdownTimer) clearInterval(sessionCountdownTimer);
  showLogin();
}

function showLogin() {
  document.getElementById("login-section").style.display = "";
  document.getElementById("dashboard-section").style.display = "none";
  const errorDiv = document.getElementById("login-error");
  if (sessionExpired) {
    errorDiv.textContent = "Your session has expired, please log in again.";
    errorDiv.style.display = "block";
    sessionExpired = false;
  } else {
    errorDiv.textContent = "";
    errorDiv.style.display = "none";
  }
}

async function showDashboard() {
  document.getElementById("login-section").style.display = "none";
  document.getElementById("dashboard-section").style.display = "";
  const username = getUsernameFromJWT();
  document.getElementById("welcome-user").textContent = username ? `Welcome, ${username}` : "";
  await initDashboardSchema();
  if (typeof loadDashboard === "function") loadDashboard();
  startSessionCountdown();
}

// Au chargement : check JWT
window.addEventListener("DOMContentLoaded", () => {
  const token = localStorage.getItem("jwt");
  if (!token || isJWTExpired(token)) {
    localStorage.removeItem("jwt");
    showLogin();
  } else {
    showDashboard();
  }
});

// Logique login (même chose qu’avant, on adapte la fin)
document.getElementById("login-form").addEventListener("submit", async function(e) {
  e.preventDefault();
  const username = document.getElementById("login-username").value;
  const password = document.getElementById("login-password").value;
  const errorDiv = document.getElementById("login-error");
  errorDiv.style.display = "none";
  errorDiv.textContent = "";

  try {
    const resp = await fetch("/api/login", {
      method: "POST",
      headers: {"Content-Type": "application/json"},
      body: JSON.stringify({username, password}),
    });
    if (!resp.ok) {
      const msg = await resp.text();
      errorDiv.textContent = "Login failed: " + msg;
      errorDiv.style.display = "block";
      return;
    }
    const data = await resp.json();
    if (data.token) {
      localStorage.setItem("jwt", data.token);
      showDashboard();
    } else {
      errorDiv.textContent = "No token received";
      errorDiv.style.display = "block";
    }
  } catch (err) {
    errorDiv.textContent = "Network error: " + err;
    errorDiv.style.display = "block";
  }
});

function loadDashboard() {
  // Ici, tu charges le reporting/dashboard (filtres, graphiques, requêtes API...)
  // Exemple : fetch des rapports, génération des graphs...
  //document.getElementById("dashboard-content").textContent = "Welcome! Reporting will be loaded here.";
// js/login.js
  let today = new Date();
  let dd = String(today.getDate()).padStart(2, '0');
  let mm = String(today.getMonth()+1).padStart(2, '0');
  let yyyy = today.getFullYear();
  let dateStr = yyyy + '-' + mm + '-' + dd;
  if (document.getElementById('start-date').value == "") {
    document.getElementById('start-date').value = dateStr;
  }
  if (document.getElementById('end-date').value == "") {
    document.getElementById('end-date').value = dateStr;
  }
  updateDimensionMetricLists();
  renderLists();
}

// Log out
document.getElementById("logout-btn").addEventListener("click", () => {
  localStorage.removeItem("jwt");
  location.reload();
});

function getUsernameFromJWT() {
  const token = localStorage.getItem("jwt");
  if (!token) return "";
  try {
    // JWT format : header.payload.signature
    const payload = token.split('.')[1];
    // Base64 decode, handle missing padding
    const pad = payload.length % 4 === 0 ? '' : '='.repeat(4 - payload.length % 4);
    const decoded = atob(payload.replace(/-/g, '+').replace(/_/g, '/') + pad);
    const obj = JSON.parse(decoded);
    return obj.sub || "";
  } catch(e) {
    return "";
  }
}

function isJWTExpired(token) {
  if (!token) return true;
  try {
    const payload = token.split('.')[1];
    const pad = payload.length % 4 === 0 ? '' : '='.repeat(4 - payload.length % 4);
    const decoded = atob(payload.replace(/-/g, '+').replace(/_/g, '/') + pad);
    const obj = JSON.parse(decoded);
    if (!obj.exp) return false;
    const now = Math.floor(Date.now() / 1000);
    return obj.exp < now;
  } catch(e) {
    return true;
  }
}

async function apiFetch(url, options = {}) {
  const token = localStorage.getItem("jwt");
  if (!token || isJWTExpired(token)) {
    forceLogoutSessionExpired();
    return;
  }
  options.headers = options.headers || {};
  options.headers.Authorization = "Bearer " + token;
  const resp = await fetch(url, options);
  if (resp.status === 401) {
    localStorage.removeItem("jwt");
    showLogin();
    throw new Error("Session expired");
  }
  return resp;
}

function startSessionCountdown() {
  if (sessionCountdownTimer) clearInterval(sessionCountdownTimer);

  const token = localStorage.getItem("jwt");
  if (!token) return;
  let exp = null;
  try {
    const payload = token.split('.')[1];
    const pad = payload.length % 4 === 0 ? '' : '='.repeat(4 - payload.length % 4);
    const decoded = atob(payload.replace(/-/g, '+').replace(/_/g, '/') + pad);
    const obj = JSON.parse(decoded);
    if (!obj.exp) return;
    exp = obj.exp;
  } catch(e) { return; }

  function updateTimer() {
    const now = Math.floor(Date.now() / 1000);
    const left = exp - now;
    const timerDiv = document.getElementById("session-timer");
    if (left <= 0) {
      timerDiv.textContent = "Session expired.";
      forceLogoutSessionExpired();
      clearInterval(sessionCountdownTimer);
      return;
    }
    const min = Math.floor(left / 60);
    const sec = left % 60;
    timerDiv.textContent = "Session expires in " + min + ":" + (sec < 10 ? "0" : "") + sec;
  }

  updateTimer(); // Appelle une première fois tout de suite
  sessionCountdownTimer = setInterval(updateTimer, 1000);
}
