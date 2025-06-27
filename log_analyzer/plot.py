import pandas as pd
import numpy as np
import matplotlib.pyplot as plt

def empirical_cdf(arr):
        # Sort values and compute CDF
        x = np.sort(arr)
        y = np.arange(1, len(x)+1) / len(x)
        return x, y

def hop_count_stretch_factor_cdf_plot(overall: pd.DataFrame) -> None:
    stretch_values = overall.loc[
        overall["FromDevice"] != "ALL", "Stretch_factor"
    ].dropna().values
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

def generate_plots(overall: pd.DataFrame, results_dir: str) -> None:
    """
    Generate plots for the analysis results.
    """
    # Plot Hop Count and Stretch Factor CDF
    hop_count_stretch_factor_cdf_plot(overall)

    # Save the plot 
    plt.savefig(f"{results_dir}/hop_stretch_cdf_plot.png")
    print(f'Stretch Factor and Hop Count CDF plot saved as {results_dir}/hop_stretch_cdf_plot.png')
    