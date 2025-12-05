import cudf
import networkx as nx
import pandas as pd

class CUGraphBuilder:
    def __init__(self, nodes, constellation_name):
        self.node_to_id = {node: i for i, node in enumerate(nodes)}
        self.id_to_node = {i: node for node, i in self.node_to_id.items()}
        self.constellation_name = constellation_name
        self.ground_station_id_set = {i for i, node in self.id_to_node.items() if self.is_ground_station(node)}
    
    def is_id_ground_station(self, id):
        return id in self.ground_station_id_set

    def is_ground_station(self, node_id):
        splitted_id = node_id.split("-")
        if (len(splitted_id) == 3) and (splitted_id[0] == self.constellation_name):
            return False
        
        return True

    def to_id(self, nodes):
        return [self.node_to_id[node] for node in nodes]

    def build_graph(self, src, dst, weight):
        sources = self.to_id(src)
        unique_sources = list(set(sources))
        return cudf.DataFrame({'src': sources, 'dst': self.to_id(dst), 'weight': weight}), unique_sources

class NXGraphBuilder:
    def build_graph(self, src, dst, weight):
        g = nx.Graph()
        edges = list(zip(src, dst, weight))
        g.add_weighted_edges_from(edges)
        return g

class NXDirectedGraphBuilder:
    def build_graph(self, src, dst, weight):
        g = nx.DiGraph()
        edges = list(zip(src, dst, weight))
        g.add_weighted_edges_from(edges)
        return g

class GraphLinkFilter:
    def __init__(self, distances_df, constellation_name, graph_builder, graph_generator, number_of_nodes):
        self.distances_df = distances_df
        self.graph_generator = graph_generator
        self.constellation_name = constellation_name
        self.graph_builder = graph_builder
        self.number_of_nodes = number_of_nodes

    def is_satellite_id(self, id):
        splitted_name = id.split("-")
        if len(splitted_name) == 3 and splitted_name[0] == self.constellation_name:
            return True
        return False
    
    def generate_graph(self, time_stamp, gsl_pairs, isl_pairs):
        pass
    
    def get_graph(self, time_stamp):
        timestamp_data = self.distances_df.loc[self.distances_df['TimeStamp(ms)'] == time_stamp]

        timestamp_data = timestamp_data.loc[
            timestamp_data['FirstDeviceId'] != timestamp_data['SecondDeviceId']
        ] # remove self-loops

        first_is_sat = timestamp_data['FirstDeviceId'].apply(self.is_satellite_id)
        second_is_sat = timestamp_data['SecondDeviceId'].apply(self.is_satellite_id)

        both_sat_mask = first_is_sat & second_is_sat
        isl_pairs = timestamp_data.loc[both_sat_mask]

        gsl_pairs = timestamp_data.loc[~both_sat_mask]

        gsl_pairs, isl_pairs = self.generate_graph(time_stamp, gsl_pairs, isl_pairs)

        final_edges = pd.concat([isl_pairs, gsl_pairs], ignore_index=True)

        src = final_edges['FirstDeviceId'].tolist()
        dst = final_edges['SecondDeviceId'].tolist()
        weight = final_edges['Distance(m)'].tolist()

        return self.graph_builder.build_graph(src, dst, weight)

class OnlyISLLinkFilter(GraphLinkFilter):
    def __init__(self, distances_df, constellation_name, graph_builder, graph_generator, number_of_nodes):
        super().__init__(distances_df, constellation_name, graph_builder, graph_generator, number_of_nodes)
        self.gsl_gs_source_pairs = None

    def generate_graph(self, time_stamp, gsl_pairs, isl_pairs):
        gsl_satellite_source_pairs = gsl_pairs[gsl_pairs["FirstDeviceId"].apply(self.is_satellite_id)]
        self.gsl_gs_source_pairs = gsl_pairs[~gsl_pairs["FirstDeviceId"].apply(self.is_satellite_id)]

        return gsl_satellite_source_pairs, self.graph_generator.get_graph(time_stamp, isl_pairs)

class OnlyGSLLinkFilter(GraphLinkFilter):
    def __init__(self, distances_df, constellation_name, graph_builder, graph_generator, number_of_nodes):
        super().__init__(distances_df, constellation_name, graph_builder, graph_generator, number_of_nodes)

    def generate_graph(self, time_stamp, gsl_pairs, isl_pairs):
        return gsl_pairs, pd.DataFrame(columns=isl_pairs.columns)

class ISLAndGSLLinkFilter(GraphLinkFilter):
    def __init__(self, distances_df, constellation_name, graph_builder, graph_generator, number_of_nodes):
        super().__init__(distances_df, constellation_name, graph_builder, graph_generator, number_of_nodes)

    def generate_graph(self, time_stamp, gsl_pairs, isl_pairs):
        return gsl_pairs, self.graph_generator.get_graph(time_stamp, isl_pairs)

class GraphGenerator:
    def check_sanity(self, time_stamp_data, time_stamp):
        pass

    def check_isl_edge(self, first_satellite_name, second_satellite_name, time_stamp):
        return True

    def get_graph(self, time_stamp, isl_pairs):
        self.check_sanity(isl_pairs, time_stamp)

        if not isl_pairs.empty:
            isl_mask = [
                self.check_isl_edge(f, s, time_stamp)
                for f, s in zip(isl_pairs['FirstDeviceId'], isl_pairs['SecondDeviceId'])
            ]
            isl_pairs = isl_pairs.loc[isl_mask]

        return isl_pairs

class GridPlusGraphGenerator(GraphGenerator):
    def __init__(self, number_of_orbits, number_of_satellites_per_orbit):
        super().__init__()
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
    def __init__(self, topology_file):
        super().__init__()
        self.topology = self.load_topology(topology_file)

class StaticTopologyGraphGenerator(TopologyGraphGenerator):
    def __init__(self, topology_file):
        super().__init__(topology_file)

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
    def __init__(self, topology_file):
        super().__init__(topology_file)
        self.time_ranges = sorted(list(self.topology.keys()))
    
    def load_topology(self, topology_file):
        topology_dataframe = pd.read_csv(f"./input/{topology_file}")
        return topology_dataframe.groupby('TimeStamp(ms)').apply(lambda x: set(zip(x['FirstSatellite'], x['SecondSatellite'])), include_groups=False).to_dict()

    def find_range(self, time_stamp):
        for i in range(len(self.time_ranges)):
            if i == len(self.time_ranges) - 1:
                return self.time_ranges[i]
            if time_stamp < self.time_ranges[i+1] and time_stamp >= self.time_ranges[i]:
                return self.time_ranges[i]
        
        return 0

    def check_sanity(self, time_stamp_data, time_stamp):
        distance_pairs = set(zip(time_stamp_data.FirstDeviceId, time_stamp_data.SecondDeviceId))
        time_stamp = self.find_range(time_stamp)
        if not (self.topology[time_stamp] <= distance_pairs):
            raise ValueError("Topology is not consistent with distances!")

    def check_isl_edge(self, first_satellite_name, second_satellite_name, time_stamp):
        time_stamp = self.find_range(time_stamp)
        return (first_satellite_name, second_satellite_name) in self.topology[time_stamp]