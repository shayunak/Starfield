import sys, os
import pandas as pd
import numpy as np
import csv
from geopy.distance import geodesic

def read_ground_station_file(ground_station_file):
    ground_station_dataframe = pd.read_csv(f"./configs/{ground_station_file}")
    ground_stations = ground_station_dataframe['Id'].tolist()

    return ground_stations, len(ground_stations)

def read_ground_station_locs(ground_station_file):
    ground_station_dataframe = pd.read_csv(f"./configs/{ground_station_file}")
    ground_stations = ground_station_dataframe['Id'].tolist()
    # store coordinates {Id: (Latitude, Longitude)}
    gs_coords = {}
    for _, row in ground_station_dataframe.iterrows():
        gs_coords[row['Id']] = (row['Latitude'], row['Longitude'])

    return gs_coords, ground_stations, len(ground_stations)

def read_ground_station_population_file(ground_station_population_file):
    ground_station_dataframe = pd.read_csv(f"./configs/{ground_station_population_file}")
    ground_stations = ground_station_dataframe['Id'].tolist()
    population = ground_station_dataframe['Population'].fillna(0).to_numpy(dtype=float)

    return population, ground_stations, len(ground_stations)

def create_output_file_single_traffic(distribution, source, destination, buffer_size, packet_length, packet_transmission_time, time_period):
    if not os.path.exists("./input"):
        os.makedirs("./input")
    traffic_file = open(f"./input/{distribution}_{source}_to_{destination}_demand#{buffer_size}#{packet_length}Kb#{packet_transmission_time}ms#{time_period}s.csv", "w", newline= "")
    csv_writer = csv.writer(traffic_file)
    csv_writer.writerow(["Timestamp(ms)", "Source", "Destination", "Length(Mb)"])

    return csv_writer, traffic_file

def create_output_file(distribution, ground_station_file, buffer_size, packet_length, packet_transmission_time, time_period):
    if not os.path.exists("./input"):
        os.makedirs("./input")
    file_name_without_csv = ground_station_file[:-4]
    traffic_file = open(f"./input/{distribution}_demand#{file_name_without_csv}#{buffer_size}#{packet_length}Kb#{packet_transmission_time}ms#{time_period}s.csv", "w", newline= "")
    csv_writer = csv.writer(traffic_file)
    csv_writer.writerow(["Timestamp(ms)", "Source", "Destination", "Length(Mb)"])

    return csv_writer, traffic_file

def generate_rows(time, matrix, ground_stations, number_of_ground_stations):
    rows = []
    for i in range(0, number_of_ground_stations):
        for j in range(0, number_of_ground_stations):
            if matrix[i][j] > 0:
                rows.append([time, ground_stations[i], ground_stations[j], matrix[i][j]])

    return rows

def generate_random_traffic(ground_stations, number_of_ground_stations, ground_station_file_name, weight_matrix, distribution, buffer_size, packet_length, packet_transmission_time, time_period):
    largest_traffic = buffer_size * packet_length / (1000.0 * (number_of_ground_stations - 1))
    time_step = int(packet_transmission_time * buffer_size)
    csv_writer, traffic_file = create_output_file(distribution, ground_station_file_name, buffer_size, packet_length, packet_transmission_time, time_period)

    for time in range(0, time_period*1000, time_step):
        uniform_matrix = np.random.uniform(0.0, largest_traffic, size=(number_of_ground_stations, number_of_ground_stations))
        demand_matrix = weight_matrix * uniform_matrix
        np.fill_diagonal(demand_matrix, 0)
        rows = generate_rows(time, demand_matrix, ground_stations, number_of_ground_stations)
        csv_writer.writerows(rows)

    traffic_file.close()

def generate_single_uniform_traffic(source, destination, buffer_size, packet_length, packet_transmission_time, time_period):
    largest_traffic = buffer_size * packet_length / 1000.0
    time_step = int(packet_transmission_time * buffer_size)
    csv_writer, traffic_file = create_output_file_single_traffic("uniform", source, destination, buffer_size, packet_length, packet_transmission_time, time_period)

    for time in range(0, time_period*1000, time_step):
        traffic = np.random.uniform(0.0, largest_traffic)
        csv_writer.writerow([time, source, destination, traffic])

    traffic_file.close()

