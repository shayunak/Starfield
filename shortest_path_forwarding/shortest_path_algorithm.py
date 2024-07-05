import sys, datetime, os, csv
import matplotlib.pyplot as plt
import distance_file_graph_generator as dfg
import networkx as nx

def calculate_shortest_path_hops(csv_writer, timestamp, distance_graph):
    paths = nx.all_pairs_dijkstra_path(distance_graph)
    for node, other_nodes in paths:
        for other_node, path in other_nodes.items():
            if other_node != node:
                csv_writer.writerow((timestamp, node, other_node, path[1]))

def forwarding_table_csv_file(simulation_details):
    # making forwarding table output file folder, name
    file_name = f"DijkstraForwardingTable#{datetime.datetime.today().strftime('%Y_%m_%d,%H_%M_%S')}#{simulation_details}.csv"
    if not os.path.exists("./forwarding_table"):
        os.makedirs("./forwarding_table")

    forwarding_file = open(f"./forwarding_table/{file_name}", "w", newline='', encoding='utf-8')
    csv_writer = csv.writer(forwarding_file)
    csv_writer.writerow(["TimeStamp", "Source", "Destination", "NextHop"])

    return forwarding_file, csv_writer

if __name__ == "__main__":
    if len(sys.argv) < 2:
        raise NameError("Please provide a distance file name as a command line argument!")
    
    distance_file_name = sys.argv[1]
    distance_csv_dataframe, time_step, total_time, simulation_details = dfg.read_distance_file(distance_file_name)
    forwarding_file, forwarding_output_writer = forwarding_table_csv_file(simulation_details)
    for timestamp in range(0, total_time + 1, time_step):
        graph = dfg.generate_graph_from_timestamp_data(timestamp, distance_csv_dataframe)
        calculate_shortest_path_hops(forwarding_output_writer, timestamp, graph)
        print(f"Calculated forwarding table for timestamp {timestamp}...")
    forwarding_file.close()