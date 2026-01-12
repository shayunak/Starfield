import sys
import pandas as pd
import numpy as np
import matplotlib.pyplot as plt

def unit_phi(point):
    x, y, _ = point
    r_xy = np.sqrt(x**2 + y**2)
    return np.array([-y / r_xy, x / r_xy, 0.0])

def unit_theta(point):
    x, y, z = point
    r_xy = np.sqrt(x**2 + y**2)
    return np.array([- (x * z) / r_xy, - (y * z) / r_xy, r_xy])

def project_onto_geodesic_curve(point, source, dest):
    normal_plane = np.cross(source, dest)
    normal_unit_plane = normal_plane / np.linalg.norm(normal_plane)
    projected_point = point - np.dot(point, normal_unit_plane) * normal_unit_plane
    return projected_point / np.linalg.norm(projected_point)

def tangent_vector(point, end_point):
    perp_plane = np.cross(end_point, point)
    tangent = np.cross(perp_plane, point)
    return tangent / np.linalg.norm(tangent)

def convert_to_cartesian(lat, lon):
    lat = np.deg2rad(lat)
    lon = np.deg2rad(lon)

    x = np.cos(lat) * np.cos(lon)
    y = np.cos(lat) * np.sin(lon)
    z = np.sin(lat)

    return np.array([x, y, z])

def adjust_direction(phi_comp, theta_comp):
    if phi_comp >= 0 and theta_comp >= 0:
        return phi_comp, theta_comp
    elif phi_comp < 0 and theta_comp >= 0:
        return -phi_comp, -theta_comp
    elif phi_comp < 0 and theta_comp < 0:
        return -phi_comp, -theta_comp
    else:
        return phi_comp, theta_comp

def get_region(p_cart, lat_step, lon_step):
    x, y, z = p_cart

    lat = np.arcsin(z)
    lon = np.arctan2(y, x)
    lon = np.mod(lon, 2.0 * np.pi)

    return np.floor((lat + np.pi / 2.0) / lat_step), np.floor(lon / lon_step)

def get_flows_traffics(demand_file):
    demand_df = pd.read_csv(f"./input/{demand_file}")
    demand_df = demand_df.loc[demand_df['Timestamp(ms)'] == 0].drop(columns=['Timestamp(ms)'])
    demand_df['Length(Mb)'] = demand_df['Length(Mb)'] / demand_df['Length(Mb)'].max()
    return list(zip(demand_df['Source'], demand_df['Destination'], demand_df['Length(Mb)']))

def get_ground_stations(ground_station_file):
    gs_df = pd.read_csv(f"./configs/{ground_station_file}")
    return {id: convert_to_cartesian(lat, lon) for id, lat, lon in zip(gs_df['Id'], gs_df['Latitude'], gs_df['Longitude'])}

def generate_regions(lat_step, lon_step, lat_bins, lon_bins):
    regions = []
    for i in range(lat_bins):
        for j in range(lon_bins):
            region_name = f"L-{i}-{j}"
            rep_point_lat = -np.pi / 2.0 + (i + 0.5) * lat_step
            rep_point_lon = (j + 0.5) * lon_step
            rep_cart = convert_to_cartesian(rep_point_lat, rep_point_lon)
            regions.append((region_name, i, j, rep_cart))

    return regions

def get_region_flows(regions, flows, gs, lat_step, lon_step):
    region_flow = {}
    for region in regions:
        for flow in flows:
           region_name, i, j, rep_cart = region
           flow_source, flow_dest, flow_size = flow
           source_cart = gs[flow_source]
           dest_cart = gs[flow_dest]
           proj_point = project_onto_geodesic_curve(rep_cart, source_cart, dest_cart)
           proj_point_region = get_region(proj_point, lat_step, lon_step)
           if proj_point_region == (i, j):
               if region_name not in region_flow:
                   region_flow[region_name] = []
               region_flow[region_name].append((proj_point, source_cart, flow_size))


    return region_flow