def generate_uniform_traffic(ground_station_file, buffer_size, packet_length, packet_transmission_time, time_period):
    distribution = "uniform"
    ground_stations, number_of_ground_stations = read_ground_station_file(ground_station_file)
    weight_matrix = np.ones((number_of_ground_stations, number_of_ground_stations))
    generate_random_traffic(ground_stations, number_of_ground_stations, ground_station_file, weight_matrix, distribution, buffer_size, packet_length, packet_transmission_time, time_period)

def generate_exponential_hotspot_traffic(ground_station_file, buffer_size, packet_length, packet_transmission_time, time_period, decay_param=None):
    distribution = "exponential_hotspot"
    ground_stations, number_of_ground_stations = read_ground_station_file(ground_station_file)
    # Create an exponential hotspot weight matrix
    col = np.arange(number_of_ground_stations).reshape(-1, 1)
    row = np.arange(number_of_ground_stations).reshape(1, -1)

    if decay_param is None:
        decay_param = 1 / number_of_ground_stations

    W = np.exp(-decay_param * (col + row))
    W = W / np.sum(W)
    np.fill_diagonal(W, 0)
    W = W * buffer_size

    generate_random_traffic(ground_stations, number_of_ground_stations, ground_station_file, W, distribution, buffer_size, packet_length, packet_transmission_time, time_period)

def generate_distance_traffic(ground_station_file, buffer_size, packet_length, packet_transmission_time, time_period):
    distribution = "distance"
    gs_coords, ground_stations, number_of_ground_stations = read_ground_station_locs(ground_station_file)
    #calculate distance matrix
    D = np.zeros((number_of_ground_stations, number_of_ground_stations))
    for i in range(number_of_ground_stations):
        p1 = gs_coords[ground_stations[i]]
        for j in range(i + 1, number_of_ground_stations):
            p2 = gs_coords[ground_stations[j]]
            distance = geodesic(p1, p2).kilometers
            D[i, j] = distance
            D[j, i] = distance
    
    W = D / np.sum(D)
    np.fill_diagonal(W, 0)
    W = W * buffer_size
    generate_random_traffic(ground_stations, number_of_ground_stations, ground_station_file, W, distribution, buffer_size, packet_length, packet_transmission_time, time_period)

def generate_population_traffic(ground_station_population_file, buffer_size, packet_length, packet_transmission_time, time_period):
    distribution = "population"
    population, ground_stations, number_of_ground_stations = read_ground_station_population_file(ground_station_population_file)
    
    pair_population = np.outer(population, population)
    total = np.sum(pair_population)
    W = np.zeros((number_of_ground_stations, number_of_ground_stations), dtype=float)
    if total > 0:
        W = pair_population / total
    np.fill_diagonal(W, 0)
    W = W * buffer_size
    generate_random_traffic(ground_stations, number_of_ground_stations, ground_station_population_file, W, distribution, buffer_size, packet_length, packet_transmission_time, time_period)

def distort_traffic_gaussian(demand_file, packet_size, mean, stddev):
    demand_df = pd.read_csv(f"./input/{demand_file}")
    gaussian_noise = np.random.normal(loc=mean, scale=stddev, size=len(demand_df))
    demand_df['Length(Mb)'] = np.clip(demand_df['Length(Mb)'] + gaussian_noise * packet_size / 1000.0, 0, None)
    output_file = f"distorted_gaussian({mean},{stddev})_{demand_file}"
    demand_df.to_csv(f"./input/{output_file}", index=False)

