import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import matplotlib.colors as mcolors
import random

def empirical_cdf(arr):
    # Sort values and compute CDF
    x = np.sort(arr)
    y = np.arange(1, len(x)+1) / len(x)
    return x, y 

def stretch_factor_effective_latency_cdf_plot(overall: pd.DataFrame,results_dir: str) -> None:
    stretch_x, stretch_cdf = empirical_cdf(overall.loc[overall["FromDevice"] != "ALL", "Stretch_Factor"].dropna().values)
    effective_x, effective_cdf = empirical_cdf(overall.loc[overall["FromDevice"] != "ALL", "Effective_Latency_Factor"].dropna().values)

    # Draw the CDF plot
    plt.figure()
    plt.plot(stretch_x, stretch_cdf, linestyle='-',  label="Stretch Factor CDF", color='blue')
    plt.plot(effective_x, effective_cdf, linestyle='--', label="Effective Latency Factor CDF", color='red')
    plt.xlabel("Factor(Number)")
    plt.ylabel("CDF")
    plt.title("CDF of Stretch and Effective Latency Factor", fontsize=12)
    plt.legend()
    plt.xlim(left=0, right=3)
    plt.grid(True)

    plt.savefig(f"{results_dir}/stretch_effective_cdf_plot.png")
    print(f'Stretch Factor and Effective Latency Factor CDF plot saved as {results_dir}/stretch_effective_cdf_plot.png')

def hop_count_cdf_plot(overall: pd.DataFrame,results_dir: str) -> None:
    hop_x, hop_cdf = empirical_cdf(overall.loc[overall["FromDevice"] != "ALL", "Hop_Count"].dropna().values)

    # Draw the CDF plot
    plt.figure()
    plt.plot(hop_x, hop_cdf, linestyle='-', label="Hop Count CDF", color='blue')
    plt.xlabel("Count")
    plt.ylabel("CDF")
    plt.title("CDF of Hop Count", fontsize=12)
    plt.legend()
    plt.grid(True)
    plt.xlim(left=0, right=20)

    plt.savefig(f"{results_dir}/hop_count_cdf_plot.png")
    print(f'Hop Count CDF plot saved as {results_dir}/hop_count_cdf_plot.png')

# Plotting city pair RTT
def RTT_cdf_plot(overall: pd.DataFrame,results_dir: str) -> None:
    rtt_series = pd.to_numeric(overall["RTT_ms"], errors="coerce").dropna()
    draw_rtt = np.sort(rtt_series.values)
    cdf = np.arange(1, len(draw_rtt) + 1) / len(draw_rtt)
    
    plt.figure()
    plt.plot(draw_rtt, cdf, drawstyle='steps-post', color='orange', linewidth=3, label=r'City-RTT CDF')
    plt.xlabel('City-city RTT (ms)', fontsize=12)
    plt.ylabel('CDF across city-pairs', fontsize=12)
    plt.title('CDF of City-RTTs', fontsize=14)
    plt.legend()
    plt.grid(True)

    plt.savefig(f"{results_dir}/RTT_cdf_plot.png")
    print(f'RTT and CDF plot saved as {results_dir}/RTT_cdf_plot.png')

def satellite_usage_plot(results_dir: str) -> None:
    satellite_df = pd.read_csv(results_dir/"link_usage.csv")
    plt.figure()
    plt.plot(satellite_df["UsageCount"].values,drawstyle="steps-post", color='orange',label="Link usage")
    plt.xlabel("Links ordered by usage")
    plt.ylabel("No. of packets using a link")
    plt.title('Inter-Satellite Link (ISL) Usage Frequency', fontsize=14)
    plt.grid(True)
    plt.legend()

    plt.savefig(f"{results_dir}/Link_usage_plot.png")
    print(f'Link usage plot saved as {results_dir}/Link_usage_plot.png')

def generate_plots(overall: pd.DataFrame, results_dir: str) -> None:
    """
    Generate plots for the analysis results.
    """
    # Plot Hop Count and Stretch Factor CDF
    stretch_factor_effective_latency_cdf_plot(overall, results_dir)
    hop_count_cdf_plot(overall, results_dir)
    RTT_cdf_plot(overall, results_dir)
    satellite_usage_plot(results_dir)

