import csv
from datetime import datetime
import sys
from matplotlib import pyplot as plt
import networkx as nx
import random
import consistent_distance_graph_generator as cdg

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

def generate_intra_orbit_isls(graph, orbit, number_of_satellites_per_orbit, constellation_name, distance_graph, number_of_isls):
    for id in range(number_of_satellites_per_orbit):
        satellite_id = f"{constellation_name}-{orbit}-{id}"
        possible_neighbors = ([neighbor for neighbor in list(distance_graph.neighbors(satellite_id)) 
                      if in_same_orbit(satellite_id, neighbor) and  graph.degree(neighbor) < number_of_isls])
        
        while graph.degree(satellite_id) < number_of_isls and len(possible_neighbors) > 0:
            selected_neighbor = random.sample(possible_neighbors, 1)
            if not graph.has_edge(satellite_id, selected_neighbor[0]):
                graph.add_edge(satellite_id, selected_neighbor[0])
                possible_neighbors = [neighbor for neighbor in possible_neighbors if graph.degree(neighbor) < number_of_isls]

def generate_inter_orbit_isls(graph, orbit, number_of_satellites_per_orbit, number_of_orbits, constellation_name, distance_graph, isls_between_orbits):
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

def generate_random_topology(distance_file, num_isls):
    is_consistent_graph, df_graph, constellation_name, time_step, total_time, simulation_details, nodes, num_orbits, num_satellites = cdg.read_distance_file(distance_file)
    consistent_distance_graph, satellite_nodes = df_graph, nodes
    if not is_consistent_graph:
        consistent_distance_graph, satellite_nodes = cdg.get_consistent_distance_graph(df_graph, distance_file, nodes, constellation_name, time_step, total_time)
    
    topology_graph = nx.Graph()
    topology_graph.add_nodes_from(satellite_nodes)

    # Generate inter-orbit ISLs
    for orbit in range(num_orbits):
        num_inter_orbit_isls = 2 * random.randint(1, num_isls // 2)
        generate_inter_orbit_isls(topology_graph, orbit, num_satellites, num_orbits, constellation_name, consistent_distance_graph, num_inter_orbit_isls)

    # Generate intra-orbit ISLs
    for orbit in range(num_orbits):
        generate_intra_orbit_isls(topology_graph, orbit, num_satellites, constellation_name, consistent_distance_graph, num_isls)

    filename = f"RandomStaticTopology#{datetime.today().strftime('%Y_%m_%d,%H_%M_%S')}#{simulation_details}.csv"
    save_topology_to_file(topology_graph, satellite_nodes, f'./input/{filename}')

def save_topology_to_file(graph, nodes, filename):
    with open(filename, 'w', newline='') as file:
        writer = csv.writer(file)
        writer.writerow(['FirstSatellite', 'SecondSatellite'])
        for node in nodes:
            for neighbor in graph.neighbors(node):
                writer.writerow([node, neighbor])

    print(f"Topology saved to {filename}")

def printHelp():
    print("topology_generator.py --help")
    print("topology_generator.py --random_static [distance_file] [number of ISLs]")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Please provide an option as a command line argument!")
        printHelp()
        exit(1)
    
    if sys.argv[1] == "--help":
        printHelp()
    elif sys.argv[1] == "--random_static" and len(sys.argv) == 4:
        generate_random_topology(sys.argv[2], int(sys.argv[3]))
    else:
        print("Invalid Option or Missing Arguments!")
        printHelp()
        exit(1)

    