import pandas as pd
import networkx as nx

def read_distance_file(filename):
    splited_filename = filename[:-4].split("#")
    if len(splited_filename) != 5:
        raise NameError("Incorrect distance file name format!")
    if splited_filename[0] != "Distances":
        raise NameError("Only distances files are accepted, and they start with 'Distances'!")
    time_step = int(splited_filename[3][:-2])
    total_time = int(splited_filename[4][:-1]) * 1000
    simulation_details = f"{splited_filename[2]}#{splited_filename[3]}#{splited_filename[4]}"
    distance_csv_dataframe = pd.read_csv(f"./generated/{filename}")
    nodes = distance_csv_dataframe['FirstSatelliteId'].unique().tolist()

    return distance_csv_dataframe, time_step, total_time, simulation_details, nodes

def generate_graph_from_timestamp_data(timeStamp, dataframe, nodes):
    distance_graph = nx.Graph()
    timestamp_data = dataframe.loc[dataframe['TimeStamp'] == timeStamp]
    distance_graph.add_nodes_from(nodes)
    for index, row in timestamp_data.iterrows():
        distance_graph.add_edge(row['FirstSatelliteId'], row['SecondSatelliteId'], weight=row['Distance'])

    # deleteing self-loops
    distance_graph.remove_edges_from(nx.selfloop_edges(distance_graph))

    return distance_graph


