import csv
import time
import numpy as np
import networkx as nx

K = 10**8 # Field constant coefficient

def is_satellite(device_id, constellation_name):
    splitted_name = device_id.split("-")
    if len(splitted_name) == 3 and splitted_name[0] == constellation_name:
        return True
    return False

def get_cartesian_positions(cartesian_positions_file, constellation_name):
    cartesian_satellite_positions = {}
    cartesian_ground_station_positions = {}
    with open(f'./generated/{cartesian_positions_file}', 'r') as file:
        reader = csv.reader(file)
        next(reader)  # Skip header
        for row in reader:
            time_stamp = int(row[0])
            device_id = row[1]
            position = np.array([float(row[2]), float(row[3]), float(row[4])]) # (X, Y, Z)
            if is_satellite(device_id, constellation_name):
                if time_stamp not in cartesian_satellite_positions:
                    cartesian_satellite_positions[time_stamp] = {}
                cartesian_satellite_positions[time_stamp][device_id] = position
            else:
                if time_stamp not in cartesian_ground_station_positions:
                    cartesian_ground_station_positions[time_stamp] = {}
                cartesian_ground_station_positions[time_stamp][device_id] = position
    
    return cartesian_ground_station_positions, cartesian_satellite_positions

def get_flows_traffics(demand_matrix_file):
    flows_traffics = {}
    with open(f'./input/{demand_matrix_file}', 'r') as file:
        reader = csv.reader(file)
        next(reader)  # Skip header
        for row in reader:
            time_stamp = int(row[0])
            flow = (row[1], row[2], float(row[3])*2**20) # (source, destination, Traffic Length(Bytes))
            if time_stamp not in flows_traffics:
                flows_traffics[time_stamp] = []
            flows_traffics[time_stamp].append(flow)

    return flows_traffics

def scale_ground_stations_to_shell(ground_station_positions, shell_radius):
    scaled_positions = {}
    for device_id, position in ground_station_positions.items():
        r = np.linalg.norm(position)
        if r == 0:
            raise ValueError("Ground station position cannot be at the origin!")
        scaling_factor = shell_radius / r
        scaled_positions[device_id] = scaling_factor * position

    return scaled_positions

def calculate_tangent_vector(point_pos, geo_base_pos):
    perp_plane_vec = np.cross(geo_base_pos, point_pos)
    tangent_vec = np.cross(perp_plane_vec, point_pos)

    return tangent_vec / np.linalg.norm(tangent_vec)

def calculate_geodesic_distance(point_a, point_b):
    half_arc_line = np.linalg.norm(point_a - point_b) / 2
    radius = np.linalg.norm(point_a)
    arc_angle = 2 * np.arcsin(half_arc_line / radius)
    return radius * arc_angle

def calculate_field(position, flow_source_pos, flow_destination_pos, flow_strength):
    source_tangent_vector = calculate_tangent_vector(position, flow_source_pos)
    destination_tangent_vector = calculate_tangent_vector(position, flow_destination_pos)

    geodesic_distance_from_source = calculate_geodesic_distance(position, flow_source_pos)
    geodesic_distance_from_destination = calculate_geodesic_distance(position, flow_destination_pos)

    source_term = K * flow_strength * source_tangent_vector / (geodesic_distance_from_source ** 2)
    destination_term = K * flow_strength * destination_tangent_vector / (geodesic_distance_from_destination ** 2)

    return source_term - destination_term

def mirror_sat_to_base_plane(base_sat_pos, other_sat_pos):
    scaling_factor = (np.linalg.norm(base_sat_pos) ** 2) / np.dot(base_sat_pos, other_sat_pos)
    return scaling_factor * other_sat_pos

def calculate_riemannian_distance_of_flow(base_sat_pos, other_sat_pos, perp_field):
    mirrored_on_plane_other_sat_pos = mirror_sat_to_base_plane(base_sat_pos, other_sat_pos)
    inter_sat_vector = mirrored_on_plane_other_sat_pos - base_sat_pos

    return np.sqrt(np.abs(np.dot(inter_sat_vector, perp_field)))

def calculate_riemannian_distance(base_sat_pos, other_sat_pos, perp_fields):
    total_distance = 0.0
    for perp_field in perp_fields:
        riemannian_distance = calculate_riemannian_distance_of_flow(base_sat_pos, other_sat_pos, perp_field)
        total_distance += riemannian_distance

    return total_distance

def calculate_perp_field(sat_pos, field):
    perp_field_dir_unnorm = np.cross(field, sat_pos)
    perp_field_dir = perp_field_dir_unnorm / np.linalg.norm(perp_field_dir_unnorm)
    perp_field = np.linalg.norm(field) * perp_field_dir

    return perp_field

def calculate_per_flow_perp_field(base_sat_pos, ground_station_positions, traffic_flows):
    flows_fields_perp = []
    for source, dest, strength in traffic_flows:
        field = calculate_field(base_sat_pos, ground_station_positions[source], ground_station_positions[dest], strength)
        perp_field = calculate_perp_field(base_sat_pos, field)
        flows_fields_perp.append(perp_field)

    return flows_fields_perp

