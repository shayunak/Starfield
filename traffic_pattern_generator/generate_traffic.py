import sys
import pandas as pd
import numpy as np
import csv

def read_ground_station_file(ground_station_file):
    ground_station_dataframe = pd.read_csv(f"./configs/{ground_station_file}")
    ground_stations = ground_station_dataframe['Id'].tolist()

    return ground_stations, len(ground_stations)

def create_output_file(distribution, ground_station_file, buffer_size, packet_length, packet_transmission_time, time_period):
    file_name_without_csv = ground_station_file[:-4]
    traffic_file = open(f"./input/{distribution}_demand#{file_name_without_csv}#{buffer_size}#{packet_length}Kb#{packet_transmission_time}ms#{time_period}s.csv", "w", newline= "")
    csv_writer = csv.writer(traffic_file)
    csv_writer.writerow(["Timestamp(ms)", "Source", "Destination", "Length(MB)"])

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

def printHelp():    
    print("shortest_path_algorithm.py --help")
    print("shortest_path_algorithm.py --uniform [ground_station_file] [buffer_size] [packet_length(KB)] [packet_transmission_time(ms)] [time_period(s)]")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Please provide an option as a command line argument!")
        printHelp()
        exit(1)
    
    if sys.argv[1] == "--help":
        printHelp()
    elif sys.argv[1] == "--uniform" and len(sys.argv) == 7:
        generate_uniform_traffic(sys.argv[2], int(sys.argv[3]), float(sys.argv[4]), int(sys.argv[5]), int(sys.argv[6]))
    else:
        print("Invalid Option or Missing Arguments!")
        printHelp()
        exit(1)