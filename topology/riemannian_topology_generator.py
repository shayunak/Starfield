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
        "Field_Z": []
    }
    
    shell_radius = np.linalg.norm(list(satellite_positions.values())[0])
    scaled_ground_station_positions = scale_ground_stations_to_shell(ground_station_positions, shell_radius)

    for sat in satellite_nodes:
        field = calculate_field(satellite_positions[sat], scaled_ground_station_positions[source], scaled_ground_station_positions[dest], strength)
        field = field / np.linalg.norm(field)  # Normalize the field vector
        fields["Satellite"].append(sat)
        fields["Field_X"].append(field[0])
        fields["Field_Y"].append(field[1])
        fields["Field_Z"].append(field[2])

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


def generate_riemannian_dynamic_topology(
        satellite_nodes, consistent_distance_graph, satellite_positions, 
        ground_station_positions, traffic_flow, num_isls
    ):

    topology_graph = nx.Graph()
    topology_graph.add_nodes_from(satellite_nodes)
    shell_radius = np.linalg.norm(list(satellite_positions.values())[0])
    scaled_ground_station_positions = scale_ground_stations_to_shell(ground_station_positions, shell_radius)

    closest_satellites = {}
    for base_sat in satellite_nodes:
        distance_list = []
        time_start = time.time()
        base_perp_fields = calculate_per_flow_perp_field(satellite_positions[base_sat], scaled_ground_station_positions, traffic_flow)
        for other_sat in consistent_distance_graph[base_sat]:
            distance = calculate_riemannian_distance(satellite_positions[base_sat], satellite_positions[other_sat], base_perp_fields)
            distance_list.append((other_sat, distance))
        distance_list.sort(key=lambda x: x[1])
        closest_satellites[base_sat] = [sat for sat, _ in distance_list]
        print(f"Base: {base_sat}, Closest: {closest_satellites[base_sat][0]}, Time taken: {time.time() - time_start:.2f} seconds")

    num_isl_perp = num_isls // 4 
    for i in range(num_isl_perp):
        for base_sat in satellite_nodes:
            neighbors = consistent_distance_graph[base_sat]
            chosen_sat = closest_satellites[base_sat][i]
            perp_neighbor = choose_perpendicular_neighbor(base_sat, chosen_sat, satellite_positions, neighbors)
            topology_graph.add_edge(base_sat, chosen_sat)
            topology_graph.add_edge(base_sat, perp_neighbor)

    return topology_graph