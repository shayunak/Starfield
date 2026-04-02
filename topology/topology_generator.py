import csv, sys, os
from datetime import datetime
import random_topology_generator as rtg
import riemannian_dynamic_topology_generator as rdtg
import riemannian_static_topology_generator as rstg
import consistent_distance_graph_generator as cdg
import riemann_fields_metrics as rfm

def save_static_topology_to_file(graph, nodes, filename):
    if not os.path.exists("./input"):
        os.makedirs("./input")
    with open(filename, 'w', newline='') as file:
        writer = csv.writer(file)
        writer.writerow(['FirstSatellite', 'SecondSatellite'])
        for node in nodes:
            for neighbor in graph.neighbors(node):
                writer.writerow([node, neighbor])

    print(f"Topology saved to {filename}")

def save_dynamic_topology_to_file(graphs, nodes, filename):
    if not os.path.exists("./input"):
        os.makedirs("./input")
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
            writer.writerow(['Timestamp', 'Satellite', 'Field_X', 'Field_Y', 'Field_Z', 'Field_Magnitude'])
        else:
            writer.writerow(['Satellite', 'Field_X', 'Field_Y', 'Field_Z', 'Field_Magnitude'])
        if len(fields) > 1:
            for time, field_data in fields:
                for i in range(len(field_data["Satellite"])):
                    writer.writerow([
                        time,
                        field_data["Satellite"][i],
                        field_data["Field_X"][i],
                        field_data["Field_Y"][i],
                        field_data["Field_Z"][i],
                        field_data["Field_Magnitude"][i]
                    ])
        else:
            field_data = fields[0]
            for i in range(len(field_data["Satellite"])):
                writer.writerow([
                    field_data["Satellite"][i],
                    field_data["Field_X"][i],
                    field_data["Field_Y"][i],
                    field_data["Field_Z"][i],
                    field_data["Field_Magnitude"][i]
                ])
    print(f"Fields saved to {filename}")

def random_static_topology(distance_file, num_isls):
    is_consistent_graph, df_graph, constellation_name, time_step, total_time, simulation_details, _, nodes, num_orbits, num_satellites = cdg.read_distance_file(distance_file)
    consistent_distance_graph, satellite_nodes = df_graph, nodes
    if not is_consistent_graph:
        consistent_distance_graph, satellite_nodes = cdg.get_static_consistent_distance_graph(df_graph, distance_file, nodes, constellation_name, time_step, total_time)

    topology_graph = rtg.generate_random_static_topology(satellite_nodes, num_orbits, num_satellites, constellation_name, consistent_distance_graph, num_isls)

    filename = f"RandomStaticTopology#{datetime.today().strftime('%Y_%m_%d,%H_%M_%S')}#{simulation_details}.csv"
    save_static_topology_to_file(topology_graph, satellite_nodes, f'./input/{filename}')
    
def riemannian_fields(cartesian_positions_file, source, destination, inclination, time_period, time_interval):
    splited_filename = cartesian_positions_file[:-4].split("#")
    constellation_name, orbital_structure = splited_filename[2].split("(")
    num_orbits, num_satellites = map(int, orbital_structure.rstrip(")").split(","))
    ground_station_positions, satellite_positions = rfm.get_cartesian_positions(cartesian_positions_file, constellation_name, time_period)

    satellite_nodes = list(satellite_positions[0].keys())
    fields_over_time = {}
    for time_stamp in range(0, time_period*1000 + 1, time_interval*1000):
        satellite_position = satellite_positions[time_stamp]
        ground_station_position = ground_station_positions[time_stamp]
        fields = rfm.calculate_fields_at_satellites(satellite_nodes, satellite_position, ground_station_position, source, destination, 10.0**6, inclination)
        fields_over_time[time_stamp] = fields

    filename = f"RiemannianFields#{datetime.today().strftime('%Y_%m_%d,%H_%M_%S')}#{constellation_name}({num_orbits},{num_satellites})#({source},{destination})#{time_period}s(every){time_interval}s.csv"
    save_fields_to_file(fields_over_time, f'./generated/{filename}')