def generate_distance_population_traffic(ground_station_population_file, buffer_size, packet_length, packet_transmission_time, time_period):
    distribution = "distance_population"
    gs_coords, ground_stations, number_of_ground_stations = read_ground_station_locs(ground_station_population_file)
    population, _, _ = read_ground_station_population_file(ground_station_population_file)
    #Weight by distance
    D = np.zeros((number_of_ground_stations, number_of_ground_stations))
    for i in range(number_of_ground_stations):
        p1 = gs_coords[ground_stations[i]]
        for j in range(i + 1, number_of_ground_stations):
            p2 = gs_coords[ground_stations[j]]
            distance = geodesic(p1, p2).kilometers
            D[i, j] = distance
            D[j, i] = distance
    
    W_D = D / np.sum(D)
    np.fill_diagonal(W_D, 0)
    # Weight by population
    pair_population = np.outer(population, population)
    total = np.sum(pair_population)
    W_P = np.zeros((number_of_ground_stations, number_of_ground_stations), dtype=float)
    if total > 0:
        W_P = pair_population / total
    np.fill_diagonal(W_P, 0)

    W = W_D + W_P
    W = W * buffer_size
    generate_random_traffic(ground_stations, number_of_ground_stations, ground_station_population_file, W, distribution, buffer_size, packet_length, packet_transmission_time, time_period)

def printHelp():    
    print("generate_traffic.py --help")
    print("generate_traffic.py --single_uniform [source] [destination] [buffer_size] [packet_length(KB)] [packet_transmission_time(ms)] [time_period(s)]")
    print("generate_traffic.py --uniform [ground_station_file] [buffer_size] [packet_length(KB)] [packet_transmission_time(ms)] [time_period(s)]")
    print("generate_traffic.py --exponential_hotspot [ground_station_file] [buffer_size] [packet_length(KB)] [packet_transmission_time(ms)] [time_period(s)] ([decay_param])")
    print("generate_traffic.py --distance [ground_station_file] [buffer_size] [packet_length(KB)] [packet_transmission_time(ms)] [time_period(s)]")
    print("generate_traffic.py --population [ground_station_population_file] [buffer_size] [packet_length(KB)] [packet_transmission_time(ms)] [time_period(s)]")
    print("generate_traffic.py --distance_population [ground_station_population_file] [buffer_size] [packet_length(KB)] [packet_transmission_time(ms)] [time_period(s)]")
    print("generate_traffic.py --distort_gaussian [demand_file] [packet_size(KB)] [mean(mu)] [stddev(sigma)]")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Please provide an option as a command line argument!")
        printHelp()
        exit(1)
    
    if sys.argv[1] == "--help":
        printHelp()
    elif sys.argv[1] == "--single_uniform" and len(sys.argv) == 8:
        generate_single_uniform_traffic(sys.argv[2], sys.argv[3], int(sys.argv[4]), float(sys.argv[5]), int(sys.argv[6]), int(sys.argv[7]))
    elif sys.argv[1] == "--uniform" and len(sys.argv) == 7:
        generate_uniform_traffic(sys.argv[2], int(sys.argv[3]), float(sys.argv[4]), int(sys.argv[5]), int(sys.argv[6]))
    elif sys.argv[1] == "--exponential_hotspot" and len(sys.argv) == 7:
        generate_exponential_hotspot_traffic(sys.argv[2], int(sys.argv[3]), float(sys.argv[4]), int(sys.argv[5]), int(sys.argv[6]))
    elif sys.argv[1] == "--exponential_hotspot" and len(sys.argv) == 8:
        generate_exponential_hotspot_traffic(sys.argv[2], int(sys.argv[3]), float(sys.argv[4]), int(sys.argv[5]), int(sys.argv[6]), float(sys.argv[7]))
    elif sys.argv[1] == "--distance" and len(sys.argv) == 7:
        generate_distance_traffic(sys.argv[2], int(sys.argv[3]), float(sys.argv[4]), int(sys.argv[5]), int(sys.argv[6]))
    elif sys.argv[1] == "--population" and len(sys.argv) == 7:  
        generate_population_traffic(sys.argv[2], int(sys.argv[3]), float(sys.argv[4]), int(sys.argv[5]), int(sys.argv[6]))
    elif sys.argv[1] == "--distance_population" and len(sys.argv) == 7:  
        generate_distance_population_traffic(sys.argv[2], int(sys.argv[3]), float(sys.argv[4]), int(sys.argv[5]), int(sys.argv[6]))
    elif sys.argv[1] == "--distort_gaussian" and len(sys.argv) == 6:
        distort_traffic_gaussian(sys.argv[2], float(sys.argv[3]), float(sys.argv[4]), float(sys.argv[5]))
    else:
        print("Invalid Option or Missing Arguments!")
        printHelp()
        exit(1)