import sys
import pandas as pd
import numpy as np
import csv
from geopy.distance import geodesic

def read_ground_station_file(ground_station_file):
    ground_station_dataframe = pd.read_csv(f"./configs/{ground_station_file}")
    ground_stations = ground_station_dataframe['Id'].tolist()

    return ground_stations, len(ground_stations)

def create_output_file_single_traffic(distribution, source, destination, buffer_size, packet_length, packet_transmission_time, time_period):
        traffic_file = open(f"./input/{distribution}_{source}_to_{destination}_demand#{buffer_size}#{packet_length}Kb#{packet_transmission_time}ms#{time_period}s.csv", "w", newline= "")
        csv_writer = csv.writer(traffic_file)
        csv_writer.writerow(["Timestamp(ms)", "Source", "Destination", "Length(Mb)"])

        return csv_writer, traffic_file

def create_output_file(distribution, ground_station_file, buffer_size, packet_length, packet_transmission_time, time_period):
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

def generate_uniform_traffic(ground_station_file, buffer_size, packet_length, packet_transmission_time, time_period):
    ground_stations, number_of_ground_stations = read_ground_station_file(ground_station_file)
    largest_traffic = buffer_size * packet_length / (1000.0 * (number_of_ground_stations - 1))
    time_step = int(packet_transmission_time * buffer_size)
    csv_writer, traffic_file = create_output_file("uniform", ground_station_file, buffer_size, packet_length, packet_transmission_time, time_period)

    for time in range(0, time_period*1000, time_step):
        matrix = np.random.uniform(0.0, largest_traffic, size=(number_of_ground_stations, number_of_ground_stations))
        np.fill_diagonal(matrix, 0)
        rows = generate_rows(time, matrix, ground_stations, number_of_ground_stations)
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

def generate_distance_traffic(ground_station_file, buffer_size, packet_length, packet_transmission_time, time_period):
    # load ground station data include coordinates
    ground_station_dataframe = pd.read_csv(f"./configs/{ground_station_file}") 
    ground_stations = ground_station_dataframe['Id'].tolist()
    number_of_ground_stations = len(ground_stations)
    # store coordinates {Id: (Latitude, Longitude)}
    gs_coords = {}
    for index, row in ground_station_dataframe.iterrows():
        gs_coords[row['Id']] = (row['Latitude'], row['Longitude']) 
    #calculate D matrix
    D = np.zeros((number_of_ground_stations, number_of_ground_stations))
    for i in range(number_of_ground_stations):
        p1 = gs_coords[ground_stations[i]]
        for j in range(i + 1, number_of_ground_stations):
            p2 = gs_coords[ground_stations[j]]
            distance = geodesic(p1, p2).kilometers
            D[i, j] = distance
            D[j, i] = distance
    W = np.zeros_like(D, dtype=float)
    #W_ij = d_ij / (Σ d_ij)
    for j in range(number_of_ground_stations):
        pair_distance = D[:, j]
        total_distance = np.sum(pair_distance)
        if total_distance > 0:
            W[:, j] = pair_distance / total_distance        
    np.fill_diagonal(W, 0)

    largest_traffic = buffer_size * packet_length / (1000.0 * (number_of_ground_stations - 1)) 
    time_step = int(packet_transmission_time * buffer_size)
    csv_writer, traffic_file = create_output_file("distance_uniform", ground_station_file, buffer_size, packet_length, packet_transmission_time, time_period)
    
    for time in range(0, time_period * 1000, time_step):
        matrix = np.random.uniform(0.0, largest_traffic, size=(number_of_ground_stations, number_of_ground_stations)) 
        # F = Weight * matrix
        F = W * matrix
        np.fill_diagonal(F, 0)
        rows = generate_rows(time, F, ground_stations, number_of_ground_stations)
        csv_writer.writerows(rows)

    traffic_file.close()

def generate_population_traffic(ground_station_file, buffer_size, packet_length, packet_transmission_time, time_period):
    #load ground station data include population
    ground_station_dataframe = pd.read_csv(f"./configs/{ground_station_file}")
    ground_stations = ground_station_dataframe['Id'].tolist()
    number_of_ground_stations = len(ground_stations)
    #fetch population data
    Population = ground_station_dataframe['population'].fillna(0).to_numpy(dtype=float)
    #calculate Population Weight Matrix
    total_population_sum = np.sum(Population)
    squared_population_sum = np.sum(Population**2)
    total = total_population_sum**2 - squared_population_sum
    pair_population = np.outer(Population, Population)
    #calculate Weight matrix
    Weight = np.zeros((number_of_ground_stations, number_of_ground_stations), dtype=float)
    if total > 0:
        Weight = pair_population / total
    np.fill_diagonal(Weight, 0)

    largest_traffic = buffer_size * packet_length / (1000.0 * (number_of_ground_stations - 1))
    time_step = int(packet_transmission_time * buffer_size)
    csv_writer, traffic_file = create_output_file("population_uniform", ground_station_file, buffer_size, packet_length, packet_transmission_time, time_period)

    for time in range(0, time_period * 1000, time_step): 
        matrix = np.random.uniform(0.0, largest_traffic, size=(number_of_ground_stations, number_of_ground_stations))
        F = Weight * matrix
        np.fill_diagonal(F, 0)
        rows = generate_rows(time, F, ground_stations, number_of_ground_stations)
        csv_writer.writerows(rows)
        
    traffic_file.close()

def printHelp():    
    print("generate_traffic.py --help")
    print("generate_traffic.py --uniform [ground_station_file] [buffer_size] [packet_length(Kb)] [packet_transmission_time(ms)] [time_period(s)]")
    print("generate_traffic.py --distance [ground_station_file] [buffer_size] [packet_length(Kb)] [packet_transmission_time(ms)] [time_period(s)]")
    print("generate_traffic.py --population_uniform [ground_station_file] [buffer_size] [packet_length(Kb)] [packet_transmission_time(ms)] [time_period(s)]")
    print("generate_traffic.py --single_uniform [source] [destination] [buffer_size] [packet_length(Kb)] [packet_transmission_time(ms)] [time_period(s)]")
    

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Please provide an option as a command line argument!")
        printHelp()
        exit(1)
    
    if sys.argv[1] == "--help":
        printHelp()
    elif sys.argv[1] == "--uniform" and len(sys.argv) == 7:
        generate_uniform_traffic(sys.argv[2], int(sys.argv[3]), float(sys.argv[4]), int(sys.argv[5]), int(sys.argv[6]))
    elif sys.argv[1] == "--single_uniform" and len(sys.argv) == 8:
        generate_single_uniform_traffic(sys.argv[2], sys.argv[3], int(sys.argv[4]), float(sys.argv[5]), int(sys.argv[6]), int(sys.argv[7]))
    elif sys.argv[1] == "--distance" and len(sys.argv) == 7:
        generate_distance_traffic(sys.argv[2], int(sys.argv[3]), float(sys.argv[4]), int(sys.argv[5]), int(sys.argv[6]))
    elif sys.argv[1] == "--population" and len(sys.argv) == 7:  
        generate_population_traffic(sys.argv[2], int(sys.argv[3]), float(sys.argv[4]), int(sys.argv[5]), int(sys.argv[6]))
    else:
        print("Invalid Option or Missing Arguments!")
        printHelp()
        exit(1)