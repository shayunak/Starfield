import csv
from datetime import datetime
import sys
import random_topology_generator as rtg
import riemannian_topology_generator as rmtg
import consistent_distance_graph_generator as cdg

def save_static_topology_to_file(graph, nodes, filename):
    with open(filename, 'w', newline='') as file:
        writer = csv.writer(file)
        writer.writerow(['FirstSatellite', 'SecondSatellite'])
        for node in nodes:
            for neighbor in graph.neighbors(node):
                writer.writerow([node, neighbor])

    print(f"Topology saved to {filename}")

def save_dynamic_topology_to_file(graphs, nodes, filename):
    with open(filename, 'w', newline='') as file:
        writer = csv.writer(file)
        if len(graphs) > 1:
            writer.writerow(['TimeStamp(ms)', 'FirstSatellite', 'SecondSatellite'])
        else:
            writer.writerow(['FirstSatellite', 'SecondSatellite'])
        if len(graphs) > 1:
            for time, graph in graphs:
                for node in nodes:
                    for neighbor in graph.neighbors(node):
                        writer.writerow([time, node, neighbor])
        else:
            time, graph = graphs[0]
            for node in nodes:
                for neighbor in graph.neighbors(node):
                    writer.writerow([node, neighbor])
    print(f"Topology saved to {filename}")

def save_fields_to_file(fields, filename):
    with open(filename, 'w', newline='') as file:
        writer = csv.writer(file)
        if len(fields) > 1:
            writer.writerow(['Timestamp', 'Satellite', 'Field_X', 'Field_Y', 'Field_Z'])
        else:
            writer.writerow(['Satellite', 'Field_X', 'Field_Y', 'Field_Z'])
        if len(fields) > 1:
            for time, field_data in fields:
                for i in range(len(field_data["Satellite"])):
                    writer.writerow([
                        time,
                        field_data["Satellite"][i],
                        field_data["Field_X"][i],
                        field_data["Field_Y"][i],
                        field_data["Field_Z"][i]
                    ])
        else:
            field_data = fields[0]
            for i in range(len(field_data["Satellite"])):
                writer.writerow([
                    field_data["Satellite"][i],
                    field_data["Field_X"][i],
                    field_data["Field_Y"][i],
                    field_data["Field_Z"][i]
                ])
    print(f"Fields saved to {filename}")

def random_static_topology(distance_file, num_isls):
    is_consistent_graph, df_graph, constellation_name, time_step, total_time, simulation_details, nodes, num_orbits, num_satellites = cdg.read_distance_file(distance_file)
    consistent_distance_graph, satellite_nodes = df_graph, nodes
    if not is_consistent_graph:
        consistent_distance_graph, satellite_nodes = cdg.get_consistent_distance_graph(df_graph, distance_file, nodes, constellation_name, time_step, total_time)

    topology_graph = rtg.generate_random_static_topology(satellite_nodes, num_orbits, num_satellites, constellation_name, consistent_distance_graph, num_isls)

    filename = f"RandomStaticTopology#{datetime.today().strftime('%Y_%m_%d,%H_%M_%S')}#{simulation_details}.csv"
    save_static_topology_to_file(topology_graph, satellite_nodes, f'./input/{filename}')

def riemannian_dynamic_topology(distance_file, cartesian_positions_file, demand_matrix_file, num_isls, time_period, time_interval):
    is_consistent_graph, df_graph, constellation_name, time_step, total_time, _, nodes, num_orbits, num_satellites = cdg.read_distance_file(distance_file)
    consistent_distance_graph, satellite_nodes = df_graph, nodes
    if not is_consistent_graph:
        consistent_distance_graph, satellite_nodes = cdg.get_consistent_distance_graph(df_graph, distance_file, nodes, constellation_name, time_step, total_time)

    ground_station_positions, satellite_positions = rmtg.get_cartesian_positions(cartesian_positions_file, constellation_name)
    flows_traffics = rmtg.get_flows_traffics(demand_matrix_file)

    topology_graphs = []
    for time_stamp in range(0, time_period*1000 + 1, time_interval*1000):
        traffic_flow = flows_traffics[time_stamp]
        satellite_position = satellite_positions[time_stamp]
        ground_station_position = ground_station_positions[time_stamp]
        topology_graph = rmtg.generate_riemannian_dynamic_topology(
            satellite_nodes, consistent_distance_graph, satellite_position, 
            ground_station_position, traffic_flow, num_isls
        )
        topology_graphs.append((time_stamp, topology_graph))

    filename = f"RiemannianDynamicTopology#{datetime.today().strftime('%Y_%m_%d,%H_%M_%S')}#{constellation_name}({num_orbits},{num_satellites})#{time_period}s(every){time_interval}s.csv"
    save_dynamic_topology_to_file(topology_graphs, satellite_nodes, f'./input/{filename}')
    
def riemannian_fields(cartesian_positions_file, source, destination, time_period, time_interval):
    splited_filename = cartesian_positions_file[:-4].split("#")
    constellation_name, orbital_structure = splited_filename[2].split("(")
    num_orbits, num_satellites = map(int, orbital_structure.rstrip(")").split(","))
    ground_station_positions, satellite_positions = rmtg.get_cartesian_positions(cartesian_positions_file, constellation_name)

    satellite_nodes = list(satellite_positions[0].keys())
    fields_over_time = {}
    for time_stamp in range(0, time_period*1000 + 1, time_interval*1000):
        satellite_position = satellite_positions[time_stamp]
        ground_station_position = ground_station_positions[time_stamp]
        fields = rmtg.calculate_fields_at_satellites(satellite_nodes, satellite_position, ground_station_position, source, destination, 10.0**7)
        fields_over_time[time_stamp] = fields

    filename = f"RiemannianFields#{datetime.today().strftime('%Y_%m_%d,%H_%M_%S')}#{constellation_name}({num_orbits},{num_satellites})#({source},{destination})#{time_period}s(every){time_interval}s.csv"
    save_fields_to_file(fields_over_time, f'./generated/{filename}')

def riemannian_static_topology(distance_file, cartesian_positions_file, num_isls):
    print("Riemannian Static Topology generation is not yet implemented.")
    # Placeholder for future implementation

def printHelp():
    print("topology_generator.py --help")
    print("topology_generator.py --random_static [distance_file] [number of ISLs]")
    print("topology_generator.py --riemannian_static [distance_file] [cartesian_positions_file] [demand_matrix_file] [number of ISLs]")
    print("topology_generator.py --riemannian_dynamic [distance_file] [cartesian_positions_file] [demand_matrix_file] [number of ISLs] [time_period(s)] [time_interval(s)]")
    print("topology_generator.py --riemannian_fields [cartesian_positions_file] [source] [destination] [time_period(s)] [time_interval(s)]")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Please provide an option as a command line argument!")
        printHelp()
        exit(1)
    
    if sys.argv[1] == "--help":
        printHelp()
    elif sys.argv[1] == "--random_static" and len(sys.argv) == 4:
        random_static_topology(sys.argv[2], int(sys.argv[3]))
    elif sys.argv[1] == "--riemannian_dynamic" and len(sys.argv) == 8:
        riemannian_dynamic_topology(sys.argv[2], sys.argv[3], sys.argv[4], int(sys.argv[5]), int(sys.argv[6]), int(sys.argv[7]))
    elif sys.argv[1] == "--riemannian_fields" and len(sys.argv) == 7:
        riemannian_fields(sys.argv[2], sys.argv[3], sys.argv[4], int(sys.argv[5]), int(sys.argv[6]))
    else:
        print("Invalid Option or Missing Arguments!")
        printHelp()
        exit(1)

    