def demand_analysis_csv(region_agg_comp, demand_file, latitude_lines, longitude_lines):
    df = pd.DataFrame(region_agg_comp, columns=['Region', 'Phi', 'Theta'])
    df.to_csv(f"./generated/GeometryAnalysis_{demand_file}#{latitude_lines}-{longitude_lines}Mesh.csv", index=False)
    return df

def plot_region_agggreate_components(phi, theta, demand_file, latitude_lines, longitude_lines):
    plt.figure(dpi=600)

    # Plot arrows from origin with direction (x, y) and magnitude/color = z
    plt.figure(figsize=(6,6))
    plt.quiver(
        np.zeros_like(phi),  # arrow origins x
        np.zeros_like(theta),  # arrow origins y
        phi, theta,              # arrow directions
        angles='xy',
        scale_units='xy',
        scale=1,
        cmap='viridis',
        width=0.015
    )
    plt.xlim(0, 60)
    plt.ylim(-60, 60)
    plt.xlabel('Phi')
    plt.ylabel('Theta')
    plt.title('Directional vectors with magnitude as color')
    plt.grid(True)
    plt.show()
    plt.savefig(f"./traffic_pattern_generator/GeometryAnalysis_{demand_file}#{latitude_lines}-{longitude_lines}Mesh.png")

def weighted_mean_resultant_length(phi, theta):
    mag = np.sqrt(phi**2 + theta**2)

    angles = np.arctan2(theta, phi)

    C = np.sum(mag * np.cos(angles))
    S = np.sum(mag * np.sin(angles))
    return np.sqrt(C**2 + S**2) / np.sum(mag)

def analyze_geometry(demand_file, ground_station_file, latitude_lines, longitude_lines):
    flows = get_flows_traffics(demand_file)
    gs = get_ground_stations(ground_station_file)
    lat_step = np.pi / int(latitude_lines)
    lon_step = 2.0 * np.pi / int(longitude_lines)
    regions = generate_regions(lat_step, lon_step, int(latitude_lines), int(longitude_lines))
    region_flows = get_region_flows(regions, flows, gs, lat_step, lon_step)
    
    #Calculate aggregate directional components for each region
    region_agg_comp = []
    for region_name in region_flows:
        agg_phi, agg_theta = 0.0, 0.0
        for projection in region_flows[region_name]:
            proj_point, source_cart, flow_size = projection
            tangent_vec = tangent_vector(proj_point, source_cart)
            phi_vec = unit_phi(proj_point)
            theta_vec = unit_theta(proj_point)
            phi_comp = np.dot(tangent_vec, phi_vec)
            theta_comp = np.dot(tangent_vec, theta_vec)
            adj_phi, adj_theta = adjust_direction(phi_comp, theta_comp)
            agg_phi += flow_size * adj_phi
            agg_theta += flow_size * adj_theta
        
        region_agg_comp.append((region_name, agg_phi, agg_theta))
    
    df_comp = demand_analysis_csv(region_agg_comp, demand_file, latitude_lines, longitude_lines)
    plot_region_agggreate_components(df_comp['Phi'].to_numpy(), df_comp['Theta'].to_numpy(), demand_file, latitude_lines, longitude_lines)
    print(f"Weighted Mean Resultant Length: {weighted_mean_resultant_length(df_comp['Phi'].to_numpy(), df_comp['Theta'].to_numpy())}")

def printHelp():    
    print("demand_analyzer.py --help")
    print("demand_analyzer.py --geometry [demand_file] [ground_station_file] [latitude_lines] [longitude_lines]")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Please provide an option as a command line argument!")
        printHelp()
        exit(1)
    
    if sys.argv[1] == "--help":
        printHelp()
    elif sys.argv[1] == "--geometry" and len(sys.argv) == 6:
        analyze_geometry(sys.argv[2], sys.argv[3], sys.argv[4], sys.argv[5])
    else:
        print("Invalid Option or Missing Arguments!")
        printHelp()
        exit(1)