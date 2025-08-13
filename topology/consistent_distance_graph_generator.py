import pandas as pd
import networkx as nx
import time

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
    splited_filename = filename[:-4].split("#")
    if len(splited_filename) != 5:
        raise NameError("Incorrect distance file name format!")
    if splited_filename[0] != "Distances":
        raise NameError("Only distances files are accepted, and they start with 'Distances'!")
    time_step = int(splited_filename[3][:-2])
    total_time = int(splited_filename[4][:-1]) * 1000
    simulation_details = f"{splited_filename[2]}#{splited_filename[3]}#{splited_filename[4]}"
    constellation_name, orbital_structure = splited_filename[2].split("(")
    number_of_orbits, number_of_satellites_per_orbit = map(int, orbital_structure.rstrip(")").split(","))
    distance_csv_dataframe = pd.read_csv(
        f"./generated/{filename}",
        engine="pyarrow",
        sep=",",
        dtype={
            "TimeStamp(ms)": "int64",
            "FirstDeviceId": "string",
            "SecondDeviceId": "string",
            "Distance(m)": "int64",
        }                 
    )
    nodes = distance_csv_dataframe['FirstDeviceId'].unique().tolist()
    print(f"Read distance file '{filename}' with {len(nodes)} nodes, time step {time_step} ms, and total time {total_time} ms.")

    return distance_csv_dataframe, constellation_name, time_step, total_time, simulation_details, nodes, number_of_orbits, number_of_satellites_per_orbit

def generate_general_satellite_graph_from_timestamp_data(time_stamp, dataframe, nodes, constellation_name):
    timestamp_data = dataframe.loc[dataframe['TimeStamp(ms)'] == time_stamp]

    mask = (
        (timestamp_data['FirstDeviceId'] != timestamp_data['SecondDeviceId']) &
        (~timestamp_data['FirstDeviceId'].apply(is_ground_station, args=(constellation_name,))) &
        (~timestamp_data['SecondDeviceId'].apply(is_ground_station, args=(constellation_name,)))
    )
    filtered = timestamp_data.loc[mask, ['FirstDeviceId', 'SecondDeviceId']]

    G = nx.Graph()
    G.add_nodes_from(nodes)
    G.add_edges_from(filtered.itertuples(index=False, name=None))
    return G

def get_consistent_distance_graph(df, nodes, constellation_name, time_step, total_time):
    satellite_nodes = [node for node in nodes if not is_ground_station(node, constellation_name)]
    consistent_distance_graph = generate_general_satellite_graph_from_timestamp_data(0, df, satellite_nodes, constellation_name)
    for time_stamp in range(time_step, total_time + time_step, time_step):
        graph = generate_general_satellite_graph_from_timestamp_data(time_stamp, df, satellite_nodes, constellation_name)
        consistent_distance_graph = nx.intersection(consistent_distance_graph, graph)

    return consistent_distance_graph, satellite_nodes