def calculate_fields_at_satellites(satellite_nodes, satellite_positions, ground_station_positions, source, dest, strength):
    fields = {
        "Satellite": [],
        "Field_X": [],
        "Field_Y": [],
        "Field_Z": [],
        "Field_Magnitude": []
    }
    
    shell_radius = np.linalg.norm(list(satellite_positions.values())[0])
    scaled_ground_station_positions = scale_ground_stations_to_shell(ground_station_positions, shell_radius)

    for sat in satellite_nodes:
        field = calculate_field(satellite_positions[sat], scaled_ground_station_positions[source], scaled_ground_station_positions[dest], strength)
        field_magnitude = np.linalg.norm(field)  # Normalize the field vector
        field = field / field_magnitude
        fields["Satellite"].append(sat)
        fields["Field_X"].append(field[0])
        fields["Field_Y"].append(field[1])
        fields["Field_Z"].append(field[2])
        fields["Field_Magnitude"].append(field_magnitude)

    #Normalized Magnitude
    max_mag = max(fields["Field_Magnitude"])
    fields["Field_Magnitude"] = [mag / max_mag for mag in fields["Field_Magnitude"]]

    return fields

def choose_perpendicular_neighbor(base_sat, chosen_sat, sat_positions, neighbors):
    base_sat_pos = sat_positions[base_sat]
    chosen_sat_on_plane_vec = mirror_sat_to_base_plane(base_sat_pos, sat_positions[chosen_sat])
    base_to_chosen_vec = chosen_sat_on_plane_vec - base_sat_pos
    neighbor_perp_score = []
    for neighbor in neighbors:
        neighbor_on_plane_vec = mirror_sat_to_base_plane(base_sat_pos, sat_positions[neighbor])
        base_to_neighbor_vec = neighbor_on_plane_vec - base_sat_pos
        if np.dot(np.cross(base_to_chosen_vec, base_to_neighbor_vec), base_sat_pos) > 0:
            perp_score = np.abs(np.dot(base_to_chosen_vec, base_to_neighbor_vec)) / (np.linalg.norm(base_to_chosen_vec) * np.linalg.norm(base_to_neighbor_vec))
            neighbor_perp_score.append((neighbor, perp_score))
    
    return min(neighbor_perp_score, key=lambda x: x[1])[0]

def is_edge_valid(base_sat, other_sat, num_isl, topology_graph):
    if other_sat is None:
        return False
    if topology_graph.has_edge(base_sat, other_sat):
        return False
    if topology_graph.in_degree(other_sat) >= num_isl // 2:
        return False
    return True

def is_edge_valid_undirected(base_sat, other_sat, num_isl, topology_graph):
    if other_sat is None:
        return False
    if topology_graph.has_edge(base_sat, other_sat):
        return False
    if topology_graph.degree(other_sat) >= num_isl:
        return False
    return True

def find_next_closest(base_sat, closest_satellites, closest_perp_satellites, closest_counter, closest_perp_counter):
    if closest_counter <= closest_perp_counter:
        return closest_satellites[base_sat][closest_counter], closest_counter + 1, closest_perp_counter
    else:
        return closest_perp_satellites[base_sat][closest_perp_counter], closest_counter, closest_perp_counter + 1

def choose_next_edge(base_sat, closest_satellites, closest_perp_satellites, num_isl, topology_graph, closest_counter, closest_perp_counter):
    next_sat = None
    while not is_edge_valid(base_sat, next_sat, num_isl, topology_graph) and closest_perp_counter < len(closest_perp_satellites[base_sat]):
        next_sat, closest_counter, closest_perp_counter = find_next_closest(base_sat, closest_satellites, closest_perp_satellites, closest_counter, closest_perp_counter)

    if not is_edge_valid(base_sat, next_sat, num_isl, topology_graph):
        return None, closest_counter, closest_perp_counter

    return next_sat, closest_counter, closest_perp_counter

def choose_next_edge_undirected(base_sat, closest_satellites, closest_perp_satellites, num_isl, topology_graph, closest_counter, closest_perp_counter):
    next_sat = None
    while not is_edge_valid_undirected(base_sat, next_sat, num_isl, topology_graph) and closest_perp_counter < len(closest_perp_satellites[base_sat]):
        next_sat, closest_counter, closest_perp_counter = find_next_closest(base_sat, closest_satellites, closest_perp_satellites, closest_counter, closest_perp_counter)

    if not is_edge_valid_undirected(base_sat, next_sat, num_isl, topology_graph):
        return None, closest_counter, closest_perp_counter

    return next_sat, closest_counter, closest_perp_counter

