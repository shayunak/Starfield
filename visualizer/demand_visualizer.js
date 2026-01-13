const DEMAND_WIDTH_FACTOR = 100;

// 1) Set Cesium Ion Token
Cesium.Ion.defaultAccessToken = '';

// 2) Initialize Viewer
const viewer = new Cesium.Viewer('cesiumContainer', {
  terrain: Cesium.Terrain.fromWorldTerrain(),
  timeline: false, 
  animation: false, 
  geocoder: false, 
  baseLayerPicker: true,
  sceneModePicker: true, 
  navigationHelpButton: false, 
  selectionIndicator: true
});

function getTimeIntervalOfPositionFile(positionFileName) {
  const parts = positionFileName.split('#');

  if (parts.length < 4) return console.log("Position file name does not contain time interval info!");

  const timeInterval = parts[3];
  const timeIntervalClean = timeInterval.replace('ms', ''); // remove "ms"
  return Number(timeIntervalClean);
}

// 3) Load CSV from file
function loadCsvFromFile(file) {
  return new Promise((resolve, reject) => {
    Papa.parse(file, {
      header: true,
      skipEmptyLines: true,
      complete: results => {
        if (results.errors && results.errors.length > 0) {
          console.warn('CSV parsing warnings:', results.errors);
        }
        resolve(results.data);
      },
      error: err => reject(err)
    });
  });
}

// Create arrow function
function drawGeodesicLine(sourcePos, destPos, width = 10) {
  const geoSource = Cesium.Cartographic.fromCartesian(sourcePos);
  const geoDest = Cesium.Cartographic.fromCartesian(destPos);

  const geodesic = new Cesium.EllipsoidGeodesic(geoSource, geoDest);

  // Midpoint: fraction = 0.5
  const mid = geodesic.interpolateUsingFraction(0.5);
  mid.height = geoSource.height;
  const cartMid = Cesium.Ellipsoid.WGS84.cartographicToCartesian(mid);

   return viewer.entities.add({
        polyline: {
            positions: [sourcePos, cartMid, destPos],
            width: width, // controllable width
            material: new Cesium.PolylineArrowMaterialProperty(Cesium.Color.ORANGE.withAlpha(0.05))
        }
    });
}

function drawSatellite(position, id, x, y, z, timestamp) {
  return viewer.entities.add({
    position: position,
    point: {
      pixelSize: 3,
      color: Cesium.Color.LIME,
      outlineColor: Cesium.Color.WHITE,
      outlineWidth: 1,
      heightReference: Cesium.HeightReference.NONE
    },
    description: `
      <table style="font-family: sans-serif; font-size: 12px;">
        <tr><td><b>Satellite ID</b></td><td>${id}</td></tr>
        <tr><td><b>Timestamp (ms)</b></td><td>${timestamp}</td></tr>
        <tr><td><b>Position X (m)</b></td><td>${x.toFixed(2)}</td></tr>
        <tr><td><b>Position Y (m)</b></td><td>${y.toFixed(2)}</td></tr>
        <tr><td><b>Position Z (m)</b></td><td>${z.toFixed(2)}</td></tr>
      </table>
    `
  });
}

function drawGroundStation(position, id) {
  return viewer.entities.add({
    position: position,
    point: {
      pixelSize: 5,
      color: Cesium.Color.RED,
      outlineColor: Cesium.Color.WHITE,
      outlineWidth: 1,
      heightReference: Cesium.HeightReference.NONE
    },
    label: {
      text: id,
      font: '8px sans-serif',
      pixelOffset: new Cesium.Cartesian2(0, -12),
      fillColor: Cesium.Color.WHITE,
      showBackground: true,
      backgroundColor: Cesium.Color.fromAlpha(Cesium.Color.RED, 0.6),
      heightReference: Cesium.HeightReference.NONE,
      scale: 0.8
    }
  });
}

function InputValid() {
  const status = document.getElementById('statusMsg');
  const positionFileInput = document.getElementById('positionFile');
  const demandFileInput = document.getElementById('demandFile');
  const timeInput = document.getElementById('time');

  if (!positionFileInput.files[0]) {
    status.textContent = 'Please select position file';
    return false;
  }

  if (!demandFileInput.files[0]) {
    status.textContent = 'Please select demand file';
    return false;
  }

  if (timeInput.value === '') {
    status.textContent = 'Please enter start time';
    return false;
  }

  if (!Number.isFinite(Number(timeInput.value))) {
    status.textContent = 'Time must be a number';
    return false;
  }

  return true;
}

function filterDemandsAndPositionswithTimestamp(positionRows, demandRows, time, timeEnd) {
    const gsPositions = new Map();
    const satelliteMap = new Map();
    const demandMap = new Map();

    for(const row of demandRows) {
        const timestamp = Number(row['Timestamp(ms)']);
        if (!Number.isFinite(timestamp) || timestamp < time || timestamp > timeEnd) {
            continue;
        }
        const source = row['Source'];
        const dest = row['Destination'];
        const len = Number(row['Length(Mb)']);
        
        if (!gsPositions.has(source)) 
            gsPositions.set(source, null);

        if (!gsPositions.has(dest))
            gsPositions.set(dest, null);

        if (!demandMap.has(source))
            demandMap.set(source, new Map());

        demandMap.get(source).set(dest, len);
    }

    for (const row of positionRows) {
        const timestamp = Number(row['TimeStamp(ms)']);
        if (!Number.isFinite(timestamp) || timestamp < time || timestamp > timeEnd) {
            continue;
        }

        const id = String(row.Id || row.ID || '').trim();
        const x = Number(row['X(m)'] || row.X || row.x);
        const y = Number(row['Y(m)'] || row.Y || row.y);
        const z = Number(row['Z(m)'] || row.Z || row.z);

        // Skip invalid data
        if (!id || !Number.isFinite(x) || !Number.isFinite(y) || !Number.isFinite(z)) {
            continue;
        }

        if (gsPositions.has(id)) {
            gsPositions.set(id, new Cesium.Cartesian3(x, y, z));
        } else {
            const existing = satelliteMap.get(id);
            if (!existing || timestamp > existing.timestamp)
                satelliteMap.set(id, {id, x, y, z, timestamp});
        }
    }

    return {satelliteMap, gsPositions, demandMap};
}

