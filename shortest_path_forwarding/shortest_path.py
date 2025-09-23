import sys
from time import time
import distance_file_graph_generator as dfg
import networkx as nx
import utility as util
import time
import pandas as pd

def calculate_isl_only_shortest_path_hops(csv_writers, time_stamp, distance_graph, graph_generator):
    all_pairs = dict(nx.all_pairs_dijkstra(distance_graph, weight='weight'))
    sat_source = {'SAT': [], 'GS': [], 'Min_Distance': []}
    gsl_pairs = graph_generator.gsl_gs_source_pairs
    for source, (distances, paths) in all_pairs.items():
        if graph_generator.is_satellite_id(source):
            for dest, path in paths.items():
                if source != dest and not graph_generator.is_satellite_id(dest):
                    csv_writers[source].writerow((time_stamp, dest, path[1]))
                    sat_source['SAT'].append(source)
                    sat_source['GS'].append(dest)
                    sat_source['Min_Distance'].append(distances[dest])

    sat_df = pd.DataFrame(sat_source)
    all_gs_paths = pd.merge(gsl_pairs, sat_df, left_on="SecondDeviceId", right_on="SAT")
    all_gs_paths['TotalDistance'] = all_gs_paths['Distance(m)'] + all_gs_paths['Min_Distance']
    idx = all_gs_paths.groupby(["FirstDeviceId", "GS"])['TotalDistance'].idxmin()
    min_gs_paths = all_gs_paths.loc[idx].reset_index(drop=True)
    for row in min_gs_paths.itertuples(index=False):
        if row.FirstDeviceId != row.GS:
            csv_writers[row.FirstDeviceId].writerow((time_stamp, row.GS, row.SAT))
    


def calculate_shortest_path_hops(csv_writers, timestamp, distance_graph, graph_generator):
    paths = nx.all_pairs_dijkstra_path(distance_graph)
    for node, other_nodes in paths:
        for other_node, path in other_nodes.items():
            if other_node != node and not graph_generator.is_satellite_id(other_node):
                csv_writers[node].writerow((timestamp, other_node, path[1]))

def calculate_shortest_paths(shortest_path, node_writers, total_time, time_step, graph_generator):
    for time_stamp in range(0, total_time + 1, time_step):
        start_time = time.time()
        graph = graph_generator.get_graph(time_stamp)
        shortest_path(node_writers, time_stamp, graph, graph_generator)
        print(f"Calculated forwarding table for timestamp {time_stamp} in {time.time() - start_time} seconds")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Please provide the required options as a command line argument!")
        util.printHelp()
        exit(1)
    
    if sys.argv[1] == "--help":
        util.printHelp()
        exit(1)
    elif len(sys.argv) < 3:
        print("Please provide the required options as a command line argument!")
        util.printHelp()
        exit(1)
    elif len(sys.argv) == 3:
        print("Please provide the distance file name!")
        util.printHelp()
        exit(1)

    distance_file_name = sys.argv[3]
    (distance_csv_dataframe, constellation_name, time_step, total_time, simulation_details, 
        nodes, number_of_orbits, number_of_satellites_per_orbit) = util.read_distance_file(distance_file_name)
    
    folder_name = ""
    graph_builder = dfg.NXGraphBuilder()
    shortest_path_algo = calculate_shortest_path_hops
    number_of_nodes = len(nodes)
    graph_generator = None
    link_filter_graph_generator = None

    if sys.argv[2] == "--dijkstra" and len(sys.argv) == 4:
        graph_generator = dfg.GraphGenerator()
        folder_name += "DijkstraForwardingTable"
    elif sys.argv[2] == "--dijkstra_grid_plus" and len(sys.argv) == 4:
        graph_generator = dfg.GridPlusGraphGenerator(number_of_orbits, number_of_satellites_per_orbit)
        folder_name += "DijkstraGridPlusForwardingTable"
    elif sys.argv[2] == "--dijkstra_static" and len(sys.argv) == 5:
        graph_generator = dfg.StaticTopologyGraphGenerator(sys.argv[4])
        folder_name += "DijkstraStaticForwardingTable"
    elif sys.argv[2] == "--dijkstra_dynamic" and len(sys.argv) == 5:
        graph_generator = dfg.DynamicTopologyGraphGenerator(sys.argv[4])
        folder_name += "DijkstraDynamicForwardingTable"
    else:
        print("Invalid Shortest Path Option or Missing Arguments!")
        util.printHelp()
        exit(1)

    if sys.argv[1] == "--isl":
        graph_builder = dfg.NXDirectedGraphBuilder()
        shortest_path_algo = calculate_isl_only_shortest_path_hops
        link_filter_graph_generator = dfg.OnlyISLLinkFilter(distance_csv_dataframe, constellation_name, graph_builder, graph_generator, number_of_nodes)
        folder_name += "(ISL_Only)"
    elif sys.argv[1] == "--gsl":
        link_filter_graph_generator = dfg.OnlyGSLLinkFilter(distance_csv_dataframe, constellation_name, graph_builder, graph_generator, number_of_nodes)
        folder_name += "(GSL_Only)"
    elif sys.argv[1] == "--isl&gsl":
        link_filter_graph_generator = dfg.ISLAndGSLLinkFilter(distance_csv_dataframe, constellation_name, graph_builder, graph_generator, number_of_nodes)
        folder_name += "(ISL_GSL)"
    else:
        print("Please provide the required options as a command line argument!")
        util.printHelp()
        exit(1)

    node_files, node_writers = util.forwarding_folder_csv_file(simulation_details, folder_name, nodes)
    calculate_shortest_paths(shortest_path_algo, node_writers, total_time, time_step, link_filter_graph_generator)
    util.close_files(node_files)
    

    