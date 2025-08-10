import pandas as pd
import networkx as nx

def generate_general_graph_from_timestamp_data(timeStamp, dataframe, nodes):
    distance_graph = nx.Graph()
    timestamp_data = dataframe.loc[dataframe['TimeStamp(ms)'] == timeStamp]
    distance_graph.add_nodes_from(nodes)
    for _, row in timestamp_data.iterrows():
        distance_graph.add_edge(row['FirstDeviceId'], row['SecondDeviceId'], weight=row['Distance(m)'])

    # deleteing self-loops
    distance_graph.remove_edges_from(nx.selfloop_edges(distance_graph))

    return distance_graph

def get_orbit_anomaly_id(satellite_name):
    splitted_name = satellite_name.split("-")
    return int(splitted_name[1]), int(splitted_name[2])

def is_satellite_id(satellite_id, constellation_name):
    splitted_name = satellite_id.split("-")
    if len(splitted_name) == 3 and splitted_name[0] == constellation_name:
        return True
    return False

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

def generate_grid_plus_graph_from_timestamp_data(time_stamp, dataframe, nodes, number_of_orbits, number_of_satellites_per_orbit, constellation_name):
    graph = nx.Graph()
    timestamp_data = dataframe.loc[dataframe['TimeStamp(ms)'] == time_stamp]
    graph.add_nodes_from(nodes)
    for _, row in timestamp_data.iterrows():
        if is_satellite_id(row['FirstDeviceId'], constellation_name) and is_satellite_id(row['SecondDeviceId'], constellation_name):
            if is_edge_in_grid_plus(row['FirstDeviceId'], row['SecondDeviceId'], number_of_orbits, number_of_satellites_per_orbit):
                graph.add_edge(row['FirstDeviceId'], row['SecondDeviceId'], weight=row['Distance(m)'])
        else:
            graph.add_edge(row['FirstDeviceId'], row['SecondDeviceId'], weight=row['Distance(m)'])
        
    # deleteing self-loops
    graph.remove_edges_from(nx.selfloop_edges(graph))

    return graph

def read_static_topology_file(filename):
    topology_dataframe = pd.read_csv(f"./input/{filename}")
    topology = set(zip(topology_dataframe.FirstSatellite, topology_dataframe.SecondSatellite))

    return topology

def read_dynamic_topology_file(filename):
    topology_dataframe = pd.read_csv(f"./input/{filename}")
    topology = topology_dataframe.groupby('TimeStamp(ms)').apply(lambda x: list(zip(x['FirstSatellite'], x['SecondSatellite']))).to_dict()

    return topology

def check_topology_consistency(topology, distances):
    distance_pairs = set(zip(distances.FirstDeviceId, distances.SecondDeviceId))
    if not (topology <= distance_pairs):
        raise ValueError("Topology is not consistent with distances!")

def generate_static_topology_graph_from_timestamp_data(timestamp, dataframe, nodes, topology, constellation_name):
    timestamp_data = dataframe.loc[dataframe['TimeStamp(ms)'] == timestamp]
    check_topology_consistency(topology, timestamp_data)
    graph = nx.Graph()
    graph.add_nodes_from(nodes)
    for _, row in timestamp_data.iterrows():
        if is_satellite_id(row['FirstDeviceId'], constellation_name) and is_satellite_id(row['SecondDeviceId'], constellation_name):
            if (row['FirstDeviceId'], row['SecondDeviceId']) in topology:
                graph.add_edge(row['FirstDeviceId'], row['SecondDeviceId'], weight=row['Distance(m)'])
        else:
            graph.add_edge(row['FirstDeviceId'], row['SecondDeviceId'], weight=row['Distance(m)'])

    # deleteing self-loops
    graph.remove_edges_from(nx.selfloop_edges(graph))

    return graph

