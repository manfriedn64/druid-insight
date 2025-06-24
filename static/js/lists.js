// js/lists.js

let currentSchema = null;
let selectedDatasource = null;

async function fetchSchema() {
  const token = localStorage.getItem("jwt");
  const resp = await apiFetch("/api/schema", {
    headers: { "Authorization": "Bearer " + token }
  });
  if (!resp.ok) throw new Error("Failed to load schema");
  return await resp.json();
}

async function initDashboardSchema() {
  try {
    currentSchema = await fetchSchema();
    // Remplir le select avec les datasources
    const dsSelect = document.getElementById("datasource-select");
    dsSelect.addEventListener("change", e => {
      selectedDatasource = dsSelect.value;
      // Reset sélection dimension/métrique si tu veux
      selectedDimensions = [];
      selectedMetrics = [];
      updateDimensionMetricLists();
      // (optionnel) Efface graphiques, etc.
    });
    dsSelect.innerHTML = "";
    Object.keys(currentSchema).forEach(ds => {
      const opt = document.createElement("option");
      opt.value = ds;
      opt.textContent = ds;
      dsSelect.appendChild(opt);
    });
    // Par défaut, sélectionne la première
    selectedDatasource = dsSelect.value;
    dsSelect.addEventListener("change", e => {
      selectedDatasource = dsSelect.value;
  selectedDimensions = [];
  selectedMetrics = [];
  renderLists();
      // (optionnel) reset sélection, graph, etc.
    });
    // Affiche dimensions/métriques pour la première datasource
    renderLists();
  } catch (err) {
    alert("Failed to load schema: " + err);
  }
}

function updateDimensionMetricLists() {
  if (!currentSchema || !selectedDatasource) return;
  const dims = currentSchema[selectedDatasource].dimensions;
  const mets = currentSchema[selectedDatasource].metrics;
  mets.forEach(metObj => {
    const name = metObj.name;
    const type = metObj.type; // "bar" ou "line"
  });
  updateUIWithDimensions(dims);
  updateUIWithMetrics(mets);
}

function updateUIWithDimensions(dimList) {
  const container = document.getElementById("dimensions-list");
  container.innerHTML = "";
  dimList.forEach(dim => {
    // Fabrique un bouton, une checkbox, etc. selon ta logique UI
    const el = document.createElement("button");
    el.textContent = dim;
    el.className = "dim-btn";
    el.onclick = () => handleSelectDimension(dim);
    container.appendChild(el);
  });
}

function updateUIWithMetrics(metList) {
  const container = document.getElementById("metrics-list");
  container.innerHTML = "";
  metList.forEach(met => {
    const el = document.createElement("button");
    el.textContent = met.name;
    el.className = "met-btn";
    el.onclick = () => handleSelectMetric(met.name);
    container.appendChild(el);
  });
}



function renderLists() {
  if (!currentSchema || !selectedDatasource) return;
  const dlist = document.getElementById('dimensions-list');
  dlist.innerHTML = '';
  let dims = currentSchema[selectedDatasource].dimensions;
  if (!dims.includes("time")) dims = ["time", ...dims];
  dims.forEach(dim => {
    const div = document.createElement('div');
    div.className = 'item' + (selectedDimensions.includes(dim) ? ' selected' : '');
    div.innerHTML = `${dim}`;
    div.onclick = function(e) {
      if (e.target.classList.contains('filter-btn')) return;
      if (selectedDimensions.includes(dim)) {
        selectedDimensions = selectedDimensions.filter(x=>x!==dim);
      } else selectedDimensions.push(dim);
      renderLists();
    };
    const filterBtn = document.createElement('button');
    filterBtn.className = 'filter-btn';
    filterBtn.title = 'Filtrer';
    filterBtn.innerHTML = `
<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2a1 1 0 01-.293.707L15 13.414V19a1 1 0 01-1.447.894l-4-2A1 1 0 019 17v-3.586L3.293 6.707A1 1 0 013 6V4z"/>
</svg>`;

    filterBtn.onclick = function(ev){
      ev.stopPropagation();
      openFilterPopup(dim);
    };
    if (filters[dim]) filterBtn.setAttribute('active','');
    div.appendChild(filterBtn);
    dlist.appendChild(div);
  });
  let timeContainer = document.getElementById("time-grouping-container");
  if (selectedDimensions.includes("time")) {
    if (!timeContainer) {
      timeContainer = document.createElement("div");
      timeContainer.id = "time-grouping-container";
      // Place juste après la liste des dimensions
      dlist.parentNode.insertBefore(timeContainer, dlist.nextSibling);
    }
    timeContainer.innerHTML = `
      <label for="time-group-select" style="margin-top:1em;font-weight:bold;">
        Grouper la dimension temporelle par&nbsp;
        <select id="time-group-select">
          <option value="hour">Heure</option>
          <option value="day">Jour</option>
          <option value="week">Semaine</option>
          <option value="month">Mois</option>
        </select>
      </label>
    `;
    document.getElementById("time-group-select").value = window.timeGrouping || "day";
    document.getElementById("time-group-select").onchange = function() {
      window.timeGrouping = this.value;
    };
  } else {
    if (timeContainer) timeContainer.remove();
  }
  const mlist = document.getElementById('metrics-list');
  mlist.innerHTML = '';
  const mets = currentSchema[selectedDatasource].metrics;
  mets.forEach(metric => {
    const div = document.createElement('div');
    div.className = 'item' + (selectedMetrics.includes(metric.name) ? ' selected' : '');
    div.textContent = metric.name;
    div.onclick = function(){
      if (selectedMetrics.includes(metric.name)) selectedMetrics = selectedMetrics.filter(x=>x!==metric.name);
      else selectedMetrics.push(metric.name);
      renderLists();
    };
    mlist.appendChild(div);
  });
}

function parseToISODate(s) {
  if (/^\d{4}-\d{2}-\d{2}$/.test(s)) return s;
  let d = new Date(s);
  if (!isNaN(d)) {
    let m = (d.getMonth()+1).toString().padStart(2,'0');
    let j = d.getDate().toString().padStart(2,'0');
    return `${d.getFullYear()}-${m}-${j}`;
  }
  return "";
}

window.addEventListener('DOMContentLoaded', function() {
  const params = new URLSearchParams(window.location.search);
  if (!params.has('ds')) return;

  selectedDatasource = params.get('ds');
  if (params.has('dim')) {
    selectedDimensions = params.get('dim').split(',').map(x => decodeURIComponent(x));
  }
  if (params.has('met')) {
    selectedMetrics = params.get('met').split(',').map(x => decodeURIComponent(x));
  }
  if (params.has('date1')) {
    let v = parseToISODate(params.get('date1'));
    let el = document.getElementById('start-date');
    if (el) {
      console.log(el.value);
      el.value = v;
      console.log(el.value);
    }
  }
  if (params.has('date2')) {
    let v = parseToISODate(params.get('date2'));
    let el = document.getElementById('end-date');
    if (el) el.value = v;
  }
  if (params.has('compare')) {
    document.getElementById('compare-period').value = params.get('compare');
  }
  filters = {};
  if (params.has('filters')) {
    params.get('filters').split(';').forEach(f => {
      if (!f) return;
      let [dim, vals] = f.split(':');
      if (dim && vals) filters[decodeURIComponent(dim)] = vals.split(',').map(v => decodeURIComponent(v));
    });
  }
  renderLists();
});

