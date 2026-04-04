from geopy.distance import geodesic
import networkx as nx
from itertools import combinations

def is_ground_station(node_id, constellation_name):
    splitted_id = node_id.split("-")
    if (len(splitted_id) == 3) and (splitted_id[0] == constellation_name):
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

def pairwise_gs_distances(ground_station_positions, ground_stations):
    pairwise_distances = {}
    for first_gs in ground_stations:
        for second_gs in ground_stations:
            if first_gs == second_gs:
                continue
            pos1, pos2 = ground_station_positions[first_gs], ground_station_positions[second_gs]
            distance = geodesic(pos1, pos2).meters
            if first_gs not in pairwise_distances:
                pairwise_distances[first_gs] = {}
            pairwise_distances[first_gs][second_gs] = distance

    return pairwise_distances

def evaluate_motif_topology(topology_graph, pairwise_gs_distances, ground_stations, alpha):
    hop_counts = 0
    stretch_factors = 0
    count = 0

    gs_subset = set(ground_stations)

    for u in gs_subset:
        lengths, paths = nx.single_source_dijkstra(topology_graph, u, weight='weight')

        for v in gs_subset:
            if u < v and v in lengths:  # avoid duplicates
                hop_counts += len(paths[v]) - 1
                stretch_factors += lengths[v] / pairwise_gs_distances[u][v]
                count += 1

    avg_hop_count = hop_counts / count if count > 0 else 0
    avg_stretch_factor = stretch_factors / count if count > 0 else 0

    return avg_stretch_factor*alpha + avg_hop_count

def is_pattern_valid(consistent_distance_graph, pattern, constellation_name, number_of_sats, number_of_orbits):
    for node in consistent_distance_graph:
        neighbor = apply_pattern(node, pattern, constellation_name, number_of_orbits, number_of_sats)
        if neighbor not in consistent_distance_graph[node] or node not in consistent_distance_graph[neighbor]:
            return False
    return True

def is_pattern_symmetric(first_pattern, second_pattern, number_of_orbits, number_of_sats):
    orbit_pattern1, id_pattern1 = first_pattern
    orbit_pattern2, id_pattern2 = second_pattern

    return (orbit_pattern1 + orbit_pattern2) % number_of_orbits == 0 and (id_pattern1 + id_pattern2) % number_of_sats == 0

def find_symmetric_pattern(patterns, base_pattern, number_of_orbits, number_of_sats):
    for pattern in patterns:
        if is_pattern_symmetric(pattern, base_pattern, number_of_orbits, number_of_sats):
            return True
        
    return False

def symmetric_pattern_index(base_pattern, patterns, number_of_orbits, number_of_sats):
    for i, pattern in enumerate(patterns):
        if is_pattern_symmetric(pattern, base_pattern, number_of_orbits, number_of_sats):
            return i
    return None

def remove_symmetric_patterns(patterns, number_of_orbits, number_of_sats):
    for pattern in patterns:
        indx = symmetric_pattern_index(pattern, patterns, number_of_orbits, number_of_sats)
        patterns.pop(indx)

    return patterns

def get_valid_patterns(consistent_distance_graph, constellation_name, number_of_sats, number_of_orbits):
    base_satellite = f"{constellation_name}-0-0"
    neighbors = consistent_distance_graph[base_satellite]
    patterns = [get_pattern(0, 0, neighbor) for neighbor in neighbors]
    valid_patterns = []
    for pattern in patterns:
        if find_symmetric_pattern(patterns, pattern, number_of_orbits, number_of_sats) and is_pattern_valid(consistent_distance_graph, pattern, constellation_name, number_of_sats, number_of_orbits):
            valid_patterns.append(pattern)

    return remove_symmetric_patterns(valid_patterns, number_of_orbits, number_of_sats)

def apply_motif(satellite_nodes, motif, constellation_name, number_of_orbits, number_of_sats):
    edge_list = []
    for node in satellite_nodes:
        for pattern in motif:
            neighbor = apply_pattern(node, pattern, constellation_name, number_of_orbits, number_of_sats)
            edge_list.append((node, neighbor))
            edge_list.append((neighbor, node))
    return set(edge_list)

def get_motif_graph_from_edges(base_graph, motif_edges, base_graph_isls):
    motif_graph = base_graph.copy()
    edges_to_remove = base_graph_isls - motif_edges
    motif_graph.remove_edges_from(edges_to_remove)
    return motif_graph

def generate_topology_graph(motif, satellite_nodes, num_orbits, num_satellites, constellation_name):
    G = nx.Graph()
    G.add_nodes_from(satellite_nodes)
    motif_edges = apply_motif(satellite_nodes, motif, constellation_name, num_orbits, num_satellites)
    G.add_edges_from(motif_edges)
    return G

def generate_motif_topology(base_graph, satellite_nodes, num_orbits, num_satellites, constellation_name, consistent_distance_graph, ground_stations, ground_station_positions, alpha, num_isls):
    num_motif_patterns = num_isls // 2
    gs_distances = pairwise_gs_distances(ground_station_positions, ground_stations)
    valid_patterns = get_valid_patterns(consistent_distance_graph, constellation_name, num_satellites, num_orbits)
    motifs = list(combinations(valid_patterns, num_motif_patterns))
    base_graph_isls = set([(u, v) for u, v in base_graph.edges() if not is_ground_station(str(u), constellation_name) and not is_ground_station(str(v), constellation_name)])
    best_motif = None
    best_score = float('inf')
    for motif in motifs:
        motif_edges = apply_motif(satellite_nodes, motif, constellation_name, num_orbits, num_satellites)
        motif_graph = get_motif_graph_from_edges(base_graph, motif_edges, base_graph_isls)
        score = evaluate_motif_topology(motif_graph, gs_distances, ground_stations, alpha)
        if score < best_score:
            best_score = score
            best_motif = motif

    return generate_topology_graph(best_motif, satellite_nodes, num_orbits, num_satellites, constellation_name)