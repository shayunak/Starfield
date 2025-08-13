import sys
from time import time
import distance_file_graph_generator as dfg
import networkx as nx
import utility as util
import time

def calculate_shortest_path_hops(csv_writers, timestamp, distance_graph, graph_generator):
    paths = nx.all_pairs_dijkstra_path(distance_graph)
    for node, other_nodes in paths:
        for other_node, path in other_nodes.items():
            if other_node != node and not graph_generator.is_satellite_id(other_node):
                csv_writers[node].writerow((timestamp, other_node, path[1]))

def calculate_shortest_paths(node_writers, node_files, total_time, time_step, graph_generator):
    for time_stamp in range(0, total_time + 1, time_step):
        start_time = time.time()
        graph = graph_generator.get_graph(time_stamp)
        calculate_shortest_path_hops(node_writers, time_stamp, graph, graph_generator)
        print(f"Calculated forwarding table for timestamp {time_stamp} in {time.time() - start_time} seconds")

    util.close_files(node_files)

def dijkstra_shortest_path_algorithm(distance_file_name):
    distance_csv_dataframe, constellation_name, time_step, total_time, simulation_details, nodes, _, _ = util.read_distance_file(distance_file_name)
    node_files, node_writers = util.forwarding_folder_csv_file(simulation_details, "DijkstraForwardingTable", nodes)
    graph_generator = dfg.GraphGenerator(distance_csv_dataframe, constellation_name, dfg.NXGraphBuilder(), len(nodes))
    calculate_shortest_paths(node_writers, node_files, total_time, time_step, graph_generator)

def dijkstra_grid_plus_shortest_path_algorithm(distance_file_name):
    distance_csv_dataframe, constellation_name, time_step, total_time, simulation_details, nodes, number_of_orbits, number_of_satellites_per_orbit = util.read_distance_file(distance_file_name)
    node_files, node_writers = util.forwarding_folder_csv_file(simulation_details, "DijkstraGridPlusForwardingTable", nodes)
    grid_plus_graph_generator = dfg.GridPlusGraphGenerator(distance_csv_dataframe, constellation_name, dfg.NXGraphBuilder(), len(nodes), number_of_orbits, number_of_satellites_per_orbit)
    calculate_shortest_paths(node_writers, node_files, total_time, time_step, grid_plus_graph_generator)

def dijkstra_static_topology_shortest_path_algorithm(distance_file_name, topology_file_name):
    distance_csv_dataframe, constellation_name, time_step, total_time, simulation_details, nodes, _, _ = util.read_distance_file(distance_file_name)
    node_files, node_writers = util.forwarding_folder_csv_file(simulation_details, "DijkstraStaticForwardingTable", nodes)
    static_topology_graph_generator = dfg.StaticTopologyGraphGenerator(distance_csv_dataframe, constellation_name, dfg.NXGraphBuilder(), len(nodes), topology_file_name)
    calculate_shortest_paths(node_writers, node_files, total_time, time_step, static_topology_graph_generator)

def dijkstra_dynamic_topology_shortest_path_algorithm(distance_file_name, topology_file_name):
    distance_csv_dataframe, constellation_name, time_step, total_time, simulation_details, nodes, _, _ = util.read_distance_file(distance_file_name)
    node_files, node_writers = util.forwarding_folder_csv_file(simulation_details, "DijkstraDynamicForwardingTable", nodes)
    dynamic_topology_graph_generator = dfg.DynamicTopologyGraphGenerator(distance_csv_dataframe, constellation_name, dfg.NXGraphBuilder(), len(nodes), topology_file_name)
    calculate_shortest_paths(node_writers, node_files, total_time, time_step, dynamic_topology_graph_generator)

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Please provide an option as a command line argument!")
        util.printHelp()
        exit(1)
    
    if sys.argv[1] == "--help":
        util.printHelp()
    elif sys.argv[1] == "--dijkstra" and len(sys.argv) == 3:
        dijkstra_shortest_path_algorithm(sys.argv[2])
    elif sys.argv[1] == "--dijkstra_grid_plus" and len(sys.argv) == 3:
        dijkstra_grid_plus_shortest_path_algorithm(sys.argv[2])
    elif sys.argv[1] == "--dijkstra_static_topology" and len(sys.argv) == 4:
        dijkstra_static_topology_shortest_path_algorithm(sys.argv[2], sys.argv[3])
    elif sys.argv[1] == "--dijkstra_dynamic_topology" and len(sys.argv) == 4:
        dijkstra_dynamic_topology_shortest_path_algorithm(sys.argv[2], sys.argv[3])
    else:
        print("Invalid Option or Missing Arguments!")
        util.printHelp()
        exit(1)

    