def calculate_distances_riemannian_satellites(satellite_nodes, satellite_positions, ground_station_positions, traffic_flow, consistent_distance_graph):
    shell_radius = np.linalg.norm(list(satellite_positions.values())[0])
    scaled_ground_station_positions = scale_ground_stations_to_shell(ground_station_positions, shell_radius)
    distances = {}
    
    for base_sat in satellite_nodes:
        distance_dict = {}
        base_perp_fields = calculate_per_flow_perp_field(satellite_positions[base_sat], scaled_ground_station_positions, traffic_flow)
        for other_sat in consistent_distance_graph[base_sat]:
            distance = calculate_riemannian_distance(satellite_positions[base_sat], satellite_positions[other_sat], base_perp_fields)
            perp_sat = choose_perpendicular_neighbor(base_sat, other_sat, satellite_positions, consistent_distance_graph[base_sat])
            distance_dict[other_sat] = (perp_sat, distance)
        distances[base_sat] = distance_dict

    print("calculation of closest riemannian satellites completed.")

    return distances

def get_closest_distances(distances):
    closest_satellites = {}
    closest_perp_satellite = {}
    for sat, dist_dict in distances.items():
        dist_list = [(sat, perp_sat, dist) for sat, (perp_sat, dist) in dist_dict.items()]
        dist_list.sort(key=lambda x: x[2])
        closest_satellites[sat] = [sat for sat, _, _ in dist_list]
        closest_perp_satellite[sat] = [perp_sat for _, perp_sat, _ in dist_list]

    return closest_satellites, closest_perp_satellite

def generate_riemannian_dynamic_topology(
        satellite_nodes, consistent_distance_graph, satellite_positions, 
        ground_station_positions, traffic_flow, num_isls
    ):

    topology_graph = nx.DiGraph()
    topology_graph.add_nodes_from(satellite_nodes)
    
    # Calculate Riemannian distances for each satellite, and their corresponding perpendicular satellites
    distances = calculate_distances_riemannian_satellites(
        satellite_nodes, satellite_positions, ground_station_positions, traffic_flow, consistent_distance_graph
    )
    closest_satellites, closest_perp_satellite = get_closest_distances(distances)

    # First pass to add external edges based on Riemannian distance
    closest_counters = {}
    for base_sat in satellite_nodes:
        closest_counter = 0
        closest_perp_counter = 0
        for _ in range(num_isls // 2):
            chosen_sat, closest_counter, closest_perp_counter = choose_next_edge(
                base_sat, closest_satellites, closest_perp_satellite, num_isls, topology_graph, closest_counter, closest_perp_counter
            )
            if chosen_sat is None:
                break
            topology_graph.add_edge(base_sat, chosen_sat)
        closest_counters[base_sat] = (closest_counter, closest_perp_counter)

    topology_graph = topology_graph.to_undirected()

    print("First pass of edge assignment completed. Starting second pass for full degree utilization.")

    # Second pass to ensure full degree utilization
    for node in topology_graph.nodes():
        node_degree = topology_graph.degree(node)
        closest_counter, closest_perp_counter = closest_counters[node]
        for _ in range(num_isls - node_degree):
            chosen_sat, closest_counter, closest_perp_counter = choose_next_edge_undirected(
                node, closest_satellites, closest_perp_satellite, num_isls, topology_graph, closest_counter, closest_perp_counter
            )
            if chosen_sat is None:
                break
            topology_graph.add_edge(node, chosen_sat)

    return topology_graph

def get_pattern(origin_id, origin_orbit, neighbor_full_id):
    neighbor_orbit = int(neighbor_full_id.split("-")[1])
    neighbor_id = int(neighbor_full_id.split("-")[2])
    return (neighbor_orbit - origin_orbit, neighbor_id - origin_id)

def apply_pattern(full_id, pattern, constellation_name, number_of_orbits, number_of_sats):
    base_orbit = int(full_id.split("-")[1])
    base_id = int(full_id.split("-")[2])
    new_orbit = (base_orbit + pattern[0]) % number_of_orbits
    new_id = (base_id + pattern[1]) % number_of_sats
    return f"{constellation_name}-{new_orbit}-{new_id}"

def find_best_patterns_of_orbit(orbit, base_satellite_id, number_of_sats, distances, constellation_name, number_of_orbits):
    pattern_scores = []
    neighbors_distances = distances[base_satellite_id]
    for other_sat, (perp_sat, _) in neighbors_distances.items():
        pattern_acc_flag = True
        pattern = get_pattern(0, orbit, other_sat)
        perp_pattern = get_pattern(0, orbit, perp_sat)
        pattern_avg_score = 0.0
        for sat_id in range(number_of_sats):
            sat_full_id = f"{constellation_name}-{orbit}-{sat_id}"
            pattern_sat = apply_pattern(sat_full_id, pattern, constellation_name, number_of_orbits, number_of_sats)
            if pattern_sat not in distances[sat_full_id]:
                pattern_acc_flag = False
                break
            distance = distances[sat_full_id][pattern_sat][1]
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
    
    # Possibly needs a second pass to ensure full degree utilization
        
    return topology_graph

