import networkx as nx
from riemann_fields_metrics import calculate_distances_riemannian_satellites

def is_edge_valid(base_sat, other_sat, num_isl, topology_graph):
    if other_sat is None:
        return False
    if topology_graph.has_edge(base_sat, other_sat):
        return False
    if topology_graph.in_degree(other_sat) >= num_isl // 2:
        return False
    return True

def get_pattern(origin_id, origin_orbit, neighbor_full_id):
    neighbor_orbit = int(neighbor_full_id.split("-")[1])
    neighbor_id = int(neighbor_full_id.split("-")[2])
    return (neighbor_orbit - origin_orbit, neighbor_id - origin_id)

def apply_pattern(full_id, pattern, constellation_name, number_of_orbits, number_of_sats):
    base_orbit = int(full_id.split("-")[1])
    base_id = int(full_id.split("-")[2])
    orbit_pattern, id_pattern = pattern
    new_orbit = (base_orbit + orbit_pattern) % number_of_orbits
    new_id = (base_id + id_pattern) % number_of_sats
    return f"{constellation_name}-{new_orbit}-{new_id}"

def find_best_patterns_of_orbit(orbit, base_satellite_id, number_of_sats, distances, constellation_name, number_of_orbits):
    pattern_scores = []
    neighbors_distances = distances[base_satellite_id]
    for other_sat, (perp_sat, distance) in neighbors_distances.items():
        pattern_acc_flag = True
        pattern = get_pattern(0, orbit, other_sat)
        perp_pattern = get_pattern(0, orbit, perp_sat)
        pattern_avg_score = 0
        for sat_id in range(number_of_sats):
            sat_full_id = f"{constellation_name}-{orbit}-{sat_id}"
            pattern_sat = apply_pattern(sat_full_id, pattern, constellation_name, number_of_orbits, number_of_sats)
            perp_pattern_sat = apply_pattern(sat_full_id, perp_pattern, constellation_name, number_of_orbits, number_of_sats)
            if pattern_sat not in distances[sat_full_id] or perp_pattern_sat not in distances[sat_full_id]:
                pattern_acc_flag = False
                break
            _, distance = distances[sat_full_id][pattern_sat]
            pattern_avg_score = (pattern_avg_score*sat_id + distance) / (sat_id + 1)
        if pattern_acc_flag:
            pattern_scores.append((pattern, perp_pattern, pattern_avg_score))
    
    pattern_scores.sort(key=lambda x: x[2])
    closest_patterns = [pattern for pattern, _, _ in pattern_scores]
    closest_perp_patterns = [perp_pattern for _, perp_pattern, _ in pattern_scores]

    return closest_patterns, closest_perp_patterns

def connect_pattern(pattern, orbit, topology_graph, constellation_name, number_of_sats, number_of_orbits):
    for sat_id in range(number_of_sats):
        sat_full_id = f"{constellation_name}-{orbit}-{sat_id}"
        pattern_sat = apply_pattern(sat_full_id, pattern, constellation_name, number_of_orbits, number_of_sats)
        topology_graph.add_edge(sat_full_id, pattern_sat)

def is_pattern_valid(orbit, pattern, topology_graph, constellation_name, number_of_sats, number_of_orbits, num_isl):
    if pattern is None:
        return False
    for sat_id in range(number_of_sats):
        sat_full_id = f"{constellation_name}-{orbit}-{sat_id}"
        pattern_sat = apply_pattern(sat_full_id, pattern, constellation_name, number_of_orbits, number_of_sats)
        if not is_edge_valid(sat_full_id, pattern_sat, num_isl, topology_graph):
            return False
    
    return True

def find_next_closest_pattern(closest_satellites, closest_perp_satellites, closest_counter, closest_perp_counter):
    if closest_counter <= closest_perp_counter:
        return closest_satellites[closest_counter], closest_counter + 1, closest_perp_counter
    else:
        return closest_perp_satellites[closest_perp_counter], closest_counter, closest_perp_counter + 1

def choose_next_pattern(orbit, closest_patterns, closest_perp_patterns, num_isls, topology_graph, closest_counter, closest_perp_counter, constellation_name, number_of_sats, number_of_orbits):
    next_pattern = None
    while not is_pattern_valid(orbit, next_pattern, topology_graph, constellation_name, number_of_sats, number_of_orbits, num_isls) and closest_perp_counter < len(closest_perp_patterns):
        next_pattern, closest_counter, closest_perp_counter = find_next_closest_pattern(closest_patterns, closest_perp_patterns, closest_counter, closest_perp_counter)
    
    if not is_pattern_valid(orbit, next_pattern, topology_graph, constellation_name, number_of_sats, number_of_orbits, num_isls):
        return None, closest_counter, closest_perp_counter
    
    return next_pattern, closest_counter, closest_perp_counter


def generate_riemannian_static_topology(
        satellite_nodes, number_of_orbits, number_of_sats, constellation_name, consistent_distance_graph, satellite_positions, 
        ground_station_positions, traffic_flow, num_isls
    ):
    topology_graph = nx.DiGraph()
    topology_graph.add_nodes_from(satellite_nodes)

    # Calculate riemannian distances for each pair of satellites
    distances = calculate_distances_riemannian_satellites(
        satellite_nodes, satellite_positions, ground_station_positions, traffic_flow, consistent_distance_graph
    )

    for orbit in range(number_of_orbits):
        base_satellite_id = f"{constellation_name}-{orbit}-0"
        closest_patterns, closest_perp_patterns = find_best_patterns_of_orbit(
            orbit, base_satellite_id, number_of_sats, distances, constellation_name, number_of_orbits
        )
        closest_counter = 0
        closest_perp_counter = 0
        for _ in range(num_isls // 2):
            chosen_pattern, closest_counter, closest_perp_counter = choose_next_pattern(
                orbit, closest_patterns, closest_perp_patterns, num_isls, topology_graph, closest_counter, 
                closest_perp_counter, constellation_name, number_of_sats, number_of_orbits
            )
            if chosen_pattern is None:
                break
            connect_pattern(chosen_pattern, orbit, topology_graph, constellation_name, number_of_sats, number_of_orbits)

    topology_graph = topology_graph.to_undirected()    

    return topology_graph