import random
import networkx as nx

def is_neighbor_on_right_orbit(orbit, neighbor_id, number_of_orbits):
    neighbor_orbit = int(neighbor_id.split("-")[1])
    orbit_on_right = (orbit + 1) % number_of_orbits
    return neighbor_orbit == orbit_on_right

def get_inter_orbit_pattern(id, neighbor_id):
    neighbor_id = int(neighbor_id.split("-")[2])
    return neighbor_id - id

def in_same_orbit(satellite_id, neighbor_id):
    satellite_orbit = int(satellite_id.split("-")[1])
    neighbor_orbit = int(neighbor_id.split("-")[1])
    return satellite_orbit == neighbor_orbit

def generate_random_intra_orbit_isls(graph, orbit, number_of_satellites_per_orbit, constellation_name, distance_graph, number_of_isls):
    for id in range(number_of_satellites_per_orbit):
        satellite_id = f"{constellation_name}-{orbit}-{id}"
        possible_neighbors = ([neighbor for neighbor in list(distance_graph.neighbors(satellite_id)) 
                      if in_same_orbit(satellite_id, neighbor) and graph.degree(neighbor) < number_of_isls and not graph.has_edge(satellite_id, neighbor)])
        
        while graph.degree(satellite_id) < number_of_isls and len(possible_neighbors) > 0:
            selected_neighbor = random.sample(possible_neighbors, 1)
            if not graph.has_edge(satellite_id, selected_neighbor[0]):
                graph.add_edge(satellite_id, selected_neighbor[0])
                possible_neighbors = [neighbor for neighbor in possible_neighbors if graph.degree(neighbor) < number_of_isls and not graph.has_edge(satellite_id, neighbor)]

def generate_random_inter_orbit_isls(graph, orbit, number_of_satellites_per_orbit, number_of_orbits, constellation_name, distance_graph, isls_between_orbits):
    base_satellite_id = f"{constellation_name}-{orbit}-0"
    neighbors = list(distance_graph.neighbors(base_satellite_id)) 
    neighbors_on_right = [neighbor for neighbor in neighbors if is_neighbor_on_right_orbit(orbit, neighbor, number_of_orbits)]
    selected_neighbors = random.sample(neighbors_on_right, isls_between_orbits // 2)
    inter_orbit_pattern = [get_inter_orbit_pattern(0, neighbor) for neighbor in selected_neighbors]
    for id in range(number_of_satellites_per_orbit):
        satellite_id = f"{constellation_name}-{orbit}-{id}"
        for pattern in inter_orbit_pattern:
            neighbor_id = f"{constellation_name}-{(orbit + 1) % number_of_orbits}-{(id + pattern) % number_of_satellites_per_orbit}"
            graph.add_edge(satellite_id, neighbor_id)

def generate_random_static_topology(satellite_nodes, num_orbits, num_satellites, constellation_name, consistent_distance_graph, num_isls):
    topology_graph = nx.Graph()
    topology_graph.add_nodes_from(satellite_nodes)

    # Generate inter-orbit ISLs
    for orbit in range(num_orbits):
        num_inter_orbit_isls = 2 * random.randint(1, num_isls // 2)
        generate_random_inter_orbit_isls(topology_graph, orbit, num_satellites, num_orbits, constellation_name, consistent_distance_graph, num_inter_orbit_isls)

    # Generate intra-orbit ISLs
    for orbit in range(num_orbits):
        generate_random_intra_orbit_isls(topology_graph, orbit, num_satellites, constellation_name, consistent_distance_graph, num_isls)

    return topology_graph