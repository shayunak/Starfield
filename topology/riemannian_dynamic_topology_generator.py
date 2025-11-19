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

