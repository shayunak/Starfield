from pathlib import Path
from geopy.distance import geodesic
import sys, os, csv
import pandas as pd
import numpy as np
import plot

SPEED_OF_LIGHT_VAC = 299.792458  # km/ms

# Command line help
def printHelp():
    print("analyze.py --help")
    print("analyze.py [simulation.csv] [groundstation.csv]")

# Calculate lantnecy
def packet_latency(group: pd.DataFrame) -> pd.Series:
    send_times = group.loc[group["Event"] == "SEND", "TimeStamp(ms)"]
    deliver_times = group.loc[group["Event"] == "DELIVERED", "TimeStamp(ms)"]
    if send_times.empty or deliver_times.empty:
        return pd.NA
    return deliver_times.min() - send_times.min()

def pairwise_latency(sim_df: pd.DataFrame) -> pd.Series:
    return sim_df.groupby("PacketId").apply(packet_latency, include_groups=False ).dropna().reset_index(name="Latency_ms")

def add_gs_pairs(sim_df: pd.DataFrame, perf_metrics_df: pd.DataFrame) -> pd.Series:
    send_map = (sim_df[sim_df["Event"] == "SEND"]
                .sort_values("TimeStamp(ms)")
                .drop_duplicates("PacketId")
                .set_index("PacketId")["FromDevice"])
    
    deliver_map = (sim_df[sim_df["Event"] == "DELIVERED"]
                   .sort_values("TimeStamp(ms)")
                   .drop_duplicates("PacketId")
                   .set_index("PacketId")["ToDevice"])
    
    perf_metrics_df["FromDevice"] = perf_metrics_df["PacketId"].map(send_map)
    perf_metrics_df["ToDevice"] = perf_metrics_df["PacketId"].map(deliver_map)

def pairwise_hop_count(sim_df: pd.DataFrame, perf_metrics_df: pd.DataFrame) -> pd.Series:
    hop_counts = sim_df.query('Event == "RECEIVE"').groupby("PacketId").size().rename("Hop_count")
    hop_counts = hop_counts.reindex(perf_metrics_df["PacketId"], fill_value=0)
    perf_metrics_df["Hop_count"] = perf_metrics_df["PacketId"].map(hop_counts)

def pairwise_stretch_factor(gs_df: pd.DataFrame, overall: pd.DataFrame) -> pd.Series:
    gs_coords = gs_df.set_index(gs_df.columns[0])[["Latitude", "Longitude"]].apply(tuple, axis=1).to_dict()
    def stretch_factor(row):
        if row["FromDevice"] == "ALL":
            return pd.NA
        p1 = gs_coords.get(row["FromDevice"])
        p2 = gs_coords.get(row["ToDevice"])
        if p1 is None or p2 is None:
            return pd.NA
        distance = geodesic(p1, p2).kilometers
        light_speed_latency = distance / SPEED_OF_LIGHT_VAC
        return light_speed_latency / row["Latency_ms"] if row["Latency_ms"] else pd.NA
    
    overall["Stretch_factor"] = overall.apply(stretch_factor, axis=1)

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



def show_results(overall: pd.DataFrame, number_of_dropped_packets, number_of_delivered_packets, throughput) -> None:
    print("Analysis Results:")
    display_cols = ["FromDevice", "ToDevice", "Latency_ms", "RTT_ms", "Avg_Hop", "Stretch_factor"]
    print(overall[display_cols].to_string(index=False))
    print("================================================================")

    print(f"Delivered packets count: {number_of_delivered_packets}")
    print(f"Dropped packet count: {number_of_dropped_packets}")
    print(f"Avg throughput: {throughput:.2f} pkt/s")
    print("================================================================")

# Analyze
def analyze(sim_csv: Path, gs_csv: Path) -> None:
    # Load CSV files
    sim_df = pd.read_csv(sim_csv)
    gs_df = pd.read_csv(gs_csv)

    # 1. Pair-wise latency
    perf_metrics_df = pairwise_latency(sim_df)

    # Find source and destination names
    add_gs_pairs(sim_df, perf_metrics_df)

    # 2. Calculate Hop Counts
    pairwise_hop_count(sim_df, perf_metrics_df)

    # 3. Averaging
    # Calculate average latency and hops for each link
    each_link_mean = (perf_metrics_df
                    .groupby(["FromDevice", "ToDevice"], as_index=False)
                    .agg(Latency_ms=("Latency_ms", "mean"),
                        Avg_Hop=("Hop_count", "mean")))
    
    # 4. Calculate average latency and hops for all links
    overall = (each_link_mean.sort_values(["FromDevice", "ToDevice"]).reset_index(drop=True))

    # 5. Calculate Stretch Factor
    pairwise_stretch_factor(gs_df, overall)

    # 6. Calculate RTT
    pairwise_rtt(overall)

    # 7. Starlink link usage
    satellite_df = pairwise_usage(sim_df)
    satellite_df.to_csv(results_folder / "link_usage.csv", index=False)

    # 8. Calculate average latency and hops for all links
    overall_mean = pd.DataFrame({
        "FromDevice": ["ALL"], "ToDevice": ["ALL"],
        "Latency_ms": [overall["Latency_ms"].mean()],
        "Avg_Hop": [overall["Avg_Hop"].mean()],
        "RTT_ms":          [overall["RTT_ms"].mean()],
        "Stretch_factor":  [overall["Stretch_factor"].mean()]})
    # 9. Combine all summary into overall
    overall = (pd.concat([overall, overall_mean], ignore_index=True).sort_values(["FromDevice", "ToDevice"]).reset_index(drop=True))

    # 10. Calculate dropped packets
    dropped_packets = sim_df.loc[sim_df["Event"] == "DROP", "PacketId"]

    # 11. Total Throughput
    delivered_set = set(sim_df.loc[sim_df["Event"] == "DELIVERED", "PacketId"])
    duration_s = (sim_df["TimeStamp(ms)"].max() - sim_df["TimeStamp(ms)"].min()) / 1000
    throughput = len(delivered_set) / duration_s

    # 12. Print Results
    show_results(overall, len(dropped_packets), len(delivered_set), throughput)

    return overall



if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Error: Need exactly two arguments!")
        printHelp()
        sys.exit(1)

    sim_csv_name = sys.argv[1]
    gs_csv_name  = sys.argv[2]

    # Absolute csv file path
    repo_root = Path(__file__).resolve().parent.parent

    sim_csv_path = repo_root / "generated" / sim_csv_name
    gs_csv_path  = repo_root / "configs"  / gs_csv_name
    results_folder = repo_root / "results" / f'AnalysisOf{sim_csv_name}'

    os.makedirs(results_folder, exist_ok=True)

    overall = analyze(sim_csv_path, gs_csv_path)
    overall.to_csv(results_folder / "overall.csv", index=False)
    
    plot.generate_plots(overall, results_folder)