def combine_stretch_factor_effective_latency(overalls, results_dir, combination_name: str) -> None:
    plt.figure(dpi=300)
    for overall_name, overall_df, _, _color in overalls:
        stretch_x, stretch_cdf = empirical_cdf(overall_df.loc[overall_df["FromDevice"] != "ALL", "Stretch_Factor"].dropna().values)
        effective_x, effective_cdf = empirical_cdf(overall_df.loc[overall_df["FromDevice"] != "ALL", "Effective_Latency_Factor"].dropna().values)
        plt.plot(stretch_x, stretch_cdf, linestyle='-',  label=f"Stretch Factor({overall_name})", color=_color, lw=0.8)
        plt.plot(effective_x, effective_cdf, linestyle=':', label=f"Effective Latency Factor({overall_name})", color=_color, lw=0.8)

    plt.xlabel("Factor(Number)")
    plt.ylabel("CDF")
    plt.title(f"CDF of Stretch Factor and Effective Latency Factor for {combination_name}", fontsize=8)
    plt.xlim(left=0, right=3)
    plt.legend(fontsize=6)
    plt.grid(True)

    plt.savefig(f"{results_dir}/combined_stretch_effective_cdf_plot.png")
    print(f'Combined Stretch and Effective Latency Factor CDF plot saved as {results_dir}/combined_stretch_effective_cdf_plot.png')

def combine_hop_count(overalls, results_dir, combination_name: str) -> None:
    plt.figure(dpi=300)
    for overall_name, overall_df, _, _color in overalls:
        hop_x, hop_cdf = empirical_cdf(overall_df.loc[overall_df["FromDevice"] != "ALL", "Hop_Count"].dropna().values)
        plt.plot(hop_x, hop_cdf, linestyle='--', label=f"Hop Count({overall_name})", color=_color, lw=0.8)

    plt.xlabel("Count")
    plt.ylabel("CDF")
    plt.title(f"CDF of Hop Count for {combination_name}", fontsize=10)
    plt.xlim(left=0, right=20)
    plt.legend(fontsize=6)
    plt.grid(True)

    plt.savefig(f"{results_dir}/combined_hop_cdf_plot.png")
    print(f'Combined Hop Count CDF plot saved as {results_dir}/combined_hop_cdf_plot.png')

def combine_RTT(overalls, results_dir, combination_name: str) -> None:
    plt.figure(dpi=300)
    for overall_name, overall_df, _, _color in overalls:
        rtt_series = pd.to_numeric(overall_df["RTT_ms"], errors="coerce").dropna()
        draw_rtt = np.sort(rtt_series.values)
        cdf = np.arange(1, len(draw_rtt) + 1) / len(draw_rtt)
        plt.plot(draw_rtt, cdf, drawstyle='steps-post', color=_color, label=f'City-RTT CDF({overall_name})', lw=0.8)

    plt.xlabel('City-city RTT (ms)', fontsize=12)
    plt.ylabel('CDF across city-pairs', fontsize=12)
    plt.title(f'CDF of City-RTTs for {combination_name}', fontsize=10)
    plt.legend()
    plt.grid(True)

    plt.savefig(f"{results_dir}/combined_RTT_cdf_plot.png")
    print(f'Combined RTT CDF plot saved as {results_dir}/combined_RTT_cdf_plot.png')

def combine_satellite_usage(overalls, results_dir, combination_name: str) -> None:
    plt.figure(dpi=300)
    for overall_name, _, link_usage_df, _color in overalls:
        plt.plot(link_usage_df["UsageCount"].values,drawstyle="steps-post", color=_color,label=f"Link usage({overall_name})", lw=0.8)

    plt.xlabel("Links ordered by usage")
    plt.ylabel("No. of packets using a link")
    plt.title(f'Inter-Satellite Link (ISL) Usage Frequency for {combination_name}', fontsize=10)
    plt.xlim(left=0, right=2)
    plt.grid(True)
    plt.legend()

    plt.savefig(f"{results_dir}/combined_link_usage_plot.png")
    print(f'Combined Link usage plot saved as {results_dir}/combined_link_usage_plot.png')

def combine_overalls(overalls: list, combination_folder: str, combination_name: str) -> None:
    colors = list(mcolors.TABLEAU_COLORS.values())
    random.shuffle(colors)
    overalls_with_colors = [(overall_name, overall_df, link_usage_df, colors[i % len(colors)]) for i, (overall_name, overall_df, link_usage_df) in enumerate(overalls)]
    combine_stretch_factor_effective_latency(overalls_with_colors, combination_folder, combination_name)
    combine_hop_count(overalls_with_colors, combination_folder, combination_name)
    combine_RTT(overalls_with_colors, combination_folder, combination_name)
    combine_satellite_usage(overalls_with_colors, combination_folder, combination_name)