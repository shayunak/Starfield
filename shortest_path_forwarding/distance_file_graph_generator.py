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

def generate_general_graph_from_timestamp_data(timeStamp, dataframe, nodes):
    distance_graph = nx.Graph()
    timestamp_data = dataframe.loc[dataframe['TimeStamp'] == timeStamp]
    distance_graph.add_nodes_from(nodes)
    for index, row in timestamp_data.iterrows():
        distance_graph.add_edge(row['FirstSatelliteId'], row['SecondSatelliteId'], weight=row['Distance'])

    # deleteing self-loops
    distance_graph.remove_edges_from(nx.selfloop_edges(distance_graph))

    return distance_graph

def get_orbit_anomaly_id(satellite_name):
    splitted_name = satellite_name.split("-")
    return int(splitted_name[1]), int(splitted_name[2])

def is_edge_in_grid_plus(first_satellite_name, second_satellite_name,  number_of_orbits, number_of_satellites_per_orbit):
    first_orbit_id, first_satellite_id = get_orbit_anomaly_id(first_satellite_name)
    second_orbit_id, second_satellite_id = get_orbit_anomaly_id(second_satellite_name)
    next_of_first_orbit = (first_orbit_id + 1) % number_of_orbits
    next_of_second_orbit = (second_orbit_id + 1) % number_of_orbits
    next_of_first_satellite_id = (first_satellite_id + 1) % number_of_satellites_per_orbit
    next_of_second_satellite_id = (second_satellite_id + 1) % number_of_satellites_per_orbit

    if first_orbit_id == second_orbit_id:
        if next_of_first_satellite_id == second_satellite_id or next_of_second_satellite_id == first_satellite_id:
            return True
    
    if first_satellite_id == second_satellite_id:
        if next_of_first_orbit == second_orbit_id or next_of_second_orbit == first_orbit_id:
            return True

    return False


def generate_grid_plus_graph_from_timestamp_data(timeStamp, dataframe, nodes, number_of_orbits, number_of_satellites_per_orbit):
    distance_graph = nx.Graph()
    timestamp_data = dataframe.loc[dataframe['TimeStamp'] == timeStamp]
    distance_graph.add_nodes_from(nodes)
    for index, row in timestamp_data.iterrows():
        if is_edge_in_grid_plus(row['FirstSatelliteId'], row['SecondSatelliteId'], number_of_orbits, number_of_satellites_per_orbit):
            distance_graph.add_edge(row['FirstSatelliteId'], row['SecondSatelliteId'], weight=row['Distance'])

    # deleteing self-loops
    distance_graph.remove_edges_from(nx.selfloop_edges(distance_graph))

    return distance_graph

