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
    if len(splited_filename) < 5:
        raise NameError("Incorrect distance file name format!")

    if splited_filename[0] != "Distances" and splited_filename[0] != "StaticConsistentDistances" and splited_filename[0] != "DynamicConsistentDistances":
        raise NameError("Only distances or consistent distances files are accepted, and they start with 'Distances' or 'ConsistentDistances'!")
    
    time_step = int(splited_filename[3][:-2])
    total_time = int(splited_filename[4][:-1]) * 1000
    file_time = splited_filename[1]
    simulation_details = f"{splited_filename[2]}#{splited_filename[3]}#{splited_filename[4]}"
    constellation_name, orbital_structure = splited_filename[2].split("(")
    number_of_orbits, number_of_satellites_per_orbit = map(int, orbital_structure.rstrip(")").split(","))

    if splited_filename[0] == "Distances":
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
        distance_csv_dataframe.drop("Distance(m)", axis=1)
        nodes = distance_csv_dataframe['FirstDeviceId'].unique().tolist()
        print(f"Read distance file '{filename}' with {len(nodes)} nodes, time step {time_step} ms, and total time {total_time} ms.")
        return False, distance_csv_dataframe, constellation_name, time_step, total_time, simulation_details, file_time, nodes, number_of_orbits, number_of_satellites_per_orbit
    elif splited_filename[0] == "StaticConsistentDistances":
        distance_csv_dataframe = pd.read_csv(
            f"./generated/{filename}",
            engine="pyarrow",
            sep=",",
            dtype={
                "FirstSatelliteId": "string",
                "SecondSatelliteId": "string",
            }                 
        )
        nodes = distance_csv_dataframe['FirstSatelliteId'].unique().tolist()
        consistent_graph = nx.from_pandas_edgelist(
            distance_csv_dataframe,
            source="FirstSatelliteId",
            target="SecondSatelliteId",
            create_using=nx.Graph()  # Ensures undirected
        )
        print(f"Read static consistent distance file '{filename}' with {len(nodes)} nodes, time step {time_step} ms, and total time {total_time} ms.")
        return True, consistent_graph, constellation_name, time_step, total_time, simulation_details, file_time, nodes, number_of_orbits, number_of_satellites_per_orbit
    else:
        time_interval = int(splited_filename[5][:-1]) * 1000
        distance_csv_dataframe = pd.read_csv(
            f"./generated/{filename}",
            engine="pyarrow",
            sep=",",
            dtype={
                "TimeStamp(ms)": "int64",
                "FirstSatelliteId": "string",
                "SecondSatelliteId": "string",
            }                 
        )
        nodes = distance_csv_dataframe['FirstSatelliteId'].unique().tolist()
        graphs = {}
        for t, group in distance_csv_dataframe.groupby("TimeStamp(ms)"):
            G = nx.Graph()
            G.add_edges_from(zip(group["FirstSatelliteId"], group["SecondSatelliteId"]))
            graphs[t] = G

        print(f"Read dynamic consistent distance file '{filename}' with {len(nodes)} nodes, time step {time_step} ms, and total time {total_time} ms.")
        return True, graphs, constellation_name, time_step, total_time, simulation_details, time_interval, nodes, number_of_orbits, number_of_satellites_per_orbit

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
    G.add_edges_from(
        (u, v) for u, v in filtered.itertuples(index=False, name=None)
    )
    return G

def get_static_consistent_distance_graph(df, distance_file_name, nodes, constellation_name, time_step, total_time):
    satellite_nodes = [node for node in nodes if not is_ground_station(node, constellation_name)]
    consistent_distance_graph = generate_general_satellite_graph_from_timestamp_data(0, df, satellite_nodes, constellation_name)
    for time_stamp in range(time_step, total_time + time_step, time_step):
        graph = generate_general_satellite_graph_from_timestamp_data(time_stamp, df, satellite_nodes, constellation_name)
        consistent_distance_graph = nx.intersection(consistent_distance_graph, graph)

    consistent_edges = []
    for u, v in consistent_distance_graph.edges():
        consistent_edges.append((u, v))
        consistent_edges.append((v, u))

    edges_df = pd.DataFrame(consistent_edges, columns=["FirstSatelliteId", "SecondSatelliteId"])
    edges_df.to_csv(f"./generated/StaticConsistent{distance_file_name}", index=False)

    return consistent_distance_graph, satellite_nodes

def get_dynamic_consistent_distance_graphs(df, nodes, constellation_name, num_orbits, num_sats, file_time, time_step, time_interval, time_period):
    satellite_nodes = [node for node in nodes if not is_ground_station(node, constellation_name)]
    consistent_distance_graphs = {}
    for time in range(0, time_period*1000 + 1, time_interval*1000):
        consistent_distance_graph = generate_general_satellite_graph_from_timestamp_data(time, df, satellite_nodes, constellation_name)
        for t in range(time, time + time_interval*1000, time_step):
            graph = generate_general_satellite_graph_from_timestamp_data(t, df, satellite_nodes, constellation_name)
            consistent_distance_graph = nx.intersection(consistent_distance_graph, graph)
        consistent_distance_graphs[time] = consistent_distance_graph

    consistent_edges = []
    for time, graph in consistent_distance_graphs.items():
        for u, v in graph.edges():
            consistent_edges.append((time, u, v))
            consistent_edges.append((time, v, u))

    edges_df = pd.DataFrame(consistent_edges, columns=["TimeStamp(ms)", "FirstSatelliteId", "SecondSatelliteId"])
    edges_df.to_csv(f"./generated/DynamicConsistentDistances#{file_time}#{constellation_name}({num_orbits},{num_sats})#{time_step}ms#{time_period}s#{time_interval}s.csv", index=False)

    return consistent_distance_graphs, satellite_nodes