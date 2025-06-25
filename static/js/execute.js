// js/execute.js
function getComparisonRange() {
  let start = document.getElementById('start-date').value;
  let end = document.getElementById('end-date').value;
  let compare = document.getElementById('compare-period').value;
  if (!start || !end || !compare) return null;
  let d0 = new Date(start), d1 = new Date(end);
  let nbJours = Math.floor((d1-d0)/(1000*3600*24)) + 1;
  let offset = 0;
  if (compare === "prev_day") offset = -1;
  if (compare === "prev_week") offset = -7;
  if (compare === "prev_month") {
    let ds = new Date(d0), de = new Date(d1);
    ds.setMonth(ds.getMonth() - 1);
    de.setMonth(de.getMonth() - 1);
    return [
      ds.toISOString().slice(0,10),
      de.toISOString().slice(0,10)
    ];
  }
  let ds = new Date(d0), de = new Date(d1);
  ds.setDate(ds.getDate() + offset);
  de.setDate(de.getDate() + offset);
  return [
    ds.toISOString().slice(0,10),
    de.toISOString().slice(0,10)
  ];
}
function cloneDeep(obj) {
  return JSON.parse(JSON.stringify(obj));
}
document.getElementById('execute-btn').onclick = async function() {
  if (!selectedDimensions.length) return alert('At least one dimension is required');
  if (!selectedMetrics.length) return alert('At least one metric is required');
  let start = document.getElementById('start-date').value;
  let end = document.getElementById('end-date').value;
  if (!start || !end) return alert('Start date and end date are required');

  if (window.reportInProgress) return alert("Please wait for previous report to be completed");
  window.reportInProgress = true;
  document.getElementById('execute-btn').disabled = true;
  document.getElementById('loading').style.display = '';
  let compareValue = document.getElementById('compare-period').value;
  let chartType = "auto";

  // Ajout : récupère le groupement temporel si la dimension "time" est sélectionnée
  let groupByTime = selectedDimensions.includes("time") ? (window.timeGrouping || "day") : null;

  let payload = {
    datasource: selectedDatasource,
    dimensions: [...selectedDimensions],
    metrics: [...selectedMetrics],
    filters: Object.keys(filters).map(dim => ({
      dimension: dim,
      values: filters[dim]
    })),
    dates: [start, end],
    compare: compareValue || "",
    chartType
  };
  if (groupByTime) payload["time_group"] = groupByTime;

  try {
    // 1. Lancer la requête d'exécution
    const res = await apiFetch('/api/reports/execute', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    if (!res.ok) throw new Error("Error running report");
    const { id } = await res.json();

    // 2. Polling de l'état du rapport
    let status = "processing";
    let pollResult = null;
    while (status === "processing" || status === "waiting") {
      await new Promise(res => setTimeout(res, 1200));
      const sres = await apiFetch('/api/reports/status?id=' + encodeURIComponent(id));
      if (!sres.ok) throw new Error("Erreur polling");
      pollResult = await sres.json();
      status = pollResult.status;
    }

    if (status !== "complete") throw new Error(pollResult.errorMsg || "Report error");

    // 3. Télécharger le CSV (récupérer le lien)
    let url = `/api/reports/download?id=${encodeURIComponent(id)}`;
    let lines = (pollResult.result && pollResult.result.length) || 0;
    let bytes = 0; // (tu peux calculer la taille du fichier via HEAD ou lors du download si tu veux)
    let report = {
      payload: cloneDeep(payload),
      lines,
      bytes,
      url,
      chartData: null, // à générer plus tard si besoin
      dt: (new Date()).toLocaleString('fr-FR', {hour:'2-digit',minute:'2-digit',second:'2-digit',year:'numeric',month:'2-digit',day:'2-digit'})
    };
    allResults.unshift(report);
    if (report.payload.dimensions.length === 1) {
      try {
        const res = await apiFetch(url, { headers: { 'Accept': 'text/csv' } });
        if (res.ok) {
          const csvText = await res.text();
          // Parser CSV vanilla
          const lines = csvText.split('\n').filter(l => l.trim());
          if (lines.length < 2) {
            alert("empty result");y
            return
             // pas de données utiles
          }
          const headers = lines[0].split(',');
          const rows = lines.slice(1).map(l => l.split(','));
          // Récupère la dimension et les métriques demandées
          const dim = report.payload.dimensions[0];
          const metrics = report.payload.metrics;
          // Indices des colonnes
          const dimIdx = headers.indexOf(dim);
          const metricsIdx = metrics.map(m => headers.indexOf(m));
          // Prépare labels (abscisses)
          const labels = rows.map(row => row[dimIdx]);
          // Prépare datasets
          let datasets = [];
          metrics.forEach((metric, i) => {
            let serie = rows.map(row => Number(row[metricsIdx[i]]));
            datasets.push({ label: metric, data: serie });
          });
          // chartData attendu par reports.js
          report.chartData = {};
          report.chartData[dim] = datasets.map(ds => ({
            label: ds.label,
            data: ds.data
          }));
          report.chartData[dim].xLabels = labels;
        }
      } catch (e) {
        console.warn("Can not parse CSV file for chart", e);
      }
    }

    renderResultsList();

  } catch (e) {
    alert(e.message || e);
  } finally {
    window.reportInProgress = false;
    document.getElementById('execute-btn').disabled = false;
    document.getElementById('loading').style.display = 'none';
  }
};

