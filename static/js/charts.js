// js/charts.js

// Pour éviter des bugs Chart.js "already exists", on stocke les instances :
window.chartInstances = window.chartInstances || {};

function renderMultiMetricChartJS(canvasId, labels, datasets, xlabel, chartType, metrics) {
  const ctx = document.getElementById(canvasId).getContext('2d');
  if (window.chartInstances[canvasId]) {
    window.chartInstances[canvasId].destroy();
  }
  window.chartInstances[canvasId] = new Chart(ctx, {
    type: 'bar', // peu importe, chaque dataset a son propre type
    data: {
      labels: labels,
      datasets: datasets.map((ds, i) => {
        // On récupère le type de la metric directement depuis metrics[i].type (bar/line)
        const thisType = (chartType === 'auto')
          ? (metrics[i].type || 'line')
          : chartType;
        return {
          ...ds,
          type: thisType,
          yAxisID: thisType === 'bar' ? 'y2' : 'y1',
          backgroundColor: thisType === 'bar' ? chartColor(i, 0.5) : 'transparent',
          borderColor: chartColor(i, 1),
          fill: thisType === 'bar',
          borderWidth: 2,
          tension: 0.3
        }
      })
    },
    options: {
      responsive: true,
      maintainAspectRatio: true,
      aspectRatio: 2,
      interaction: { mode: 'index', intersect: false },
      plugins: { legend: { display: true } },
      scales: {
        x: { title: { display: true, text: xlabel } },
        y1: {
          type: 'linear',
          position: 'left',
          title: { display: true, text: "Valeur (line)" },
          beginAtZero: true
        },
        y2: {
          type: 'linear',
          position: 'right',
          title: { display: true, text: "Valeur (bar)" },
          beginAtZero: true,
          grid: { drawOnChartArea: false }
        }
      }
    }
  });
}


function chartColor(i, alpha=1) {
  // Palette sympa, ajoute des couleurs si besoin
  const palette = [
    '#4e79a7', '#f28e2b', '#e15759', '#76b7b2',
    '#59a14f', '#edc948', '#b07aa1', '#ff9da7', '#9c755f', '#bab0ab'
  ];
  let c = palette[i % palette.length];
  // Conversion alpha -> rgba
  if(alpha < 1) {
    let hex = c.replace('#','');
    let r = parseInt(hex.substring(0,2),16);
    let g = parseInt(hex.substring(2,4),16);
    let b = parseInt(hex.substring(4,6),16);
    return `rgba(${r},${g},${b},${alpha})`;
  }
  return c;
}
