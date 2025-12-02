import csv, torch
import random
import numpy as np

K = 10**7 # Field constant coefficient

def _ensure_device(x, device):
    t = torch.as_tensor(x, dtype=torch.float32, device=device)
    return t

def is_satellite(device_id, constellation_name):
    splitted_name = device_id.split("-")
    if len(splitted_name) == 3 and splitted_name[0] == constellation_name:
        return True
    return False

def get_cartesian_positions(cartesian_positions_file, constellation_name, time_period, device=torch.device('cpu')):
    cartesian_satellite_positions = {}
    cartesian_ground_station_positions = {}
    end_time_stamp = time_period * 1000
    with open(f'./generated/{cartesian_positions_file}', 'r') as file:
        reader = csv.reader(file)
        next(reader)  # Skip header
        for row in reader:
            time_stamp = int(row[0])
            if time_stamp > end_time_stamp:
                break
            device_id = row[1]
            position = _ensure_device([float(row[2]), float(row[3]), float(row[4])], device) # (X, Y, Z)
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
    max_time = int(demand_matrix_file[:-4].split("#")[-1][:-1])
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
    for (source, dest), total_traffic in total_flows.items():
        avg_traffic = (total_traffic / max_time) * 2**20
        avg_flows.append((source, dest, avg_traffic))

    return flows_traffics, avg_flows

def avg_flow_traffics(flows_traffics, time_interval, time_period):
    sorted_flows = {}
    for time in flows_traffics:
        if time < time_period:
            time_stamp = int(time / time_interval) * time_interval
            sorted_flows.setdefault(time_stamp, []).append(flows_traffics[time])

    avg_flows = {}
    time_interval_s = time_interval / 1000
    for time_stamp, flows_list in sorted_flows.items():
        flow_sums = {}
        for flows in flows_list:
            for source, dest, strength in flows:
                if (source, dest) not in flow_sums:
                    flow_sums[(source, dest)] = 0
                flow_sums[(source, dest)] += strength
        avg_flows[time_stamp] = [(source, dest, flow_sums[(source, dest)] / time_interval_s) for (source, dest) in flow_sums]

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
    for key, pos in ground_station_positions.items():
        r = torch.linalg.norm(pos)
        scaled_positions[key] = pos * (shell_radius / r)
    return scaled_positions


def mirror_sat_to_base_plane(base_pos, other_pos):
    base_norm2 = (base_pos * base_pos).sum()
    dot = (base_pos * other_pos).sum()
    factor = base_norm2 / dot
    return factor * other_pos


def calculate_tangent_vector(point_pos, geo_pos):
    perp_plane = torch.cross(geo_pos, point_pos, dim=0)
    tangent = torch.cross(perp_plane, point_pos, dim=0)
    return tangent / torch.linalg.norm(tangent)


def calculate_geodesic_distance(point_a, point_b):
    half_line = torch.linalg.norm(point_a - point_b) / 2
    R = torch.linalg.norm(point_a)
    x = (half_line / R)
    arc = 2 * torch.asin(x)
    return R * arc


def calculate_field(pos, src, dst, strength, K):
    t_src = calculate_tangent_vector(pos, src)
    t_dst = calculate_tangent_vector(pos, dst)

    d_src = calculate_geodesic_distance(pos, src)
    d_dst = calculate_geodesic_distance(pos, dst)

    term_src = K * strength * t_dst / (d_src * d_src)
    term_dst = K * strength * t_src / (d_dst * d_dst)
    return term_dst - term_src


def calculate_perp_field(pos, field):
    perp_dir = torch.cross(field, pos, dim=0)
    perp_dir = perp_dir / torch.linalg.norm(perp_dir)
    return torch.linalg.norm(field) * perp_dir

def calculate_per_flow_perp_field(base_pos, gs_positions, flows, K=1.0):
    """Vectorized across flows."""
    perp_list = []
    for src, dst, strength in flows:
        f = calculate_field(base_pos, gs_positions[src], gs_positions[dst], strength, K=K)
        perp = calculate_perp_field(base_pos, f)
        perp_list.append(perp)
    return torch.stack(perp_list) if perp_list else torch.zeros((0,3), device=base_pos.device)

def calculate_riemannian_distances(base_pos, other_pos, perp_fields):
    # mirror on tangent plane
    base_norm2 = (base_pos * base_pos).sum()
    dot_base_other = (base_pos * other_pos).sum()
    factor = base_norm2 / dot_base_other
    mirrored = factor * other_pos

    inter_vec = mirrored - base_pos
    inter_vec_norm = torch.linalg.norm(inter_vec)
    
    field_norms = torch.linalg.norm(perp_fields)

    inter_hop_stretch_factors = 2.0 * torch.exp(-field_norms)
    directional_components = torch.abs((inter_vec * perp_fields).sum(dim=1))
    distance_priorities = inter_vec_norm ** inter_hop_stretch_factors

    distances = directional_components / distance_priorities

    return float(distances.sum().item())

def choose_perpendicular_neighbor(base_id, chosen_id, sat_positions, neighbors):
    base_pos = sat_positions[base_id]
    chosen_pos = sat_positions[chosen_id]

    chosen_plane = mirror_sat_to_base_plane(base_pos, chosen_pos)
    base_to_chosen = chosen_plane - base_pos
    base_to_chosen_norm = torch.linalg.norm(base_to_chosen)

    # Pack neighbors into tensor
    nb_pos = torch.stack([sat_positions[n] for n in neighbors])  # (N,3)
    nb_plane = torch.stack([mirror_sat_to_base_plane(base_pos, x) for x in nb_pos])  # (N,3)
    base_to_nb = nb_plane - base_pos

    # orientation test
    cross_prod = torch.cross(base_to_chosen.expand_as(base_to_nb), base_to_nb, dim=1)
    orient = (cross_prod * base_pos).sum(dim=1)
    mask = orient > 0

    if not mask.any():
        return random.choice(list(neighbors))

    valid_nb = base_to_nb[mask]
    valid_ids = [n for i, n in enumerate(neighbors) if mask[i].item()]

    dots = torch.abs((valid_nb * base_to_chosen).sum(dim=1))
    norms = (base_to_chosen_norm * torch.linalg.norm(valid_nb, dim=1))

    scores = dots / norms
    best_idx = torch.argmin(scores).item()

    return valid_ids[best_idx]

def calculate_distances_riemannian_satellites(
    satellite_nodes,
    sat_positions,
    gs_positions,
    traffic_flows,
    neighbor_graph
):
    shell_radius = torch.linalg.norm(next(iter(sat_positions.values()))).item()
    gs_scaled = scale_ground_stations_to_shell(gs_positions, shell_radius)

    distances = {}

    for base in satellite_nodes:
        base_pos = sat_positions[base]
        perp_fields = calculate_per_flow_perp_field(base_pos, gs_scaled, traffic_flows, K=K)

        dist_dict = {}
        for other in neighbor_graph[base]:
            other_pos = sat_positions[other]

            d = calculate_riemannian_distances(base_pos, other_pos, perp_fields)
            perp_nb = choose_perpendicular_neighbor(base, other, sat_positions, neighbor_graph[base])

            dist_dict[other] = (perp_nb, d)

        distances[base] = dist_dict
    
    return distances