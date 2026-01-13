const ARROW_SCALE_FACTOR = 1000000;

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

function getSourceDestinationFromFile(fieldFileName) {
  const parts = fieldFileName.split('#');

  if (parts.length < 5) return console.log("Field file name does not contain source-destination info!");

  const sourceDestPart = parts[3];
  const commaIndex = sourceDestPart.indexOf(",");
  if (commaIndex !== -1) {
    const source = sourceDestPart.slice(1, commaIndex).trim();
    const destination = sourceDestPart.slice(commaIndex + 1, sourceDestPart.length - 1).trim();
    return { source, destination };
  } else {
    console.log("Source-Destination part does not contain a comma!");
    return { source: null, destination: null };
  }
}

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
function createArrowFromTo(basePosition, direction, scale = 100000) {
  const scaledDirection = Cesium.Cartesian3.multiplyByScalar(
    direction,
    scale,
    new Cesium.Cartesian3()
  );
  const targetPosition = Cesium.Cartesian3.add(
    basePosition,
    scaledDirection,
    new Cesium.Cartesian3()
  );

  // Create arrow entity (from basePosition to targetPosition)
  return viewer.entities.add({
    polyline: {
      positions: [basePosition, targetPosition],
      width: 4,
      material: new Cesium.PolylineArrowMaterialProperty(Cesium.Color.YELLOW),
      clampToGround: false
    },
    name: 'vector_arrow',
    description: 'Vector field arrow'
  });
}

