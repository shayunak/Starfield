from pathlib import Path
from geopy.distance import geodesic
import sys, os, csv
import pandas as pd
import matplotlib.pyplot as plt
import numpy as np


# Command line help
def printHelp():
    print("log_analyzer/analyze.py --help")
    print("  python analyze.py <simulation.csv> <groundstation.csv>")
    print("\nArguments:")
    print("  <simulation.csv>     will be in generated/")
    print("  <groundstation.csv>  will be in configs/")



# Calculate lantnecy
def packet_latency(group: pd.DataFrame) -> pd.Series:
    """Return latency (ms) for one PacketId group."""
    send_times = group.loc[group["Event"] == "SEND", "TimeStamp(ms)"]
    deliver_times = group.loc[group["Event"] == "DELIVERED", "TimeStamp(ms)"]
    if send_times.empty or deliver_times.empty:
        return pd.NA
    return deliver_times.min() - send_times.min()

# Analyze
def analyze(sim_csv: Path, gs_csv: Path) -> None:
    # Load CSV files
    df = pd.read_csv(sim_csv)
    gs_df = pd.read_csv(gs_csv)

    # Average latency
    temp_df = (
        df.groupby("PacketId")
          .apply(packet_latency)
          .dropna()
          .reset_index(name="Latency_ms")
    )

    # Find source and destination places
    send_map = (df[df["Event"] == "SEND"]
                .sort_values("TimeStamp(ms)")
                .drop_duplicates("PacketId")
                .set_index("PacketId")["FromDevice"])
    deliver_map = (df[df["Event"] == "DELIVERED"]
                   .sort_values("TimeStamp(ms)")
                   .drop_duplicates("PacketId")
                   .set_index("PacketId")["ToDevice"])
    temp_df["FromDevice"] = temp_df["PacketId"].map(send_map)
    temp_df["ToDevice"] = temp_df["PacketId"].map(deliver_map)

    # 2. Calculate Hop Counts
    hop_counts = (df.query('Event == "RECEIVE"')
                    .groupby("PacketId").size()
                    .rename("Hop_count"))
    hop_counts = hop_counts.reindex(temp_df["PacketId"], fill_value=0)
    temp_df["Hop_count"] = temp_df["PacketId"].map(hop_counts)

    # Calculate average latency and hops for each link
    each_link_mean = (temp_df
                    .groupby(["FromDevice", "ToDevice"], as_index=False)
                    .agg(Latency_ms=("Latency_ms", "mean"),
                        Avg_Hop=("Hop_count", "mean")))
    # Calculate average latency and hops for all links
    overall_mean = pd.DataFrame({
        "FromDevice": ["ALL"], "ToDevice": ["ALL"],
        "Latency_ms": [temp_df["Latency_ms"].mean()],
        "Avg_Hop": [temp_df["Hop_count"].mean()]})

    # Combine all summary into overall
    overall = (pd.concat([each_link_mean, overall_mean], ignore_index=True)
                         .sort_values(["FromDevice", "ToDevice"])
                         .reset_index(drop=True))

    # Calculate Stretch Factor
    gs_coords = gs_df.set_index(gs_df.columns[0])[["Latitude", "Longitude"]].apply(tuple, axis=1).to_dict()
    speed_of_light_vac = 299.792458  # km/ms
    def stretch_factor(row):
        if row["FromDevice"] == "ALL":
            return pd.NA
        p1 = gs_coords.get(row["FromDevice"])
        p2 = gs_coords.get(row["ToDevice"])
        if p1 is None or p2 is None:
            return pd.NA
        distance = geodesic(p1, p2).kilometers
        light_speed_latency = distance / speed_of_light_vac
        return light_speed_latency / row["Latency_ms"] if row["Latency_ms"] else pd.NA
    # Add stretch factor to latency summary
    overall["Stretch_factor"] = overall.apply(stretch_factor, axis=1)

    # Show results
    print("Summary:")
    display_cols = ["FromDevice", "ToDevice", "Latency_ms", "Avg_Hop", "Stretch_factor"]
    print(overall[display_cols].to_string(index=False))
    print("================================================================")

    # Throughput & drops
    delivered_set = set(df.loc[df["Event"] == "DELIVERED", "PacketId"])
    duration_s = (df["TimeStamp(ms)"].max() - df["TimeStamp(ms)"].min()) / 1000
    throughput = len(delivered_set) / duration_s
    dropped_packets = df.loc[df["Event"] == "DROP", "PacketId"]
    
    print(f"Delivered packets from source to destination: {len(delivered_set)}")
    print(f"Dropped packet count: {len(dropped_packets)}")
    print(f"Avg throughput: {throughput:.2f} pkt/s")
    print("================================================================")
    

    # Plotting
    # Delete 'ALL→ALL' and NaN
    stretch_values = overall.loc[
        overall["FromDevice"] != "ALL", "Stretch_factor"
    ].dropna().values
    # Get average hop
    hop_values = overall["Avg_Hop"].dropna().values

    def empirical_cdf(arr):
        # Sort values and compute CDF
        x = np.sort(arr)
        y = np.arange(1, len(x)+1) / len(x)
        return x, y

    sort_x, sort_cdf = empirical_cdf(stretch_values)
    hop_x, hop_cdf = empirical_cdf(hop_values)

    # Draw the CDF plot
    plt.figure()
    plt.plot(sort_x, sort_cdf, linestyle='-',  label="Stretch factor CDF")
    plt.plot(hop_x, hop_cdf, linestyle='--', label="Hop count CDF")
    plt.xlabel("Stretch factor or hop count")
    plt.ylabel("CDF across links")
    plt.title("CDF of Stretch & Hop")
    plt.legend()
    plt.grid(True)
    plt.show()


# CLI entry point
if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Error: Need exactly two arguments!\n")
        printHelp()
        sys.exit(1)

    sim_csv_name = sys.argv[1]
    gs_csv_name  = sys.argv[2]

    # Check the csv file
    if not sim_csv_name.endswith('.csv') or not gs_csv_name.endswith('.csv'):
        print("Error: The file must be a CSV file ending with .csv")
        printHelp()
        sys.exit(1)

    # Absolute csv file path
    repo_root = Path(__file__).resolve().parent.parent

    sim_csv_path = repo_root / "generated" / sim_csv_name
    gs_csv_path  = repo_root / "configs"  / gs_csv_name
    
    if not sim_csv_path.exists():
        sys.exit(f"Simulation file not found: {sim_csv_path}")
    if not gs_csv_path.exists():
        sys.exit(f"Ground‑station file not found: {gs_csv_path}")

    analyze(sim_csv_path, gs_csv_path)