function drawSatellites(satelliteMap) {
  const created = [];
  for (const [id, satellite] of satelliteMap) {
    const { x, y, z, timestamp } = satellite;

    // Use Cartesian3 coordinates
    const basePosition = new Cesium.Cartesian3(x, y, z);

    // Add green satellite point
    const entity = drawSatellite(basePosition, id, x, y, z, timestamp);
    created.push(entity);
  }

  return created;
}

// Display each satellite's vector field in the specified time range
async function displayDemandsInTimeRange() {
  const status = document.getElementById('statusMsg');
  const positionFileInput = document.getElementById('positionFile');
  const demandFileInput = document.getElementById('demandFile');
  const timeInput = document.getElementById('time');

  if (!InputValid()) {
    return;
  }

  const timeRange = getTimeIntervalOfPositionFile(positionFileInput.files[0].name);
  const time = Number(timeInput.value) * 1000; // convert to ms
  const timeEnd = time + timeRange - 1;

  status.textContent = 'Loading CSV files...';
  viewer.entities.removeAll();

  try {
    // Load CSV files
    const [positionRows, demandRows] = await Promise.all([
      loadCsvFromFile(positionFileInput.files[0]),
      loadCsvFromFile(demandFileInput.files[0])
    ]);
    
    if (positionRows.length === 0) {
      status.textContent = 'No data in position file';
      return;
    }

    if (demandRows.length === 0) {
      status.textContent = 'No data in demand file';
      return;
    }

    // Filter satellites in specified time range
    const {satelliteMap, gsPositions, demandMap} = filterDemandsAndPositionswithTimestamp(positionRows, demandRows, time, timeEnd);

    if (demandMap.size === 0) {
      status.textContent = `No demands found with demand data in time range ${time}~${timeEnd}ms.`;
      return;
    }

    // Draw all satellites
    const created = drawSatellites(satelliteMap);

    // Draw ground stations (sources and destinations)
    for (const [gsId, gsPosition] of gsPositions) {
      if (gsPosition) {
        created.push(drawGroundStation(gsPosition, gsId));
      }
    }

    // Draw Demand Geodesics
    for (const [sourceId, destMap] of demandMap) {
      const sourcePos = gsPositions.get(sourceId);
      if (!sourcePos) continue;

      for (const [destId, length] of destMap) {
        const destPos = gsPositions.get(destId);
        if (!destPos) continue;
        // Draw geodesic line
        created.push(drawGeodesicLine(sourcePos, destPos, DEMAND_WIDTH_FACTOR * length));
      }
    }
    
    if (created.length > 0) {
      await viewer.zoomTo(viewer.entities);
      const gsCount = gsPositions.size;
      const satelliteCount = satellites.size;
      status.textContent = `Successfully displayed ${satelliteCount} satellites and ${gsCount} ground stations in time range ${time}~${timeEnd}ms.`;
    }

  } catch (error) {
    status.textContent = 'Failed to load files: ' + error.message;
    console.error('File loading error:', error);
  }
}

// Bind UI events
const positionFileInput = document.getElementById('positionFile');
const demandFileInput = document.getElementById('demandFile');
const timeInput = document.getElementById('time');
const loadBtn = document.getElementById('loadBtn');

// Check input status and update button
function updateButtonState() {
  const hasPositionFile = positionFileInput.files.length > 0;
  const hasDemandFile = demandFileInput.files.length > 0;
  const hasTime = timeInput.value !== '';
  loadBtn.disabled = !(hasPositionFile && hasDemandFile && hasTime);
  
  if (!hasPositionFile || !hasDemandFile) {
    document.getElementById('statusMsg').textContent = 'Please select position.csv and demand.csv files and enter time';
  } else if (!hasTime) {
    document.getElementById('statusMsg').textContent = 'Please enter time';
  } else {
    const time = Number(timeInput.value);
    if (Number.isFinite(time)) {
      document.getElementById('statusMsg').textContent = `Ready, will display satellites from ${time}s`;
    } else {
      document.getElementById('statusMsg').textContent = 'Ready, click button to display satellites';
    }
  }
}

// Listen to input changes
positionFileInput.addEventListener('change', updateButtonState);
demandFileInput.addEventListener('change', updateButtonState);
timeInput.addEventListener('input', updateButtonState);timeInput

// Enter key quick execution
timeInput.addEventListener('keydown', (e) => {
  if (e.key === 'Enter' && !loadBtn.disabled) {
    displayDemandsInTimeRange();
  }
});

// Load button click event
loadBtn.addEventListener('click', displayDemandsInTimeRange);
