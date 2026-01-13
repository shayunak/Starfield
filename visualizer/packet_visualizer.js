// Cesium Ion Token
Cesium.Ion.defaultAccessToken = '';

const App = {
  // WGS84 constants
  WGS84: {
    a: 6378137.0,
    f: 1.0 / 298.257223563,
    get b() { return this.a * (1 - this.f); },
    get e2() { return 1 - (this.b * this.b) / (this.a * this.a); }
  },

  // CSV streaming utility
  async *streamLinesFromFile(file) {
    const reader = file.stream().pipeThrough(new TextDecoderStream()).getReader();
    let { value, done } = await reader.read();
    let buffer = value || '';
    while (!done) {
      const lastNewline = buffer.lastIndexOf('\n');
      if (lastNewline !== -1) {
        const lines = buffer.slice(0, lastNewline).split('\n');
        for (const line of lines) yield line.replace(/\r$/, '');
        buffer = buffer.slice(lastNewline + 1);
      }
      ({ value, done } = await reader.read());
      buffer += value || '';
    }
    if (buffer.length > 0) yield buffer.replace(/\r$/, '');
  },

  // CSV parsing
  parseCsvHeaders(line, requiredFields) {
    const headers = line.split(',');
    const idx = {};
    for (const field of requiredFields) {
      idx[field] = headers.indexOf(field);
    }
    return idx;
  },

  // Height calculation from geocentric radius
  heightFromGeocentricRadius(latDeg, radiusM) {
    const phi = Cesium.Math.toRadians(Number(latDeg));
    const sinPhi = Math.sin(phi), cosPhi = Math.cos(phi);
    const N = this.WGS84.a / Math.sqrt(1 - this.WGS84.e2 * sinPhi * sinPhi);
    const rSurface = Math.sqrt((N * cosPhi) ** 2 + (N * (1 - this.WGS84.e2) * sinPhi) ** 2);
    return Number(radiusM) - rSurface;
  },

  // Generic position reader with filtering
  async readPositions(positionFile, filterFn, processFn) {
    const results = new Map();
    let headerParsed = false, idx = null;
    
    for await (const line of this.streamLinesFromFile(positionFile)) {
      if (!headerParsed) {
        idx = this.parseCsvHeaders(line, ['TimeStamp(ms)', 'Id', 'Latitude(deg)', 'Longitude(deg)', 'Radius(m)']);
        headerParsed = true;
        continue;
      }
      if (!line) continue;

      const cols = line.split(',');
      const id = String(cols[idx['Id']] ?? '').trim();
      const ts = Number(cols[idx['TimeStamp(ms)']]);
      const lat = Number(cols[idx['Latitude(deg)']]);
      const lon = Number(cols[idx['Longitude(deg)']]);
      const r = Number(cols[idx['Radius(m)']]);

      if (!filterFn || filterFn(id, ts, lat, lon, r)) {
        const row = { Id: id, 'TimeStamp(ms)': ts, 'Latitude(deg)': lat, 'Longitude(deg)': lon, 'Radius(m)': r };
        processFn(results, id, row, ts);
      }
    }
    return results;
  },

  // Extract packet hops
  async extractPacketData(packetFile, packetId) {
    const deviceSet = new Set();
    const hops = [];
    let minTs = Number.POSITIVE_INFINITY, maxTs = Number.NEGATIVE_INFINITY;
    let headerParsed = false, idx = null;

    for await (const line of this.streamLinesFromFile(packetFile)) {
      if (!headerParsed) {
        idx = this.parseCsvHeaders(line, ['TimeStamp(ms)', 'FromDevice', 'ToDevice', 'PacketId']);
        headerParsed = true;
        continue;
      }
      if (!line) continue;

      const cols = line.split(',');
      const pid = (cols[idx['PacketId']] ?? '').trim();
      if (pid !== String(packetId).trim()) continue;

      const ts = Number(cols[idx['TimeStamp(ms)']]);
      const from = (cols[idx['FromDevice']] ?? '').trim();
      const to = (cols[idx['ToDevice']] ?? '').trim();

      if (Number.isFinite(ts)) {
        if (from) deviceSet.add(from);
        if (to) deviceSet.add(to);
        if (ts < minTs) minTs = ts;
        if (ts > maxTs) maxTs = ts;
        hops.push({ 'TimeStamp(ms)': ts, FromDevice: from, ToDevice: to, PacketId: pid });
      }
    }

    if (!Number.isFinite(minTs) || !Number.isFinite(maxTs)) {
      minTs = NaN; maxTs = NaN;
    }
    return { hops, deviceSet, minTs, maxTs };
  },

  // Parse topology CSV
  async parseTopology(file) {
    const pairs = [];
    const idSet = new Set();
    const seen = new Set();
    let headerParsed = false, idx = null;

    for await (const line of this.streamLinesFromFile(file)) {
      if (!headerParsed) {
        idx = this.parseCsvHeaders(line, ['FirstSatellite', 'SecondSatellite']);
        headerParsed = true;
        continue;
      }
      if (!line) continue;

      const cols = line.split(',');
      const a = String(cols[idx['FirstSatellite']] ?? '').trim();
      const b = String(cols[idx['SecondSatellite']] ?? '').trim();
      if (!a || !b || a === b) continue;

      const [x, y] = a < b ? [a, b] : [b, a];
      const key = `${x}||${y}`;
      if (seen.has(key)) continue;

      seen.add(key);
      idSet.add(x);
      idSet.add(y);
      pairs.push([x, y]);
    }
    return { pairs, idSet };
  },

  // Find nearest position by timestamp
  findNearestPosition(positionIndex, deviceId, targetTs, maxDeltaMs = 500) {
    const positions = positionIndex.get(deviceId);
    if (!positions || positions.length === 0) return null;

    let best = null, bestDiff = Infinity;
    for (const pos of positions) {
      const diff = Math.abs(Number(pos['TimeStamp(ms)']) - targetTs);
      if (diff < bestDiff && diff <= maxDeltaMs) {
        best = pos;
        bestDiff = diff;
      }
    }
    return best;
  },

  // Create position index by device ID
  buildPositionIndex(positionRows) {
    const index = new Map();
    for (const row of positionRows) {
      const id = String(row.Id ?? '').trim();
      if (!id) continue;
      if (!index.has(id)) index.set(id, []);
      index.get(id).push(row);
    }
    // Sort by timestamp
    for (const positions of index.values()) {
      positions.sort((a, b) => Number(a['TimeStamp(ms)']) - Number(b['TimeStamp(ms)']));
    }
    return index;
  },

  // Parse simulation filename to extract info
  // Expected format: SimulationSummary#2025_09_23,08_22_01#Starlink(72,22)#1000ms#1000s
  parseSimulationFilename(filename) {
    try {
      // Remove .csv extension if present
      const basename = filename.replace(/\.csv$/i, '');
      
      // Split by '#' delimiter
      const parts = basename.split('#');
      
      if (parts.length < 5) {
        console.warn('Filename does not match expected format');
        return null;
      }

      // Extract constellation info from parts[2]: e.g., "Starlink(72,22)"
      const constellationPart = parts[2];
      const constellationMatch = constellationPart.match(/^([^(]+)\((\d+),(\d+)\)$/);
      
      if (!constellationMatch) {
        console.warn('Cannot parse constellation info from:', constellationPart);
        return null;
      }

      const Constellation_name = constellationMatch[1].trim();
      const Number_of_orbits = parseInt(constellationMatch[2], 10);
      const Satellites_per_orbit = parseInt(constellationMatch[3], 10);

      // Extract time duration from parts[3]: e.g., "1000ms"
      const timePart = parts[3];
      const timeMatch = timePart.match(/^(\d+)ms$/);
      
      if (!timeMatch) {
        console.warn('Cannot parse time duration from:', timePart);
        return null;
      }

      const timeduration = parseInt(timeMatch[1], 10);

      return {
        Constellation_name,
        Number_of_orbits,
        Satellites_per_orbit,
        timeduration
      };
    } catch (error) {
      console.error('Error parsing simulation filename:', error);
      return null;
    }
  },

  // Generate Grid Plus topology structure
  // Returns array of topology pairs
  generateGridPlusTopology(numberOfOrbits, numberOfSatellitesPerOrbit, constellationName) {
    if (!numberOfOrbits || !numberOfSatellitesPerOrbit || !constellationName) {
      console.warn('Invalid parameters for GridPlus topology generation');
      return { pairs: [], idSet: new Set() };
    }

    const gridPlus = new Array(4 * numberOfOrbits * numberOfSatellitesPerOrbit);
    const idSet = new Set();
    
    for (let orbit = 0; orbit < numberOfOrbits; orbit++) {
      const orbitOnLeft = (orbit + numberOfOrbits - 1) % numberOfOrbits;
      const orbitOnRight = (orbit + 1) % numberOfOrbits;
      
      for (let satellite = 0; satellite < numberOfSatellitesPerOrbit; satellite++) {
        const nextIdInOrbit = (satellite + 1) % numberOfSatellitesPerOrbit;
        const previousIdInOrbit = (satellite + numberOfSatellitesPerOrbit - 1) % numberOfSatellitesPerOrbit;
        const baseIndex = 4 * (orbit * numberOfSatellitesPerOrbit + satellite);
        
        const currentSat = `${constellationName}-${orbit}-${satellite}`;
        const nextSat = `${constellationName}-${orbit}-${nextIdInOrbit}`;
        const prevSat = `${constellationName}-${orbit}-${previousIdInOrbit}`;
        const leftSat = `${constellationName}-${orbitOnLeft}-${satellite}`;
        const rightSat = `${constellationName}-${orbitOnRight}-${satellite}`;
        
        gridPlus[baseIndex] = {
          firstSatellite: currentSat,
          secondSatellite: nextSat
        };
        gridPlus[baseIndex + 1] = {
          firstSatellite: currentSat,
          secondSatellite: prevSat
        };
        gridPlus[baseIndex + 2] = {
          firstSatellite: currentSat,
          secondSatellite: leftSat
        };
        gridPlus[baseIndex + 3] = {
          firstSatellite: currentSat,
          secondSatellite: rightSat
        };
        
        // Collect all unique satellite IDs
        idSet.add(currentSat);
        idSet.add(nextSat);
        idSet.add(prevSat);
        idSet.add(leftSat);
        idSet.add(rightSat);
      }
    }
    
    // Convert to pairs format (compatible with parseTopology output)
    const pairs = [];
    const seen = new Set();
    
    for (const link of gridPlus) {
      const a = link.firstSatellite;
      const b = link.secondSatellite;
      const [x, y] = a < b ? [a, b] : [b, a];
      const key = `${x}||${y}`;
      
      if (!seen.has(key)) {
        seen.add(key);
        pairs.push([x, y]);
      }
    }
    
    console.log(`Generated GridPlus topology: ${pairs.length} unique links for ${constellationName} (${numberOfOrbits}×${numberOfSatellitesPerOrbit})`);
    
    return { pairs, idSet };
  }
};

// Initialize Cesium viewer
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

// Global state for topology
let topologyState = {
  lines: [],
  points: [],
  visible: true,
  clickHandlerAttached: false
};

// Visualization functions
function drawEntity(position, options = {}) {
  return viewer.entities.add({
    position,
    ...options
  });
}

function drawPolyline(positions, material, width = 2.0) {
  return drawEntity(null, {
    polyline: {
      positions,
      width,
      material,
      arcType: Cesium.ArcType.NONE
    }
  });
}

function createEntityDescription(data) {
  return Object.entries(data)
    .map(([key, value]) => `<tr><td><b>${key}</b></td><td>${value}</td></tr>`)
    .join('');
}

// Main packet visualization
async function visualizePacket(packetId) {
  const status = document.getElementById('statusMsg');
  const packetFile = document.getElementById('packetFile').files[0];
  const positionFile = document.getElementById('positionFile').files[0];

  if (!packetId?.trim()) {
    status.textContent = 'Enter PacketId.';
    return;
  }
  if (!packetFile || !positionFile) {
    status.textContent = 'Please select summary.csv and position.csv first.';
    return;
  }

  viewer.entities.removeAll();
  topologyState.lines = [];
  topologyState.points = [];

  // Parse simulation filename to extract constellation info
  let Constellation_name = null;
  let Number_of_orbits = null;
  let Satellites_per_orbit = null;
  let timeduration = null;
  
  const filenameInfo = App.parseSimulationFilename(packetFile.name);
  if (filenameInfo) {
    Constellation_name = filenameInfo.Constellation_name;
    Number_of_orbits = filenameInfo.Number_of_orbits;
    Satellites_per_orbit = filenameInfo.Satellites_per_orbit;
    timeduration = filenameInfo.timeduration;
    
    console.log('Parsed simulation info:', {
      Constellation_name,
      Number_of_orbits,
      Satellites_per_orbit,
      timeduration
    });
  }

  try {
    status.textContent = 'Loading packet data...';
    const { hops, deviceSet, minTs, maxTs } = await App.extractPacketData(packetFile, packetId);
    
    if (hops.length === 0) {
      status.textContent = `Cannot find record for PacketId = ${packetId}.`;
      return;
    }

    const posWindowMin = Math.floor(minTs / 1000) * 1000;
    const posWindowMax = Math.ceil(maxTs / 1000) * 1000;

    status.textContent = 'Loading position data...';
    const positionRows = await App.readPositions(
      positionFile,
      (id, ts) => deviceSet.has(id) && Number.isFinite(ts) && ts >= posWindowMin && ts <= posWindowMax,
      (results, id, row) => {
        if (!results.has(id)) results.set(id, []);
        results.get(id).push(row);
      }
    );

    // Convert to array format
    const positionArray = [];
    for (const positions of positionRows.values()) {
      positionArray.push(...positions);
    }

    const positionIndex = App.buildPositionIndex(positionArray);
    const deviceColors = {};
    Array.from(deviceSet).forEach(id => {
      deviceColors[id] = Cesium.Color.fromRandom({ alpha: 1.0 });
    });

    // Build device to packet timestamp mapping
    const devicePacketTimes = new Map();
    for (const hop of hops) {
      const ts = Number(hop['TimeStamp(ms)']);
      if (hop.FromDevice) {
        if (!devicePacketTimes.has(hop.FromDevice)) devicePacketTimes.set(hop.FromDevice, []);
        devicePacketTimes.get(hop.FromDevice).push(ts);
      }
      if (hop.ToDevice) {
        if (!devicePacketTimes.has(hop.ToDevice)) devicePacketTimes.set(hop.ToDevice, []);
        devicePacketTimes.get(hop.ToDevice).push(ts);
      }
    }

    // Draw device points
    const devicePositions = new Map();
    
    // Find the starting device (first hop's FromDevice)
    const sortedHopsForStart = hops.slice().sort((a, b) => Number(a['TimeStamp(ms)']) - Number(b['TimeStamp(ms)']));
    const startingDevice = sortedHopsForStart.length > 0 ? sortedHopsForStart[0].FromDevice : null;
    
    for (const deviceId of deviceSet) {
      const positions = positionIndex.get(deviceId);
      if (!positions?.length) continue;

      // Get all packet timestamps for this device (PacketId)
      const packetTimestamps = devicePacketTimes.get(deviceId) || [];
      if (packetTimestamps.length === 0) continue;

      // Sort packet timestamps for this device
      packetTimestamps.sort((a, b) => a - b);
      
      // Use first timestamp for starting device, last timestamp for others
      const devicePacketTime = (deviceId === startingDevice) 
        ? packetTimestamps[0] 
        : packetTimestamps[packetTimestamps.length - 1];
      
      // Find position closest to when packet arrived at this device
      let bestPos = positions[0];
      let bestDiff = Math.abs(Number(bestPos['TimeStamp(ms)']) - devicePacketTime);
      
      for (const pos of positions) {
        const diff = Math.abs(Number(pos['TimeStamp(ms)']) - devicePacketTime);
        if (diff < bestDiff) {
          bestPos = pos;
          bestDiff = diff;
        }
      }

      const lat = Number(bestPos['Latitude(deg)']);
      const lon = Number(bestPos['Longitude(deg)']);
      const radius = Number(bestPos['Radius(m)']);
      
      if (!Number.isFinite(lat) || !Number.isFinite(lon)) continue;

      const height = Number.isFinite(radius) ? App.heightFromGeocentricRadius(lat, radius) : 0;
      const position = Cesium.Cartesian3.fromDegrees(lon, lat, height);
      devicePositions.set(deviceId, { position, row: bestPos, packetTimestamp: devicePacketTime });

      // Create label showing only the first packet time for this device
      const timeLabel = `${deviceId}\n${devicePacketTime}`;

      drawEntity(position, {
        point: {
          pixelSize: 10,
          color: deviceColors[deviceId],
          outlineColor: Cesium.Color.WHITE,
          outlineWidth: 1.5
        },
        label: {
          text: timeLabel,
          font: '12px sans-serif',
          pixelOffset: new Cesium.Cartesian2(0, -18),
          fillColor: Cesium.Color.WHITE,
          showBackground: true,
          backgroundColor: Cesium.Color.fromAlpha(deviceColors[deviceId], 0.6),
          disableDepthTestDistance: Number.POSITIVE_INFINITY
        },
        description: `<table>${createEntityDescription({
          'Device': deviceId,
          'PacketId': packetId,
          'Packet Time': devicePacketTime,
          'Position TimeStamp(ms)': Number(bestPos['TimeStamp(ms)']),
          'Latitude(deg)': lat,
          'Longitude(deg)': lon,
          'Radius(m)': radius,
          'Height above WGS84 (m)': height.toFixed(2)
        })}</table>`
      });
    }

    // Draw hop lines (skip the last hop for any packet)
    const sortedHopsForLast = hops.slice().sort((a, b) => Number(a['TimeStamp(ms)']) - Number(b['TimeStamp(ms)']));
    const lastHop = sortedHopsForLast.length > 0 ? sortedHopsForLast[sortedHopsForLast.length - 1] : null;
    
    for (const hop of hops) {
      const fromPos = devicePositions.get(hop.FromDevice);
      const toPos = devicePositions.get(hop.ToDevice);
      if (!fromPos || !toPos) continue;

      // Skip the last hop (by timestamp)
      if (lastHop && hop['TimeStamp(ms)'] === lastHop['TimeStamp(ms)'] && 
          hop.FromDevice === lastHop.FromDevice && hop.ToDevice === lastHop.ToDevice) {
        continue;
      }

      drawPolyline(
        [fromPos.position, toPos.position],
        new Cesium.PolylineOutlineMaterialProperty({
          color: Cesium.Color.fromAlpha(deviceColors[hop.FromDevice] ?? Cesium.Color.CYAN, 0.9),
          outlineWidth: 1,
          outlineColor: Cesium.Color.BLACK
        })
      );
    }

    // Prepare device path for info display
    const sortedHops = hops.slice().sort((a, b) => Number(a['TimeStamp(ms)']) - Number(b['TimeStamp(ms)']));
    const seen = new Set();

    for (const hop of sortedHops) {
      for (const deviceId of [hop.FromDevice, hop.ToDevice]) {
        if (!deviceId || seen.has(deviceId)) continue;
        const devicePos = devicePositions.get(deviceId);
        if (devicePos) {
          seen.add(deviceId);
        }
      }
    }

    if (devicePositions.size > 0) {
      await viewer.zoomTo(viewer.entities);
    }

    const devicePath = Array.from(seen).join(' → ');
    
    // Display packet info in fixed area (always visible)
    const packetInfoDiv = document.getElementById('packetInfo');
    packetInfoDiv.textContent = `PacketId ${packetId}: ${minTs} ~ ${maxTs}, ${seen.size-1} nodes, path: ${devicePath}`;
    packetInfoDiv.style.display = 'block';
    
    // Clear status message since info is now permanently displayed
    status.textContent = 'Packet visualization complete';

  } catch (error) {
    console.error(error);
    status.textContent = 'Failed to load data. Please check files.';
  }
}

// Topology visualization
async function visualizeTopology() {
  const status = document.getElementById('statusMsg');
  const packetId = document.getElementById('packetInput')?.value?.trim();
  const packetFile = document.getElementById('packetFile')?.files?.[0];
  const topologyFile = document.getElementById('topologyFile').files[0];
  const positionFile = document.getElementById('positionFile').files[0];
  const useGridPlus = document.getElementById('gridCheckbox')?.checked || false;

  // Check required files based on Grid+ option
  if (!useGridPlus && !topologyFile) {
    status.textContent = 'Please select topology.csv or check Grid+ option.';
    return;
  }
  if (!positionFile) {
    status.textContent = 'Please select position.csv.';
    return;
  }
  if (!packetId || !packetFile) {
    status.textContent = 'Please enter PacketId and select summary.csv.';
    return;
  }

  try {
    let pairs, idSet;
    
    if (useGridPlus) {
      // Use GridPlus topology generation
      status.textContent = 'Generating GridPlus topology...';
      // Parse filename to get constellation parameters
      const filenameInfo = App.parseSimulationFilename(packetFile.name);
      
      if (!filenameInfo) {
        status.textContent = 'Cannot parse simulation filename for GridPlus. Please check file format.';
        return;
      }
      
      const { Constellation_name, Number_of_orbits, Satellites_per_orbit } = filenameInfo;
      
      console.log('Generating GridPlus topology with:', {
        Constellation_name,
        Number_of_orbits,
        Satellites_per_orbit
      });
      
      const gridPlusResult = App.generateGridPlusTopology(
        Number_of_orbits,
        Satellites_per_orbit,
        Constellation_name
      );
      
      pairs = gridPlusResult.pairs;
      idSet = gridPlusResult.idSet;
      
      if (pairs.length === 0) {
        status.textContent = 'Failed to generate GridPlus topology.';
        return;
      }
      
      console.log(`GridPlus topology generated: ${pairs.length} links, ${idSet.size} satellites`);
    } else {
      // Use topology from file
      status.textContent = 'Loading topology from file...';
      const topologyResult = await App.parseTopology(topologyFile);
      pairs = topologyResult.pairs;
      idSet = topologyResult.idSet;
      
      if (pairs.length === 0) {
        status.textContent = 'No valid satellite pairs in topology.csv.';
        return;
      }
    }

    // Get time window from packet data
    const { minTs, maxTs } = await App.extractPacketData(packetFile, packetId);
    if (!Number.isFinite(minTs) || !Number.isFinite(maxTs)) {
      status.textContent = `Cannot find record for PacketId = ${packetId}.`;
      return;
    }

    const posWindowMin = Math.floor(minTs / 1000) * 1000;
    const posWindowMax = Math.ceil(maxTs / 1000) * 1000;
    const center = (posWindowMin + posWindowMax) / 2;

    status.textContent = `Loading positions in window ${posWindowMin} ~ ${posWindowMax}...`;
    const satellitePositions = await App.readPositions(
      positionFile,
      (id, ts) => idSet.has(id) && Number.isFinite(ts) && ts >= posWindowMin && ts <= posWindowMax,
      (results, id, row, ts) => {
        const diff = Math.abs(ts - center);
        const current = results.get(id);
        if (!current || diff < current.diff) {
          results.set(id, { row, ts, diff });
        }
      }
    );

    // Remove old topology entities
    for (const entity of [...topologyState.lines, ...topologyState.points]) {
      try {
        viewer.entities.remove(entity);
      } catch (e) {}
    }

    // Draw topology links
    topologyState.lines = [];
    for (const [satA, satB] of pairs) {
      const posA = satellitePositions.get(satA)?.row;
      const posB = satellitePositions.get(satB)?.row;
      if (!posA || !posB) continue;

      const lat1 = Number(posA['Latitude(deg)']);
      const lon1 = Number(posA['Longitude(deg)']);
      const r1 = Number(posA['Radius(m)']);
      const lat2 = Number(posB['Latitude(deg)']);
      const lon2 = Number(posB['Longitude(deg)']);
      const r2 = Number(posB['Radius(m)']);

      if (![lat1, lon1, lat2, lon2].every(Number.isFinite)) continue;

      const h1 = Number.isFinite(r1) ? App.heightFromGeocentricRadius(lat1, r1) : 0;
      const h2 = Number.isFinite(r2) ? App.heightFromGeocentricRadius(lat2, r2) : 0;

      const entity = drawEntity(null, {
        name: `Topology: ${satA} ↔ ${satB}`,
        polyline: {
          positions: Cesium.Cartesian3.fromDegreesArrayHeights([lon1, lat1, h1, lon2, lat2, h2]),
          width: 2.0,
          material: new Cesium.PolylineOutlineMaterialProperty({
            color: Cesium.Color.GRAY.withAlpha(0.35),
            outlineWidth: 1,
            outlineColor: Cesium.Color.DARKGREEN.withAlpha(0.25)
          }),
          arcType: Cesium.ArcType.NONE
        }
      });
      topologyState.lines.push(entity);
    }

    // Draw topology endpoints
    topologyState.points = [];
    for (const [id, posData] of satellitePositions.entries()) {
      const row = posData.row;
      const lat = Number(row['Latitude(deg)']);
      const lon = Number(row['Longitude(deg)']);
      const r = Number(row['Radius(m)']);
      const ts = Number(row['TimeStamp(ms)']);

      if (!Number.isFinite(lat) || !Number.isFinite(lon)) continue;

      const h = Number.isFinite(r) ? App.heightFromGeocentricRadius(lat, r) : 0;
      const entity = drawEntity(Cesium.Cartesian3.fromDegrees(lon, lat, h), {
        name: `Satellite: ${id}`,
        point: {
          pixelSize: 4,
          color: Cesium.Color.LIME,
          outlineColor: Cesium.Color.WHITE,
          outlineWidth: 0.5
        },
        label: {
          text: id,
          font: '12px sans-serif',
          pixelOffset: new Cesium.Cartesian2(0, -16),
          fillColor: Cesium.Color.WHITE,
          showBackground: true,
          backgroundColor: Cesium.Color.fromAlpha(Cesium.Color.DARKGREEN, 0.6),
          disableDepthTestDistance: Number.POSITIVE_INFINITY,
          show: false
        },
        description: `<table>${createEntityDescription({
          'Satellite': id,
          'TimeStamp(ms)': Number.isFinite(ts) ? ts : 'N/A',
          'Latitude(deg)': lat,
          'Longitude(deg)': lon,
          'Radius(m)': Number.isFinite(r) ? r : 'N/A',
          'Height above WGS84 (m)': h.toFixed(2)
        })}</table>`
      });

      try {
        entity.properties = new Cesium.PropertyBag({ topologyNode: true, satelliteId: id });
      } catch (e) {}
      topologyState.points.push(entity);
    }

    setupTopologyClickHandler();
    setTopologyVisible(true);
    
    if (document.getElementById('packetInfo').style.display !== 'none') {
      status.textContent = `Topology loaded: ${topologyState.lines.length} links, ${topologyState.points.length} nodes`;
    } else {
      status.textContent = `Topology: ${topologyState.lines.length} links, ${topologyState.points.length} nodes (${posWindowMin} ~ ${posWindowMax})`;
    }

  } catch (error) {
    console.error(error);
    status.textContent = 'Failed to render topology. Please check files.';
  }
}

function setupTopologyClickHandler() {
  if (topologyState.clickHandlerAttached) return;
  
  viewer.screenSpaceEventHandler.setInputAction((movement) => {
    // Hide all topology labels
    for (const entity of topologyState.points) {
      if (entity.label) entity.label.show = false;
    }

    const picked = viewer.scene.pick(movement.position);
    const entity = picked?.id;
    if (entity) {
      try {
        const isTopoNode = entity.properties?.getValue()?.topologyNode === true;
        if (isTopoNode && entity.label) {
          entity.label.show = true;
        }
      } catch (e) {}
      viewer.selectedEntity = entity;
    } else {
      viewer.selectedEntity = undefined;
    }
  }, Cesium.ScreenSpaceEventType.LEFT_CLICK);
  
  topologyState.clickHandlerAttached = true;
}

function setTopologyVisible(visible) {
  topologyState.visible = !!visible;
  for (const entity of topologyState.lines) {
    try { entity.show = !!visible; } catch (e) {}
  }
  for (const entity of topologyState.points) {
    try {
      entity.show = !!visible;
      if (entity.label) entity.label.show = false;
    } catch (e) {}
  }
  
  if (!visible && viewer.selectedEntity && 
      [...topologyState.points, ...topologyState.lines].includes(viewer.selectedEntity)) {
    viewer.selectedEntity = undefined;
  }
}

function toggleTopology() {
  const nextVisible = !topologyState.visible;
  setTopologyVisible(nextVisible);
  
  // Show toggle status
  const status = document.getElementById('statusMsg');
  status.textContent = `Topology visibility: ${nextVisible ? 'ON' : 'OFF'}`;
  
  // Clear the toggle message after 2 seconds if packet info is displayed
  if (document.getElementById('packetInfo').style.display !== 'none') {
    setTimeout(() => {
      status.textContent = 'Ready';
    }, 2000);
  }
}

// Event listeners
const input = document.getElementById('packetInput');
const searchBtn = document.getElementById('searchBtn');
const topologyBtn = document.getElementById('topologyBtn');
const toggleBtn = document.getElementById('toggleTopologyBtn');

searchBtn.addEventListener('click', () => visualizePacket(input.value));
input.addEventListener('keydown', (e) => {
  if (e.key === 'Enter') visualizePacket(input.value);
});
topologyBtn.addEventListener('click', visualizeTopology);
toggleBtn.addEventListener('click', toggleTopology);