function drawSatellite(position, id, x, y, z, timestamp, fieldData) {
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
        <tr><td><b>Field Vector X</b></td><td>${fieldData.x.toFixed(6)}</td></tr>
        <tr><td><b>Field Vector Y</b></td><td>${fieldData.y.toFixed(6)}</td></tr>
        <tr><td><b>Field Vector Z</b></td><td>${fieldData.z.toFixed(6)}</td></tr>
        <tr><td><b>Field Magnitude</b></td><td>${fieldData.magnitude.toFixed(6)}</td></tr>
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
  const fieldFileInput = document.getElementById('fieldFile');
  const timeInput = document.getElementById('time');

  if (!positionFileInput.files[0]) {
    status.textContent = 'Please select position file';
    return false;
  }

  if (!fieldFileInput.files[0]) {
    status.textContent = 'Please select vector field file';
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

function satelliteToFieldVector(fieldRows) {
  const fieldMap = new Map();
  for (const field of fieldRows) {
    const satelliteId = String(field.Satellite || '').trim();
    if (satelliteId) {
      const fieldX = Number(field.Field_X);
      const fieldY = Number(field.Field_Y);
      const fieldZ = Number(field.Field_Z);
      const fieldMagnitude = Number(field.Field_Magnitude);
      
      if (Number.isFinite(fieldX) && Number.isFinite(fieldY) && Number.isFinite(fieldZ) && Number.isFinite(fieldMagnitude)) {
        fieldMap.set(satelliteId, { x: fieldX, y: fieldY, z: fieldZ, magnitude: fieldMagnitude } );
      }
    }
  }
  return fieldMap;
}

function filterSatellitesInTimeRangeWithSourceDestination(positionRows, time, timeEnd, fieldMap, source, destination) {
  const satelliteMap = new Map();
  let sourcePosition = null;
  let destinationPosition = null;
  for (const row of positionRows) {
    const timestamp = Number(row['TimeStamp(ms)']);
    
    // Check if within time range
    if (!Number.isFinite(timestamp) || timestamp < time || timestamp > timeEnd) {
      continue;
    }
    const id = String(row.Id || row.ID || '').trim();
    const x = Number(row['X(m)'] || row.X || row.x);
    const y = Number(row['Y(m)'] || row.Y || row.y);
    const z = Number(row['Z(m)'] || row.Z || row.z);

    if (id === source) {
      sourcePosition = new Cesium.Cartesian3(x, y, z);
    } else if (id === destination) {
      destinationPosition = new Cesium.Cartesian3(x, y, z);
    }

    // Only process satellites with vector field data
    if (!fieldMap.has(id)) {
      continue;
    }
    // Skip invalid data
    if (!id || !Number.isFinite(x) || !Number.isFinite(y) || !Number.isFinite(z)) {
      continue;
    }

    // Keep latest data for each satellite in the time range
    const existing = satelliteMap.get(id);
    if (!existing || timestamp > existing.timestamp) {
      satelliteMap.set(id, {
        id,
        x,
        y,
        z,
        timestamp
      });
    }
  }
  return {satelliteMap, sourcePosition, destinationPosition};
}

function drawSatellitesAndArrows(satelliteMap, fieldMap) {
  const created = [];
  for (const [id, satellite] of satelliteMap) {
    const { x, y, z, timestamp } = satellite;

    // Use Cartesian3 coordinates
    const basePosition = new Cesium.Cartesian3(x, y, z);

    // Get vector field data for this satellite from the map
    const fieldData = fieldMap.get(id);
    if (!fieldData) continue;

    // Vector field direction
    const direction = new Cesium.Cartesian3(fieldData.x, fieldData.y, fieldData.z);

    // Add green satellite point
    const entity = drawSatellite(basePosition, id, x, y, z, timestamp, fieldData);
    created.push(entity);

    // Create arrow
    const arrowEntity = createArrowFromTo(basePosition, direction, ARROW_SCALE_FACTOR*fieldData.magnitude);
    created.push(arrowEntity);
  }

  return created;
}

// Display each satellite's vector field in the specified time range
async function displayFieldsInTimeRange() {
  const status = document.getElementById('statusMsg');
  const positionFileInput = document.getElementById('positionFile');
  const fieldFileInput = document.getElementById('fieldFile');
  const timeInput = document.getElementById('time');

  if (!InputValid()) {
    return;
  }

  const timeRange = getTimeIntervalOfPositionFile(positionFileInput.files[0].name);
  const time = Number(timeInput.value) * 1000; // convert to ms
  const timeEnd = time + timeRange - 1;
  const { source, destination } = getSourceDestinationFromFile(fieldFileInput.files[0].name);

  status.textContent = 'Loading CSV files...';
  viewer.entities.removeAll();

  try {
    // Load CSV files
    const [positionRows, fieldRows] = await Promise.all([
      loadCsvFromFile(positionFileInput.files[0]),
      loadCsvFromFile(fieldFileInput.files[0])
    ]);
    
    if (positionRows.length === 0) {
      status.textContent = 'No data in position file';
      return;
    }

    if (fieldRows.length === 0) {
      status.textContent = 'No data in vector field file';
      return;
    }

    // Build satellite ID to vector field mapping
    const fieldMap = satelliteToFieldVector(fieldRows);

    // Filter satellites in specified time range
    const {satelliteMap, sourcePosition, destinationPosition} = filterSatellitesInTimeRangeWithSourceDestination(positionRows, time, timeEnd, fieldMap, source, destination);

    if (satelliteMap.size === 0) {
      status.textContent = `No satellites found with vector field data in time range ${time}~${timeEnd}ms`;
      return;
    }

    // Draw all satellites and arrows for this time range
    const created = drawSatellitesAndArrows(satelliteMap, fieldMap);

    // Draw ground stations (source and destination)
    if (sourcePosition) {
      created.push(drawGroundStation(sourcePosition, source));
    }
    if (destinationPosition) {
      created.push(drawGroundStation(destinationPosition, destination));
    }

    // Draw ground stations if positions are available
    if (created.length > 0) {
      await viewer.zoomTo(viewer.entities);
      const satelliteCount = satelliteMap.size;
      status.textContent = `Successfully displayed ${satelliteCount} satellites and vector arrows in time range ${time}~${timeEnd}ms with ${source} as source and ${destination} as destination.`;
    }

  } catch (error) {
    status.textContent = 'Failed to load files: ' + error.message;
    console.error('File loading error:', error);
  }
}

// Bind UI events
const positionFileInput = document.getElementById('positionFile');
const fieldFileInput = document.getElementById('fieldFile');
const timeInput = document.getElementById('time');
const loadBtn = document.getElementById('loadBtn');

// Check input status and update button
function updateButtonState() {
  const hasPositionFile = positionFileInput.files.length > 0;
  const hasFieldFile = fieldFileInput.files.length > 0;
  const hasTime = timeInput.value !== '';
  loadBtn.disabled = !(hasPositionFile && hasFieldFile && hasTime);
  
  if (!hasPositionFile || !hasFieldFile) {
    document.getElementById('statusMsg').textContent = 'Please select position.csv and field.csv files and enter time';
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
fieldFileInput.addEventListener('change', updateButtonState);
timeInput.addEventListener('input', updateButtonState);timeInput

// Enter key quick execution
timeInput.addEventListener('keydown', (e) => {
  if (e.key === 'Enter' && !loadBtn.disabled) {
    displayFieldsInTimeRange();
  }
});

// Load button click event
loadBtn.addEventListener('click', displayFieldsInTimeRange);