def riemannian_static_topology(distance_file, cartesian_positions_file, demand_matrix_file, num_isls, inclination):
    is_consistent_graph, df_graph, constellation_name, time_step, total_time, _, _, nodes, num_orbits, num_satellites = cdg.read_distance_file(distance_file)
    consistent_distance_graph, satellite_nodes = df_graph, nodes
    if not is_consistent_graph:
        consistent_distance_graph, satellite_nodes = cdg.get_static_consistent_distance_graph(df_graph, distance_file, nodes, constellation_name, time_step, total_time)

    ground_station_positions, satellite_positions = rfm.get_cartesian_positions(cartesian_positions_file, constellation_name, total_time / 1000)
    _, avg_flows = rfm.get_flows_traffics(demand_matrix_file)
    initial_satellite_position = satellite_positions[0]
    initial_ground_station_position = ground_station_positions[0]

    topology_graph = rstg.generate_riemannian_static_topology(
        satellite_nodes, num_orbits, num_satellites, constellation_name, 
        consistent_distance_graph, initial_satellite_position, 
        initial_ground_station_position, avg_flows, num_isls, inclination
    )

    filename = f"RiemannianStaticTopology#{datetime.today().strftime('%Y_%m_%d,%H_%M_%S')}#{constellation_name}({num_orbits},{num_satellites}).csv"
    save_static_topology_to_file(topology_graph, satellite_nodes, f'./input/{filename}')

def riemannian_dynamic_topology(distance_file, cartesian_positions_file, demand_matrix_file, num_isls, inclination, time_period, time_interval):
    is_consistent_graph, df_graph, constellation_name, time_step, total_time, _, file_time, nodes, num_orbits, num_satellites = cdg.read_distance_file(distance_file)
    consistent_distance_graphs, satellite_nodes = df_graph, nodes
    if is_consistent_graph and (file_time != time_interval*1000 or total_time != time_period*1000):
        raise ValueError("The provided dynamic consistent distance graph does not match the specified time period and interval.")
    if not is_consistent_graph:
        consistent_distance_graphs, satellite_nodes = cdg.get_dynamic_consistent_distance_graphs(df_graph, nodes, constellation_name, num_orbits, num_satellites, file_time, time_step, time_interval, time_period)

    ground_station_positions, satellite_positions = rfm.get_cartesian_positions(cartesian_positions_file, constellation_name, time_period)
    flows_traffics, _ = rfm.get_flows_traffics(demand_matrix_file)
    avg_interval_flows = rfm.avg_flow_traffics(flows_traffics, time_interval * 1000, time_period * 1000)

    topology_graphs = []
    for time_stamp in range(0, time_period * 1000, time_interval * 1000):
        traffic_flow = avg_interval_flows[time_stamp]
        consistent_distance_graph = consistent_distance_graphs[time_stamp]
        middle_of_interval = time_stamp + (time_interval // 2) * 1000
        satellite_position = satellite_positions[middle_of_interval]
        ground_station_position = ground_station_positions[middle_of_interval]
        topology_graph = rdtg.generate_riemannian_dynamic_topology(
            satellite_nodes, consistent_distance_graph, satellite_position, 
            ground_station_position, traffic_flow, num_isls, inclination
        )
        topology_graphs.append((time_stamp, topology_graph))

    filename = f"RiemannianDynamicTopology#{datetime.today().strftime('%Y_%m_%d,%H_%M_%S')}#{constellation_name}({num_orbits},{num_satellites})#{time_period}s(every){time_interval}s.csv"
    save_dynamic_topology_to_file(topology_graphs, satellite_nodes, f'./input/{filename}')

def printHelp():
    print("topology_generator.py --help")
    print("topology_generator.py --random_static [distance_file] [number of ISLs]")
    print("topology_generator.py --riemannian_static [distance_file] [cartesian_positions_file] [demand_matrix_file] [number of ISLs] [inclination(deg)]")
    print("topology_generator.py --riemannian_dynamic [distance_file] [cartesian_positions_file] [demand_matrix_file] [number of ISLs] [inclination(deg)] [time_period(s)] [time_interval(s)]")
    print("topology_generator.py --riemannian_fields [cartesian_positions_file] [source] [destination] [inclination(deg)] [time_period(s)] [time_interval(s)]")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Please provide an option as a command line argument!")
        printHelp()
        exit(1)
    
    if sys.argv[1] == "--help":
        printHelp() 
    elif sys.argv[1] == "--random_static" and len(sys.argv) == 4:
        random_static_topology(sys.argv[2], int(sys.argv[3]))
    elif sys.argv[1] == "--riemannian_static" and len(sys.argv) == 7:
        riemannian_static_topology(sys.argv[2], sys.argv[3], sys.argv[4], int(sys.argv[5]), float(sys.argv[6]))
    elif sys.argv[1] == "--riemannian_dynamic" and len(sys.argv) == 9:
        riemannian_dynamic_topology(sys.argv[2], sys.argv[3], sys.argv[4], int(sys.argv[5]), float(sys.argv[6]), int(sys.argv[7]), int(sys.argv[8]))
    elif sys.argv[1] == "--riemannian_fields" and len(sys.argv) == 8:
        riemannian_fields(sys.argv[2], sys.argv[3], sys.argv[4], float(sys.argv[5]), int(sys.argv[6]), int(sys.argv[7]))
    else:
        print("Invalid Option or Missing Arguments!")
        printHelp()
        exit(1)

    