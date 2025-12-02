from pathlib import Path
from geopy.distance import geodesic
import sys, os, csv
import pandas as pd
import plot

SPEED_OF_LIGHT_VAC = 299792458.0  # m/s

def get_distance_time_detail(distance_file, max_sim_time):
    splited_filename = distance_file[:-4].split("#")
    if len(splited_filename) != 5:
        raise NameError("Incorrect distance file name format!")
    if splited_filename[0] != "Distances":
        raise NameError("Only distances files are accepted, and they start with 'Distances'!")
    time_step = int(splited_filename[3][:-2])
    total_time = int(splited_filename[4][:-1]) * 1000

    last_dist_time = (max_sim_time // time_step) * time_step
    if last_dist_time > total_time:
        raise ValueError("The provided max simulation time exceeds the total time in the distance file!")

    return last_dist_time, time_step

def extract_distance_time_limit(distance_file, last_dist_time):
    distances_dict = {}
    with open(distance_file, "r", newline="") as file:
        reader = csv.reader(file)
        next(reader)  # Skip header
        for row in reader:
            time_stamp = int(row[0])
            first_device_id = row[1]
            second_device_id = row[2]
            if time_stamp > last_dist_time:
                break
            distances_dict[(time_stamp, first_device_id, second_device_id)] = float(row[3])
        
    return distances_dict

# Calculate lantnecy
def packet_latency(group: pd.DataFrame) -> pd.Series:
    send_times = group.loc[group["Event"] == "SEND", "TimeStamp(ms)"]
    deliver_times = group.loc[group["Event"] == "DELIVERED", "TimeStamp(ms)"]
    if send_times.empty or deliver_times.empty:
        return pd.NA
    return deliver_times.min() - send_times.min()

def pairwise_latency(sim_df: pd.DataFrame) -> pd.Series:
    return sim_df.groupby("PacketId").apply(packet_latency, include_groups=False).dropna().reset_index(name="Latency_ms")

def add_gs_pairs(sim_df: pd.DataFrame, perf_metrics_df: pd.DataFrame, gs_df: pd.DataFrame) -> pd.Series:
    gs_coords = gs_df.set_index(gs_df.columns[0])[["Latitude", "Longitude"]].apply(tuple, axis=1).to_dict()
    send_map = (sim_df[sim_df["Event"] == "DELIVERED"]
                .sort_values("TimeStamp(ms)")
                .drop_duplicates("PacketId")
                .set_index("PacketId")["FromDevice"])
    
    deliver_map = (sim_df[sim_df["Event"] == "DELIVERED"]
                   .sort_values("TimeStamp(ms)")
                   .drop_duplicates("PacketId")
                   .set_index("PacketId")["ToDevice"])
    
    perf_metrics_df["FromDevice"] = perf_metrics_df["PacketId"].map(send_map)
    perf_metrics_df["ToDevice"] = perf_metrics_df["PacketId"].map(deliver_map)
    perf_metrics_df["Geodesic"] = perf_metrics_df.apply(lambda row: geodesic(gs_coords.get(row["FromDevice"]), gs_coords.get(row["ToDevice"])).meters, axis=1)

def pairwise_hop_count(sim_df: pd.DataFrame, perf_metrics_df: pd.DataFrame) -> pd.Series:
    hop_counts = sim_df.query('Event == "RECEIVE"').groupby("PacketId").size().rename("Hop_Count")
    hop_counts = hop_counts.reindex(perf_metrics_df["PacketId"], fill_value=0)
    perf_metrics_df["Hop_Count"] = perf_metrics_df["PacketId"].map(hop_counts)

def distance_at_time(group: pd.DataFrame, distances, time_step) -> pd.Series:
    group["TimeStamp(ms)"] = (group["TimeStamp(ms)"] // time_step) * time_step
    return group.apply(lambda r: distances.get((r["TimeStamp(ms)"], r["FromDevice"], r["ToDevice"]), pd.NA),axis=1).sum()

def pairwise_stretch_factor(sim_df: pd.DataFrame, distances: dict, perf_metrics_df: pd.DataFrame, time_step: int) -> pd.Series:
    distances = sim_df.query('Event == "RECEIVE"').groupby("PacketId").apply(lambda g: distance_at_time(g, distances, time_step), include_groups=False).dropna().rename("Distance")
    distances = distances.reindex(perf_metrics_df["PacketId"], fill_value=0)
    perf_metrics_df["Stretch_Factor"] = perf_metrics_df["PacketId"].map(distances) / perf_metrics_df["Geodesic"]

#Calculate city pair RTT
def pairwise_rtt(overall_df: pd.DataFrame) -> None:
    df = overall_df.copy()
    mask = (df["FromDevice"] != "ALL") & (df["FromDevice"] != df["ToDevice"])
    df = df[mask]
    df["city_pair"] = df.apply(lambda row: frozenset([row["FromDevice"], row["ToDevice"]]), axis=1)
    pair_rtt = (df.groupby("city_pair")["Latency_ms"].sum().rename("RTT_ms"))
    overall_df["RTT_ms"] = overall_df.apply(lambda row: pair_rtt.get(frozenset([row["FromDevice"], row["ToDevice"]]), pd.NA),axis=1)

#Calculate Starlink link usage
def pairwise_usage(sim_df: pd.DataFrame) -> pd.DataFrame:
    df = sim_df.loc[sim_df["Event"] == "RECEIVE"].dropna(subset=["FromDevice", "ToDevice"])
    def is_satellite(name: str) -> bool:
        return isinstance(name, str) and name.startswith('Starlink')
    df = df[df["FromDevice"].apply(is_satellite) & df["ToDevice"].apply(is_satellite)]
    df["link"] = df.apply(lambda row: tuple(sorted([row["FromDevice"].strip(), row["ToDevice"].strip()])),axis=1)
    satellite_df = (df.groupby("link")["PacketId"].nunique().reset_index().rename(columns={"PacketId": "UsageCount"}).sort_values("UsageCount", ascending=False).reset_index(drop=True))
    satellite_df[["SatelliteA", "SatelliteB"]] = pd.DataFrame(satellite_df["link"].tolist(),index=satellite_df.index)
    satellite_df = satellite_df[["link","SatelliteA", "SatelliteB", "UsageCount"]]
    return satellite_df

#Calculate network jitter
def pairwise_jitter(perf_df: pd.DataFrame, overall: pd.DataFrame) -> None:
    jitter_df = (perf_df.groupby(["FromDevice", "ToDevice"])["Latency_ms"].std(ddof=0).rename("Jitter").reset_index())
    return overall.merge(jitter_df, on=["FromDevice", "ToDevice"],how="left", copy=False)


def show_results(overall: pd.DataFrame, number_of_dropped_packets, number_of_delivered_packets, throughput) -> None:
    print("Analysis Results:")
    display_cols = ["FromDevice", "ToDevice", "Latency_ms", "RTT_ms", "Hop_Count", "Stretch_Factor", "Effective_Latency_Factor", "Jitter", "Total_Packets"]
    print(overall[display_cols].to_string(index=False))
    print("================================================================")

    print(f"Delivered packets count: {number_of_delivered_packets}")
    print(f"Dropped packet count: {number_of_dropped_packets}")
    print(f"Avg throughput: {throughput:.2f} pkt/s")
    print("================================================================")

# Analyze
def analyze(sim_csv: Path, gs_csv: Path, distance_csv: Path) -> None:
    # Load CSV files
    sim_df = pd.read_csv(sim_csv)
    gs_df = pd.read_csv(gs_csv)

    # 1. Pair-wise latency
    perf_metrics_df = pairwise_latency(sim_df)
    print("Pairwise latency calculated.")

    # Find source and destination names
    add_gs_pairs(sim_df, perf_metrics_df, gs_df)

    # 2. Calculate Hop Counts
    pairwise_hop_count(sim_df, perf_metrics_df)
    print("Pairwise hop counts calculated.")

    #3. Calculate Stretch Factor
    max_sim_time = sim_df["TimeStamp(ms)"].max()
    last_dist_time, time_step = get_distance_time_detail(distance_csv.name, max_sim_time)
    distances = extract_distance_time_limit(distance_csv, last_dist_time)
    pairwise_stretch_factor(sim_df, distances, perf_metrics_df, time_step)
    print("Pairwise stretch factors calculated.")

    # 3. Averaging
    # Calculate average latency and hops for each link
    each_link_mean = (perf_metrics_df
                    .groupby(["FromDevice", "ToDevice"], as_index=False)
                    .agg(Latency_ms=("Latency_ms", "mean"),
                        Hop_Count=("Hop_Count", "mean"),
                        Stretch_Factor=("Stretch_Factor", "mean"),
                        Geodesic=("Geodesic", "mean"),
                        Total_Packets=("Latency_ms", "count")
                        )
                    )
    
    # 4. Calculate average latency, hops, and stretch factor for all links
    overall = (each_link_mean.sort_values(["FromDevice", "ToDevice"]).reset_index(drop=True))
    print("Per link aggregation Calculated.")


    # 5. Calculate Effective Latency Factor
    overall["Effective_Latency_Factor"] = (overall["Latency_ms"] * SPEED_OF_LIGHT_VAC) / (1000.0 * overall["Geodesic"] * overall["Stretch_Factor"])
    overall.drop(columns=["Geodesic"], inplace=True)
    print("Effective latency factor calculated.")

    # 6. Calculate RTT
    pairwise_rtt(overall)
    print("RTT calculated.")

    # 7. Starlink link usage
    satellite_df = pairwise_usage(sim_df)
    satellite_df.to_csv(results_folder / "link_usage.csv", index=False)
    print("ISL link usage calculated.")

    # 8. jitter
    overall = pairwise_jitter(perf_metrics_df, overall)
    print("Jitter calculated.")
    
    # 9. Calculate average latency and hops for all links
    overall["Total_Packets"] = overall.pop("Total_Packets")
    packet_nums = overall["Total_Packets"]
    total_packets = packet_nums.sum()
    overall_mean = pd.DataFrame({
        "FromDevice": ["ALL"], "ToDevice": ["ALL"],
        "Latency_ms": [(overall["Latency_ms"] * packet_nums).sum() / total_packets],
        "Hop_Count": [(overall["Hop_Count"] * packet_nums).sum() / total_packets],
        "Stretch_Factor": [(overall["Stretch_Factor"] * packet_nums).sum() / total_packets],
        "Effective_Latency_Factor":  [(overall["Effective_Latency_Factor"] * packet_nums).sum() / total_packets],
        "RTT_ms": [(overall["RTT_ms"] * packet_nums).sum() / total_packets],
        "Jitter": [(overall["Jitter"] * packet_nums).sum() / total_packets],
        "Total_Packets": [total_packets]
        }
    )
    
    # 10. Combine all summary into overall
    overall = (pd.concat([overall, overall_mean], ignore_index=True).sort_values(["FromDevice", "ToDevice"]).reset_index(drop=True))
    print("Mean values calculated.")

    # 11. Calculate dropped packets
    dropped_packets = sim_df.loc[sim_df["Event"] == "DROP", "PacketId"]

    # 12. Total Throughput
    delivered_set = set(sim_df.loc[sim_df["Event"] == "DELIVERED", "PacketId"])
    duration_s = (sim_df["TimeStamp(ms)"].max() - sim_df["TimeStamp(ms)"].min()) / 1000
    throughput = len(delivered_set) / duration_s
    print("Dropped Packets and throughput calculated.")

    # 13. Print Results
    show_results(overall, len(dropped_packets), len(delivered_set), throughput)

    return overall

# Command line help
def printHelp():
    print("analyze.py --help")
    print("analyze.py --analyze [simulation_summary_file.csv] [ground_stations_file.csv] [distances_file.csv]")
    print("analyze.py --combine [name_of_combination] [<overall_folder1> ...] [<overall_name1> ...]")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Please provide an option as a command line argument!")
        printHelp()
        exit(1)
    if sys.argv[1] == "--help":
        printHelp()
        exit(1)
    elif sys.argv[1] == "--analyze" and len(sys.argv) == 5:
        sim_csv_name = sys.argv[2]
        gs_csv_name  = sys.argv[3]
        distance_csv_name = sys.argv[4]

        # Absolute csv file path
        repo_root = Path(__file__).resolve().parent.parent

        sim_csv_path = repo_root / "generated" / sim_csv_name
        distance_csv_path = repo_root / "generated" / distance_csv_name
        gs_csv_path  = repo_root / "configs"  / gs_csv_name
        results_folder = repo_root / "results" / f'AnalysisOf{sim_csv_name[:-4]}'

        os.makedirs(results_folder, exist_ok=True)

        overall = analyze(sim_csv_path, gs_csv_path, distance_csv_path)
        overall.to_csv(results_folder / "overall.csv", index=False)
        
        plot.generate_plots(overall, results_folder)
    elif sys.argv[1] == "--combine" and len(sys.argv) >= 3:
        comb_name = sys.argv[2]
        file_args = sys.argv[3:]
        if len(file_args) % 2 != 0:
            print("Please provide an even number of arguments for combining overall files and their names!")
            printHelp()
            exit(1)
        if len(file_args) < 4:
            print("At least two overall files are required for combination!")
            printHelp()
            exit(1)
        
        repo_root = Path(__file__).resolve().parent.parent

        overall_folders = [repo_root / "results" / f for f in file_args[:len(file_args)//2]]
        overall_names = file_args[len(file_args)//2:]
        overall_dfs = [pd.read_csv(f / "overall.csv") for f in overall_folders]
        link_usage_dfs = [pd.read_csv(f / "link_usage.csv") for f in overall_folders]
        overalls = list(zip(overall_names, overall_dfs, link_usage_dfs))

        combination_folder = repo_root / "results" / f'CombinedAnalysisOf_{comb_name}_{"_".join(overall_names)}'

        os.makedirs(combination_folder, exist_ok=True)

        plot.combine_overalls(overalls, combination_folder, comb_name)
    else:
        print("Invalid Option or Missing Arguments!")
        printHelp()
        exit(1)
