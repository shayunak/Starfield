import cudf
import networkx as nx
import pandas as pd
from multiprocessing import shared_memory

class CUGraphBuilder:
    def __init__(self, nodes):
        self.node_to_id = {node: i for i, node in enumerate(nodes)}
        self.id_to_node = {i: node for node, i in self.node_to_id.items()}
    
    def to_id(self, nodes):
        return [self.node_to_id[node] for node in nodes]

    def build_graph(self, src, dst, weight): 
        return cudf.DataFrame({'src': self.to_id(src), 'dst': self.to_id(dst), 'weight': weight})

class NXGraphBuilder:
    def build_graph(self, src, dst, weight):
        g = nx.Graph()
        edges = list(zip(src, dst, weight))
        g.add_weighted_edges_from(edges)
        return g

class GraphGenerator:
    def __init__(self, distances_df, constellation_name, graph_builder, number_of_nodes):
        self.distances_df = distances_df
        self.constellation_name = constellation_name
        self.graph_builder = graph_builder
        self.number_of_nodes = number_of_nodes

    def is_ground_station(self, node_id):
        splitted_id = node_id.split("-")
        if (len(splitted_id) == 3) and (splitted_id[0] == self.constellation_name):
            return False
        
        return True

    def check_sanity(self, time_stamp_data, time_stamp):
        pass

    def is_satellite_id(self, satellite_id):
        splitted_name = satellite_id.split("-")
        if len(splitted_name) == 3 and splitted_name[0] == self.constellation_name:
            return True
        return False

    def check_isl_edge(self, first_satellite_name, second_satellite_name, time_stamp):
        return True

    def get_graph(self, time_stamp):
        src, dst, weight = [], [], []
        timestamp_data = self.distances_df.loc[self.distances_df['TimeStamp(ms)'] == time_stamp]
        self.check_sanity(timestamp_data, time_stamp)

        for _, row in timestamp_data.iterrows():
            if row['FirstDeviceId'] != row['SecondDeviceId']:
                if self.is_satellite_id(row['FirstDeviceId']) and self.is_satellite_id(row['SecondDeviceId']):
                    if self.check_isl_edge(row['FirstDeviceId'], row['SecondDeviceId'], time_stamp):
                        src.append(row['FirstDeviceId'])
                        dst.append(row['SecondDeviceId'])
                        weight.append(row['Distance(m)'])
                else:
                    src.append(row['FirstDeviceId'])
                    dst.append(row['SecondDeviceId'])
                    weight.append(row['Distance(m)'])
                
        return self.graph_builder.build_graph(src, dst, weight)

class GridPlusGraphGenerator(GraphGenerator):
    def __init__(self, distances_df, constellation_name, graph_builder, number_of_nodes, number_of_orbits, number_of_satellites_per_orbit):
        super().__init__(distances_df, constellation_name, graph_builder, number_of_nodes)
        self.number_of_orbits = number_of_orbits
        self.number_of_satellites_per_orbit = number_of_satellites_per_orbit

    def get_orbit_anomaly_id(self, satellite_name):
        splitted_name = satellite_name.split("-")
        return int(splitted_name[1]), int(splitted_name[2])

    def check_isl_edge(self, first_satellite_name, second_satellite_name, time_stamp):
        first_orbit_id, first_satellite_id = self.get_orbit_anomaly_id(first_satellite_name)
        second_orbit_id, second_satellite_id = self.get_orbit_anomaly_id(second_satellite_name)
        next_of_first_orbit = (first_orbit_id + 1) % self.number_of_orbits
        next_of_second_orbit = (second_orbit_id + 1) % self.number_of_orbits
        next_of_first_satellite_id = (first_satellite_id + 1) % self.number_of_satellites_per_orbit
        next_of_second_satellite_id = (second_satellite_id + 1) % self.number_of_satellites_per_orbit

        if first_orbit_id == second_orbit_id:
            if next_of_first_satellite_id == second_satellite_id or next_of_second_satellite_id == first_satellite_id:
                return True
        
        if first_satellite_id == second_satellite_id:
            if next_of_first_orbit == second_orbit_id or next_of_second_orbit == first_orbit_id:
                return True

        return False
    
class TopologyGraphGenerator(GraphGenerator):
    def __init__(self, distances_df, constellation_name, graph_builder, number_of_nodes, topology_file):
        super().__init__(distances_df, constellation_name, graph_builder, number_of_nodes)
        self.topology = self.load_topology(topology_file)

class StaticTopologyGraphGenerator(TopologyGraphGenerator):
    def __init__(self, distances_df, constellation_name, graph_builder, number_of_nodes, topology_file):
        super().__init__(distances_df, constellation_name, graph_builder, number_of_nodes, topology_file)

    def load_topology(self, topology_file):
        topology_dataframe = pd.read_csv(f"./input/{topology_file}")
        return set(zip(topology_dataframe.FirstSatellite, topology_dataframe.SecondSatellite))

    def check_sanity(self, time_stamp_data, time_stamp):
        distance_pairs = set(zip(time_stamp_data.FirstDeviceId, time_stamp_data.SecondDeviceId))
        if not (self.topology <= distance_pairs):
            raise ValueError("Topology is not consistent with distances!")
        
    def check_isl_edge(self, first_satellite_name, second_satellite_name, time_stamp):
        return (first_satellite_name, second_satellite_name) in self.topology
        
class DynamicTopologyGraphGenerator(TopologyGraphGenerator):
    def __init__(self, distances_df, constellation_name, graph_builder, number_of_nodes, topology_file):
        super().__init__(distances_df, constellation_name, graph_builder, number_of_nodes, topology_file)

    def load_topology(self, topology_file):
        topology_dataframe = pd.read_csv(f"./input/{topology_file}")
        return topology_dataframe.groupby('TimeStamp(ms)').apply(lambda x: list(zip(x['FirstSatellite'], x['SecondSatellite']))).to_dict()

    def check_sanity(self, time_stamp_data, time_stamp):
        distance_pairs = set(zip(time_stamp_data.FirstDeviceId, time_stamp_data.SecondDeviceId))
        if not (self.topology[time_stamp] <= distance_pairs):
            raise ValueError("Topology is not consistent with distances!")

    def check_isl_edge(self, first_satellite_name, second_satellite_name, time_stamp):
        return (first_satellite_name, second_satellite_name) in self.topology[time_stamp]