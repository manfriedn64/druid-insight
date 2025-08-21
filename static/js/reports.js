function escapeHTML(str) {
  return str.replace(/[&<>"']/g, function(m) {
    return ({
      '&':'&amp;',
      '<':'&lt;',
      '>':'&gt;',
      '"':'&quot;',
      "'":'&#39;'
    })[m];
  });
}

function renderResultsList() {
  let rlist = document.getElementById('results-list');
  rlist.innerHTML = '';
  allResults.forEach((r, idx) => {
    let block = document.createElement('div');
    block.className = 'report-block';
    block.innerHTML = `
      <div style="font-size:1.04em;"><b>Report from ${r.dt}</b></div>
      <div>
        <button class="download-csv-btn download-btn" data-id="${r.url}">Download CSV file</button>
        <button class="download-xlsx-btn download-btn" data-id="${r.url}">Download Excel file</button>
        <button class="share-btn" data-idx="${idx}">Share link</button>
      </div>
      <div>Taille du fichier : <b>${(r.bytes/1024).toFixed(1)} Ko</b></div>
      <div>Nombre de lignes de résultats : <b>${r.lines}</b></div>
      <button class="modify-btn" data-idx="${idx}">Modify</button>
      <button class="show-api-btn" style="margin:0.5em 0 0.3em 0;">Show API request</button>
      <div class="api-json" style="display:none">
        <div class="api-call-url" style="color:#888;font-size:0.95em;">
          <b>API:</b> <span style="font-family:monospace;">/api/reports/execute</span>
        </div>
        <pre style="background:#f5f5f5; border-radius:1em; margin:0; padding:1em; font-size:0.9em;">${escapeHTML(JSON.stringify(r.payload, null, 2))}</pre>
      </div>
    `;
    // Si graphique à afficher :
    if (r.chartData) {
      let chartDiv = document.createElement('div');
      chartDiv.className = "report-graphs-container";
      Object.keys(r.chartData).forEach(dim => {
        if (dim === "xLabels") return;
        let chartId = `chart_${idx}_${dim}`;
        let graph = document.createElement('div');
        graph.className = "report-graph";
        graph.innerHTML = `<div class="caption">${dim}</div><canvas id="${chartId}"></canvas>`;
        chartDiv.appendChild(graph);
        let datasets = r.chartData[dim].map((series, i) => ({
          label: series.label,
          data: series.data,
        }));
        let xLabels = r.chartData[dim].xLabels || DUMMY_VALUES[dim] || [];
        let chartType = "auto";
        const allMetricObjs = currentSchema[selectedDatasource].metrics;
        let metrics = r.payload.metrics.map(metName => allMetricObjs.find(m => m.name === metName)).filter(Boolean);
        setTimeout(() => {
          renderMultiMetricChartJS(
            chartId,
            xLabels,
            datasets,
            dim,
            chartType,
            metrics
          );
        }, 0);
      });
      block.appendChild(chartDiv);
    }
    rlist.appendChild(block);
  });
  document.querySelectorAll('.modify-btn').forEach(btn => {
    btn.onclick = function() {
      let report = allResults[parseInt(this.getAttribute('data-idx'))];
      selectedDimensions = [...report.payload.dimensions];
      selectedMetrics = [...report.payload.metrics];
      filters = {};
      (report.payload.filters || []).forEach(f => {
        filters[f.dimension] = [...f.values];
      });
      document.getElementById('start-date').value = report.payload.dates[0] || '';
      document.getElementById('end-date').value = report.payload.dates[1] || '';
      document.getElementById('compare-period').value = '';
      if (report.payload.comparison && report.payload.comparison.length === 2) {
        let c = report.payload.comparison;
        let d0 = new Date(document.getElementById('start-date').value);
        let dc0 = new Date(c[0]);
        let delta = Math.round((d0 - dc0)/86400000);
        if (delta === 1) document.getElementById('compare-period').value = "prev_day";
        else if (delta === 7) document.getElementById('compare-period').value = "prev_week";
        else document.getElementById('compare-period').value = "prev_month";
      }
      renderLists();
      window.scrollTo({top:0, behavior:"smooth"});
    };
  });
  document.querySelectorAll('.download-csv-btn').forEach(btn => {
    btn.onclick = async function() {
      let reportId = this.getAttribute('data-id');
      try {
        const res = await apiFetch(reportId, {
          headers: {
            'Accept': 'text/csv'
          }
        });
        if (!res.ok) throw new Error("Fail to download");
        const blob = await res.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `report_${reportId}.csv`;
        document.body.appendChild(a);
        a.click();
        setTimeout(() => {
          window.URL.revokeObjectURL(url);
          document.body.removeChild(a);
        }, 500);
      } catch (e) {
        alert("Can not download : " + (e.message || e));
      }
    };
  });
  document.querySelectorAll('.download-xlsx-btn').forEach(btn => {
    btn.onclick = async function() {
      let reportId = this.getAttribute('data-id');
      try {
        const res = await apiFetch(reportId + "&type=excel", {
          headers: {
            'Accept': 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
          }
        });
        if (!res.ok) throw new Error("Fail to download");
        const blob = await res.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `report_${reportId}.xlsx`;
        document.body.appendChild(a);
        a.click();
        setTimeout(() => {
          window.URL.revokeObjectURL(url);
          document.body.removeChild(a);
        }, 500);
      } catch (e) {
        alert("Can not download : " + (e.message || e));
      }
    };
  });

  // Gestion du bouton "voir la requête API"
  document.querySelectorAll('.show-api-btn').forEach((btn, i) => {
    btn.onclick = function() {
      const pre = btn.parentNode.querySelector('.api-json');
      if (pre.style.display === "none") {
        pre.style.display = "";
        btn.textContent = "Hide API request";
      } else {
        pre.style.display = "none";
        btn.textContent = "Show API request";
      }
    };
  });

  // Gestion du bouton "Partager"
  document.querySelectorAll('.share-btn').forEach(btn => {
    btn.onclick = function() {
      let idx = parseInt(this.getAttribute('data-idx'));
      let report = allResults[idx];
      const p = report.payload;
      const params = [];
      params.push(`ds=${encodeURIComponent(p.datasource)}`);
      if (p.dimensions && p.dimensions.length)
        params.push(`dim=${encodeURIComponent(p.dimensions.join(','))}`);
      if (p.metrics && p.metrics.length)
        params.push(`met=${encodeURIComponent(p.metrics.join(','))}`);
      if (p.dates && p.dates.length === 2) {
        params.push(`date1=${encodeURIComponent(p.dates[0])}`);
        params.push(`date2=${encodeURIComponent(p.dates[1])}`);
      }
      if (p.filters && p.filters.length) {
        // Format: dimension:valeur1,valeur2;autre:valeur3
        const fStr = p.filters.map(f =>
          `${encodeURIComponent(f.dimension)}:${(f.values || []).map(v => encodeURIComponent(v)).join(',')}`
        ).join(';');
        params.push(`filters=${fStr}`);
      }
      if (p.compare && typeof p.compare === "string" && p.compare) {
        params.push(`compare=${encodeURIComponent(p.compare)}`);
      }
      const baseUrl = window.location.origin + window.location.pathname;
      const shareUrl = `${baseUrl}?${params.join('&')}`;
      navigator.clipboard.writeText(shareUrl).then(() => {
        btn.textContent = "Lien copié !";
        setTimeout(() => { btn.textContent = "Partager"; }, 1300);
      });
    };
  });

}
