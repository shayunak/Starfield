from pathlib import Path
import sys, os, csv
import pandas as pd

# Command line help
def printHelp():
    print("log_analyzer/analyze.py --help")
    print("log_analyzer/analyze.py [Simulation Log]")


# Calculate lantnecy
def packet_latency(group: pd.DataFrame) -> pd.Series:
    """Return latency (ms) for one PacketId group."""
    send_times = group.loc[group["Event"] == "SEND", "TimeStamp(ms)"]
    deliver_times = group.loc[group["Event"] == "DELIVERED", "TimeStamp(ms)"]
    if send_times.empty or deliver_times.empty:
        return pd.NA
    return deliver_times.min() - send_times.min()

# Analyze
def analyze(csv_file: Path) -> None:
    df = pd.read_csv(csv_file)
    
    #Average latency
    final_df = (
        df.groupby("PacketId")
          .apply(packet_latency)
          .dropna()
          .reset_index(name="Latency_ms")
          .sort_values("PacketId")
    )
    avg_latency = final_df["Latency_ms"].mean()
    print(f"Average latency: {avg_latency:.5f} ms")
    print("================================================================")

    # Source & destination latency (per link)
    send = (df[df["Event"] == "SEND"]
                .sort_values("TimeStamp(ms)")
                .drop_duplicates("PacketId"))
    deliver = (df[df["Event"] == "DELIVERED"]
                   .sort_values("TimeStamp(ms)")
                   .drop_duplicates("PacketId"))

    final_df["FromDevice"] = final_df["PacketId"].map(
        send.set_index("PacketId")["FromDevice"])
    final_df["ToDevice"] = final_df["PacketId"].map(
        deliver.set_index("PacketId")["ToDevice"])

    link_avg_latency = (final_df
                        .groupby(["FromDevice", "ToDevice"])["Latency_ms"]
                        .mean()
                        .reset_index(name="Avg_Latency_ms")
                        .sort_values(["FromDevice", "ToDevice"]))
    link_avg_latency.index = range(1, len(link_avg_latency) + 1)

    print("Average latency for each link:")
    print(link_avg_latency.to_string(index=False))
    print("================================================================")

    # Hop count
    hop_counts = (df.query('Event == "RECEIVE"')
                    .groupby("PacketId").size()
                    .rename("Hop_count"))

    delivered_packet_ids = final_df["PacketId"]
    hop_counts = hop_counts.reindex(delivered_packet_ids, fill_value=0)

    hops_df = hop_counts.reset_index().sort_values("PacketId")
    print(f"Average Hop counts = {hops_df['Hop_count'].mean():.2f}, "
          f"across all successful delivered {len(hops_df)} packets")
    print("================================================================")

    final_df["Hop_count"] = final_df["PacketId"].map(hop_counts)
    each_hop = (final_df.groupby(["FromDevice", "ToDevice"])["Hop_count"]
                      .mean()
                      .reset_index(name="Avg_Hop")
                      .sort_values(["FromDevice", "ToDevice"]))

    print("Average hop count for each link:")
    print(each_hop.to_string(index=False))
    print("================================================================")

    # Throughput & drops
    delivered_set = set(df.loc[df["Event"] == "DELIVERED", "PacketId"])
    duration_s = (df["TimeStamp(ms)"].max() - df["TimeStamp(ms)"].min()) / 1000
    throughput = len(delivered_set) / duration_s

    dropped_packets = df.loc[df["Event"] == "DROP", "PacketId"].sort_values()
    print(f"Delivered packets from source to destination: {len(delivered_set)}")
    print(f"Dropped packet count: {len(dropped_packets)}")
    print(f"Avg throughput: {throughput:.2f} pkt/s")

# CLI entry point
if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Please provide the CSV filename as an argument!")
        printHelp()
        sys.exit(1)

    csv_filename = sys.argv[1]

    # Check the csv file
    if not csv_filename.endswith(".csv"):
        print("Error: The file must be a CSV file ending with .csv")
        printHelp()
        sys.exit(1)

    # Absolute csv file path
    repo_root = Path(__file__).resolve().parent.parent
    csv_path = repo_root / "generated" / csv_filename
    if not csv_path.exists():
        sys.exit(f"Could not find：{csv_path}")
    analyze(csv_path)


