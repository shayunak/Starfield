import pandas as pd
import networkx as nx

def is_ground_station(node_id, constellation_name):
    splitted_id = node_id.split("-")
    if (len(splitted_id) == 3) and (splitted_id[0] == constellation_name):
        return False
	
    return True

def in_same_orbit(satellite_id_1, satellite_id_2):
    splitted_id_1 = satellite_id_1.split("-")
    splitted_id_2 = satellite_id_2.split("-")
    return splitted_id_1[1] == splitted_id_2[1]

def read_distance_file(filename):
    splited_filename = filename[:-4].split("#")
    if len(splited_filename) != 5:
        raise NameError("Incorrect distance file name format!")
    if splited_filename[0] != "Distances":
        raise NameError("Only distances files are accepted, and they start with 'Distances'!")
    time_step = int(splited_filename[3][:-2])
    total_time = int(splited_filename[4][:-1]) * 1000
    simulation_details = f"{splited_filename[2]}#{splited_filename[3]}#{splited_filename[4]}"
    constellation_name = splited_filename[2]
    distance_csv_dataframe = pd.read_csv(f"./generated/{filename}")
    nodes = distance_csv_dataframe['FirstDeviceId'].unique().tolist()

    return distance_csv_dataframe, constellation_name, time_step, total_time, simulation_details, nodes

def generate_general_satellite_graph_from_timestamp_data(timeStamp, dataframe, nodes, constellation_name):
    distance_graph = nx.Graph()
    timestamp_data = dataframe.loc[dataframe['TimeStamp(ms)'] == timeStamp]
    distance_graph.add_nodes_from(nodes)
    for _, row in timestamp_data.iterrows():
        if  not is_ground_station(row['FirstDeviceId'], constellation_name) and not is_ground_station(row['SecondDeviceId'], constellation_name):
            distance_graph.add_edge(row['FirstDeviceId'], row['SecondDeviceId'])

    # deleteing self-loops
    distance_graph.remove_edges_from(nx.selfloop_edges(distance_graph))

    return distance_graph

def get_consistent_distance_graph(df, nodes, constellation_name, time_step, total_time):
    graphs = []
    satellite_nodes = [node for node in nodes if not is_ground_station(node, constellation_name)]
    for time_stamp in range(0, total_time + 1, time_step):
        graph = generate_general_satellite_graph_from_timestamp_data(time_stamp, df, satellite_nodes, constellation_name)
        graphs.append(graph)

    consistent_distance_graph = nx.intersection_all(graphs)

    return consistent_distance_graph, satellite_nodes