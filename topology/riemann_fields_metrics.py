import csv
import numpy as np

K = 5*10**6 # Field constant coefficient

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
    total_flows = {}
    with open(f'./input/{demand_matrix_file}', 'r') as file:
        reader = csv.reader(file)
        next(reader)  # Skip header
        for row in reader:
            time_stamp = int(row[0])
            flow = (row[1], row[2], float(row[3])*2**20) # (source, destination, Traffic Length(Bytes))
            if (row[1], row[2]) not in total_flows:
                total_flows[(row[1], row[2])] = 0
            if time_stamp not in flows_traffics:
                flows_traffics[time_stamp] = []
            total_flows[(row[1], row[2])] += float(row[3])
            flows_traffics[time_stamp].append(flow)

    avg_flows = []
    total_time = len(flows_traffics)
    for (source, dest), total_traffic in total_flows.items():
        avg_traffic = (total_traffic / total_time) * 2**20
        avg_flows.append((source, dest, avg_traffic))

    return flows_traffics, avg_flows

def avg_flow_traffics(flows_traffics, time_interval, time_period):
    sorted_flows = {}
    for time in flows_traffics:
        if time < time_period:
            time_stamp = int(time / time_interval) * time_interval
            sorted_flows.setdefault(time_stamp, []).append(flows_traffics[time])

    avg_flows = {}
    for time_stamp, flows_list in sorted_flows.items():
        flow_sums = {}
        for flows in flows_list:
            for source, dest, strength in flows:
                if (source, dest) not in flow_sums:
                    flow_sums[(source, dest)] = 0
                flow_sums[(source, dest)] += strength
        avg_flows[time_stamp] = [(source, dest, flow_sums[(source, dest)] / len(flows_list)) for (source, dest) in flow_sums]

    return avg_flows

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

    return fields

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

    source_term = K * flow_strength * destination_tangent_vector / (geodesic_distance_from_source ** 2)
    destination_term = K * flow_strength * source_tangent_vector / (geodesic_distance_from_destination ** 2)

    return destination_term - source_term

def mirror_sat_to_base_plane(base_sat_pos, other_sat_pos):
    scaling_factor = (np.linalg.norm(base_sat_pos) ** 2) / np.dot(base_sat_pos, other_sat_pos)
    return scaling_factor * other_sat_pos

def calculate_riemannian_distance_of_flow(base_sat_pos, other_sat_pos, perp_field):
    mirrored_on_plane_other_sat_pos = mirror_sat_to_base_plane(base_sat_pos, other_sat_pos)
    inter_sat_vector = mirrored_on_plane_other_sat_pos - base_sat_pos
    inter_sat_vector_norm = np.linalg.norm(inter_sat_vector)
    field_norm = np.linalg.norm(perp_field)
    inter_sat_hop_stretch_factor = 2.0 * np.exp(-field_norm)
    directional_component = np.abs(np.dot(inter_sat_vector, perp_field))

    return directional_component / (inter_sat_vector_norm ** inter_sat_hop_stretch_factor)

def calculate_riemannian_distance(base_sat_pos, other_sat_pos, perp_fields):
    total_distance = 0.0
    #dist = {
    #    "dir_comp": [],
    #    "field_norm": [],
    #    "inter_sat_vector_norm": [],
    #    "riemannian_distance": []
    #}
    for perp_field in perp_fields:
        riemannian_distance = calculate_riemannian_distance_of_flow(base_sat_pos, other_sat_pos, perp_field)
        #dist["dir_comp"].append(dir_comp)
        #dist["field_norm"].append(field_norm)
        #ist["inter_sat_vector_norm"].append(inter_sat_vector_norm)
        #dist["riemannian_distance"].append(riemannian_distance)
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

    #for sat, dist_dict in distances.items():
    #    for other_sat, (perp_sat, dist) in dist_dict.items():
    #        print(f"distance params from {sat} to {other_sat}: ")
    #        for i in range(len(dist["riemannian_distance"])):
    #            print(f"  Flow {i+1}: Dir Comp={dist['dir_comp'][i]:.4f}, Field Norm={dist['field_norm'][i]:.4f}, Inter-Sat Vector Norm={dist['inter_sat_vector_norm'][i]:.4f}, Riemannian Distance={dist['riemannian_distance'][i]:.4f}")

    print("calculation of closest riemannian satellites completed.")

    return distances