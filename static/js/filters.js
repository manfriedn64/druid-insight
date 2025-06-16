let currentValues = [];

const MAX_DISPLAY = 500;

function renderPopupValues() {
  let vals = currentValues || [];
  let search = (document.getElementById('popup-search').value || '').toLowerCase();
  const pl = document.getElementById('popup-values-list');
  pl.innerHTML = '';
  let shown = 0;

  // Filtrer la liste en amont
  let filtered = [];
  vals.forEach(val => {
    if (!search || val.toLowerCase().includes(search)) filtered.push(val);
  });

  if (filtered.length > MAX_DISPLAY) {
    filtered = filtered.slice(0, MAX_DISPLAY);
    pl.innerHTML = `<div style="color:#d00;font-weight:bold;">Trop de résultats, affinez votre recherche...</div>`;
  }

  filtered.forEach(val => {
    shown++;
    const id = `check_${currentFilterDim}_${val}`;
    pl.innerHTML += `<label><input type="checkbox" id="${id}" value="${val}" ${tempSelectedValues.includes(val) ? 'checked' : ''}> ${val}</label>`;
  });
  if (!shown) pl.innerHTML += `<span style="color:#888;">Aucune valeur</span>`;
  pl.querySelectorAll('input[type="checkbox"]').forEach(input => {
    input.onchange = function () {
      if (this.checked) tempSelectedValues.push(this.value);
      else tempSelectedValues = tempSelectedValues.filter(x => x !== this.value);
    };
  });
}


async function openFilterPopup(dim) {
  currentFilterDim = dim;
  tempSelectedValues = filters[dim] ? [...filters[dim]] : [];
  document.getElementById('filter-dimension').textContent = dim;
  document.getElementById('popup-search').value = '';
  currentValues = []; // Vide tant que non chargé

  // Ouvre la popup IMMÉDIATEMENT avec indicateur de chargement
  document.getElementById('popup-values-list').innerHTML = '<i>Chargement...</i>';
  document.getElementById('filter-popup').style.display = '';
  setTimeout(() => document.getElementById('popup-search').focus(), 120);

  // Ensuite, fetch en arrière-plan et remplit la popup quand prêt
  try {
    const res = await apiFetch('/api/filters/values', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        datasource: selectedDatasource,
        dimension: dim
      })
    });

    if (!res.ok) throw new Error("Réponse non valide");
    const json = await res.json();
    currentValues = json.values || [];
  } catch (err) {
    console.error("Erreur API filtre :", err);
    currentValues = [];
    document.getElementById('popup-values-list').innerHTML = '<span style="color:#e33;">Erreur de chargement</span>';
    return;
  }

  // Met à jour le contenu avec les valeurs
  renderPopupValues();
}


// Handlers (à placer en dehors de toute fonction)
document.getElementById('popup-search').oninput = renderPopupValues;

document.getElementById('popup-ok').onclick = function() {
  if (tempSelectedValues.length) filters[currentFilterDim] = [...tempSelectedValues];
  else delete filters[currentFilterDim];
  document.getElementById('filter-popup').style.display = 'none';
  renderLists();
};

document.getElementById('popup-cancel').onclick = function() {
  document.getElementById('filter-popup').style.display = 'none';
};

window.addEventListener('keydown', function(e){
  if (document.getElementById('filter-popup').style.display !== 'none') {
    if (e.key === "Escape") {
      document.getElementById('filter-popup').style.display = 'none';
    }
  }
});
