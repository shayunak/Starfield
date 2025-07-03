import pandas as pd
import numpy as np
import matplotlib.pyplot as plt

def hop_count_stretch_factor_cdf_plot(overall: pd.DataFrame,results_dir: str) -> None:
    def empirical_cdf(arr):
        # Sort values and compute CDF
        x = np.sort(arr)
        y = np.arange(1, len(x)+1) / len(x)
        return x, y
    stretch_values = overall.loc[overall["FromDevice"] != "ALL", "Stretch_factor"].dropna().values
    hop_values = overall["Avg_Hop"].dropna().values
    sort_x, sort_cdf = empirical_cdf(stretch_values)
    hop_x, hop_cdf = empirical_cdf(hop_values)

    # Draw the CDF plot
    plt.figure()
    plt.plot(sort_x, sort_cdf, linestyle='-',  label="Stretch Factor CDF")
    plt.plot(hop_x, hop_cdf, linestyle='--', label="Hop Count CDF")
    plt.xlabel("Stretch Factor or Hop Count")
    plt.ylabel("CDF Across Links")
    plt.title("CDF of Stretch Factor & Hop Count")
    plt.legend()
    plt.grid(True)

    plt.savefig(f"{results_dir}/hop_stretch_cdf_plot.png")
    print(f'Stretch Factor and Hop Count CDF plot saved as {results_dir}/hop_stretch_cdf_plot.png')

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
    satellite_df = pd.read_csv(results_dir/"starlink_usage.csv")
    plt.figure()
    plt.plot(satellite_df["UsageCount"].values,drawstyle="steps-post", color='orange',label="Starlink link usage")
    plt.xlabel("Links ordered by usage")
    plt.ylabel("No. of paths using a link")
    plt.grid(True)
    plt.legend()

    plt.savefig(f"{results_dir}/Link_usage_plot.png")
    print(f'Link usage plot saved as {results_dir}/Link_usage_plot.png')



def generate_plots(overall: pd.DataFrame, results_dir: str) -> None:
    """
    Generate plots for the analysis results.
    """
    # Plot Hop Count and Stretch Factor CDF
    hop_count_stretch_factor_cdf_plot(overall,results_dir)
    RTT_cdf_plot(overall,results_dir)
    satellite_usage_plot(results_dir)