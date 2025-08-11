import datetime, os, csv
import pandas as pd

def read_distance_file(filename):
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
    distance_csv_dataframe = pd.read_csv(f"./generated/{filename}")
    nodes = distance_csv_dataframe['FirstDeviceId'].unique().tolist()
    print(f"Read distance file '{filename}' with {len(nodes)} nodes, time step {time_step} ms, and total time {total_time} ms.")

    return distance_csv_dataframe, constellation_name, time_step, total_time, simulation_details, nodes, number_of_orbits, number_of_satellites_per_orbit

def forwarding_folder_csv_file(simulation_details, title, nodes):
    # making forwarding table output folder, name
    node_files = []
    node_writers = {}
    folder_name = f"{title}#{datetime.datetime.today().strftime('%Y_%m_%d,%H_%M_%S')}#{simulation_details}"
    if not os.path.exists("./forwarding_table"):
        os.makedirs("./forwarding_table")

    os.makedirs(f"./forwarding_table/{folder_name}")

    for node in nodes:
        file = open(f"./forwarding_table/{folder_name}/{node}.csv", "w", newline= "")
        csv_writer = csv.writer(file)
        csv_writer.writerow(["TimeStamp", "Destination", "NextHop"])
        node_files.append(file)
        node_writers[node] = csv_writer

    return node_files, node_writers

def close_files(node_files):
    for file in node_files:
        file.close()

def printHelp():
    print("shortest_path.py --help")
    print("shortest_path.py --dijkstra [distance file]")
    print("shortest_path.py --dijkstra_grid_plus [distance file]")
    print("shortest_path.py --dijkstra_static [distance file] [topology_file_static]")
    print("shortest_path.py --dijkstra_dynamic [distance file] [topology_file_dynamic]")