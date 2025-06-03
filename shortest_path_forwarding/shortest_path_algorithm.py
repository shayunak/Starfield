import sys, datetime, os, csv
import matplotlib.pyplot as plt
import distance_file_graph_generator as dfg
import networkx as nx

def is_ground_station(node_id, constellation_name):
    splitted_id = node_id.split("-")
    if (len(splitted_id) == 3) and (splitted_id[0] == constellation_name):
        return False
	
    return True

def calculate_shortest_path_hops(csv_writers, timestamp, distance_graph, constellation_name):
    paths = nx.all_pairs_dijkstra_path(distance_graph)
    for node, other_nodes in paths:
        for other_node, path in other_nodes.items():
            if other_node != node and is_ground_station(other_node, constellation_name):
                csv_writers[node].writerow((timestamp, node, other_node, path[1]))

def forwarding_folder_csv_file(simulation_details, title, nodes):
    # making forwarding table output folder, name
    node_files = []
    node_writers = {}
    folder_name = f"{title}#{datetime.datetime.today().strftime('%Y_%m_%d,%H_%M_%S')}#{simulation_details}"
    if not os.path.exists("./forwarding_table"):
        os.makedirs("./forwarding_table")

    os.makedirs(f"./forwarding_table/{folder_name}")

    for node in nodes:
        file = open(f"./forwarding_table/{folder_name}/{node}.csv", "w", newline= "")
        csv_writer = csv.writer(file)
        csv_writer.writerow(["TimeStamp", "Source", "Destination", "NextHop"])
        node_files.append(file)
        node_writers[node] = csv_writer

    return node_files, node_writers

def close_files(node_files):
    for file in node_files:
        file.close()


def printHelp():
    print("shortest_path_algorithm.py --help")
    print("shortest_path_algorithm.py --dijkstra [distance file]")
    print("shortest_path_algorithm.py --dijkstra_grid_plus [distance file] [number of orbits] [number of satellites per orbit]")
    print("shortest_path_algorithm.py --dijkstra_static [distance file] [topology_file_static]")
    print("shortest_path_algorithm.py --dijkstra_dynamic [distance file] [topology_file_dynamic]")

def dijkstraShortestPathAlgorithm(distance_file_name):
    distance_csv_dataframe, constellation_name, time_step, total_time, simulation_details, nodes = dfg.read_distance_file(distance_file_name)
    node_files, node_writers = forwarding_folder_csv_file(simulation_details, "DijkstraForwardingTable", nodes)
    for time_stamp in range(0, total_time + 1, time_step):
        graph = dfg.generate_general_graph_from_timestamp_data(time_stamp, distance_csv_dataframe, nodes)
        calculate_shortest_path_hops(node_writers, time_stamp, graph, constellation_name)
        print(f"Calculated forwarding table for timestamp {time_stamp}...")
    
    close_files(node_files)

def dijkstraGridPlusShortestPathAlgorithm(distance_file_name, number_of_orbits, number_of_satellites_per_orbit):
    distance_csv_dataframe, constellation_name, time_step, total_time, simulation_details, nodes = dfg.read_distance_file(distance_file_name)
    node_files, node_writers = forwarding_folder_csv_file(simulation_details, "DijkstraGridPlusForwardingTable", nodes)
    for time_stamp in range(0, total_time + 1, time_step):
        graph = dfg.generate_grid_plus_graph_from_timestamp_data(time_stamp, distance_csv_dataframe, nodes, number_of_orbits, number_of_satellites_per_orbit, constellation_name)
        calculate_shortest_path_hops(node_writers, time_stamp, graph, constellation_name)
        print(f"Calculated forwarding table for timestamp {time_stamp}...")
    
    close_files(node_files)

def dijkstraStaticTopologyShortestPathAlgorithm(distance_file_name, topology_file_name):
    distance_csv_dataframe, constellation_name, time_step, total_time, simulation_details, nodes = dfg.read_distance_file(distance_file_name)
    node_files, node_writers = forwarding_folder_csv_file(simulation_details, "DijkstraStaticForwardingTable", nodes)
    topology = dfg.read_static_topology_file(topology_file_name)
    for time_stamp in range(0, total_time + 1, time_step):
        graph = dfg.generate_static_topology_graph_from_timestamp_data(time_stamp, distance_csv_dataframe, nodes, topology, constellation_name)
        calculate_shortest_path_hops(node_writers, time_stamp, graph, constellation_name)
        print(f"Calculated forwarding table for timestamp {time_stamp}...")
    
    close_files(node_files)

def dijkstraDynamicTopologyShortestPathAlgorithm(distance_file_name, topology_file_name):
    distance_csv_dataframe, constellation_name, time_step, total_time, simulation_details, nodes = dfg.read_distance_file(distance_file_name)
    node_files, node_writers = forwarding_folder_csv_file(simulation_details, "DijkstraDynamicForwardingTable", nodes)
    topology = dfg.read_dynamic_topology_file(topology_file_name)
    for time_stamp in range(0, total_time + 1, time_step):
        graph = dfg.generate_static_topology_graph_from_timestamp_data(time_stamp, distance_csv_dataframe, nodes, topology[time_stamp], constellation_name)
        calculate_shortest_path_hops(node_writers, time_stamp, graph, constellation_name)
        print(f"Calculated forwarding table for timestamp {time_stamp}...")
    
    close_files(node_files)

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Please provide an option as a command line argument!")
        printHelp()
        exit(1)
    
    if sys.argv[1] == "--help":
        printHelp()
    elif sys.argv[1] == "--dijkstra" and len(sys.argv) == 3:
        dijkstraShortestPathAlgorithm(sys.argv[2])
    elif sys.argv[1] == "--dijkstra_grid_plus" and len(sys.argv) == 5:
        dijkstraGridPlusShortestPathAlgorithm(sys.argv[2], int(sys.argv[3]), int(sys.argv[4]))
    elif sys.argv[1] == "--dijkstra_static_topology" and len(sys.argv) == 4:
        dijkstraStaticTopologyShortestPathAlgorithm(sys.argv[2], sys.argv[3])
    elif sys.argv[1] == "--dijkstra_dynamic_topology" and len(sys.argv) == 4:
        dijkstraDynamicTopologyShortestPathAlgorithm(sys.argv[2], sys.argv[3])
    else:
        print("Invalid Option or Missing Arguments!")
        printHelp()
        exit(1)

    