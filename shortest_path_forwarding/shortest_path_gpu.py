import sys
import utility as util
import distance_file_graph_generator as dfg
import cugraph
import cudf
import pandas as pd
import cupy as cp
from multiprocessing import Process, Queue, set_start_method

def gpu_next_hop(df):
    df['next_hop'] = df['vertex']

    while True:
        join_df = df[['vertex', 'predecessor']].rename(columns={'vertex': 'next_hop', 'predecessor': 'next_pred'})
        df = df.merge(join_df, on='next_hop', how='left')
        
        new_next_hop = df['next_pred'].where((df['next_pred'] != df['source']) & (df['next_pred'] != -1), df['next_hop'])

        if (new_next_hop != df['next_hop']).any():
            df['next_hop'] = new_next_hop
            df = df.drop(columns=['next_pred'])
        else:
            df = df.drop(columns=['next_pred'])
            break

    df = df.rename(columns={
        'vertex': 'Destination',
        'next_hop': 'NextHop'
    })

    return df[['Source', 'Destination', 'NextHop']]

def all_pairs_shortest_path_async(G, time_stamp):
    all_next_hops = []
    nodes = G.nodes().to_pandas().values

    for src in nodes:
        sssp_df = cugraph.sssp(G, source=src)
        sssp_df['Source'] = src
        all_next_hops.append(sssp_df)
            
    sssp_all = cudf.concat(all_next_hops, ignore_index=True)
    next_hop_df = gpu_next_hop(sssp_all)
    next_hop_df.insert(0, 'TimeStamp', time_stamp)

    return next_hop_df

def run_batch_graphs(graph_generator, offset, time_step, batch_size):
    batch_result = []
    streams = [cp.cuda.Stream() for _ in range(batch_size)]
    graphs = [graph_generator.get_graph(time_stamp) for time_stamp in range(offset, offset+batch_size*time_step, time_step)]

    for i, (g, stream) in enumerate(zip(graphs, streams)):
        with stream:
            batch_result.append(all_pairs_shortest_path_async(g, offset + i * time_step))

    for s in streams:
        s.synchronize()

    gpu_results = cudf.concat(batch_result, ignore_index=True)
    cpu_results = gpu_results.to_pandas()
    del gpu_results

    return cpu_results[cpu_results.apply(
        lambda row: graph_generator.is_ground_station(row['Destination']),
        axis=1
    )]

def calculate_batch_size(number_of_nodes, gpu_id):
    with cp.cuda.Device(gpu_id):
        mem_info = cp.cuda.runtime.memGetInfo()
        free_mem = mem_info[0]
        computable_space = free_mem // 2
        batch_size = computable_space // (24 * number_of_nodes**2) 
        return max(1, batch_size)

def run_graphs_on_gpu(graph_generator, start_time, end_time, time_step, gpu_id):
    max_batch_size = calculate_batch_size(graph_generator.number_of_nodes, gpu_id)
    results = []
    for offset in range(start_time, end_time, max_batch_size * time_step):
        batch_size = min(max_batch_size, (end_time - offset) // time_step)
        results.append(run_batch_graphs(graph_generator, offset, time_step, batch_size))
        print(f"GPU {gpu_id}: Calculated forwarding table for batch timestamps {offset} to {offset + (batch_size - 1) * time_step}...")
    return pd.concat(results, ignore_index=True)

def partition_on_gpus(total_time, time_step, weights):
    graphs_on_gpus = []
    total_graphs = (total_time // time_step) + 1
    for w in weights:
        size = round(total_graphs * (w))
        graphs_on_gpus.append(size)

    diff = total_graphs - sum(graphs_on_gpus)
    if diff != 0:
        sorted_indices = sorted(range(len(weights)), key=lambda x: weights[x], reverse=True)
        for i in range(abs(diff)):
            target_idx = sorted_indices[i % len(weights)]
            if diff > 0:
                graphs_on_gpus[target_idx] += 1
            elif diff < 0 and graphs_on_gpus[target_idx]:
                graphs_on_gpus[target_idx] -= 1

    graph_times = []
    offset = 0
    for size in graphs_on_gpus:
        graph_times.append((offset, offset + size * time_step))
        offset += size * time_step

    return graph_times

def get_normalized_gpu_mem_size(n_gpu):
    mem_size = []
    for i in range(n_gpu):
        with cp.cuda.Device(i):
            mem_info = cp.cuda.runtime.memGetInfo()
            free_mem = mem_info[0]
            mem_size.append(free_mem // (1024 * 1024))

    total_mem = sum(mem_size)
    return [x / total_mem for x in mem_size]

def gpu_worker(gpu_id, period, time_step, graph_generator, out_q):
    cp.cuda.Device(gpu_id).use()
    start_time, end_time = period
    results = run_graphs_on_gpu(graph_generator, start_time, end_time, time_step, gpu_id)
    out_q.put((gpu_id, results))
    out_q.put(None)

def distribute_graphs_on_gpus(total_time, time_step, graph_generator):
    set_start_method("fork")
    n_gpu = cp.cuda.runtime.getDeviceCount()
    gpu_weights = get_normalized_gpu_mem_size(n_gpu)
    graphs_on_gpus = partition_on_gpus(total_time, time_step, gpu_weights)
    
    q = Queue()
    procs = []
    for gpu_id in range(n_gpu):
        p = Process(target=gpu_worker, args=(gpu_id, graphs_on_gpus[gpu_id], time_step, graph_generator, q))
        p.start()
        procs.append(p)

    finished = 0
    results = []
    while finished < n_gpu:
        item = q.get()
        if item is None:
            finished += 1
        else:
            results.append(item)

    for p in procs:
        p.join()

    sorted_results = sorted(results, key=lambda t: t[0])
    final_results = [r[1] for r in sorted_results]
    return pd.concat(final_results, ignore_index=True)

def calculate_shortest_paths(node_writers, node_files, total_time, time_step, graph_generator):
    result_df = distribute_graphs_on_gpus(total_time, time_step, graph_generator)
    for source, group_df in result_df.groupby('Source'):
        for row in group_df.itertuples(index=False):
            node_writers[source].writerow(row['TimeStamp'], row['Destination'], row['NextHop'])

    util.close_files(node_files)
    

def dijkstra_shortest_path_algorithm(distance_file_name):
    distance_csv_dataframe, constellation_name, time_step, total_time, simulation_details, nodes, _, _ = util.read_distance_file(distance_file_name)
    node_files, node_writers = util.forwarding_folder_csv_file(simulation_details, "DijkstraForwardingTable", nodes)
    graph_generator = dfg.GraphGenerator(distance_csv_dataframe, constellation_name, dfg.CUGraphBuilder(), len(nodes))
    calculate_shortest_paths(node_writers, node_files, total_time, time_step, graph_generator)

def dijkstra_grid_plus_shortest_path_algorithm(distance_file_name):
    distance_csv_dataframe, constellation_name, time_step, total_time, simulation_details, nodes, number_of_orbits, number_of_satellites_per_orbit = util.read_distance_file(distance_file_name)
    node_files, node_writers = util.forwarding_folder_csv_file(simulation_details, "DijkstraGridPlusForwardingTable", nodes)
    grid_plus_graph_generator = dfg.GridPlusGraphGenerator(distance_csv_dataframe, constellation_name, dfg.CUGraphBuilder(), len(nodes), number_of_orbits, number_of_satellites_per_orbit)
    calculate_shortest_paths(node_writers, node_files, total_time, time_step, grid_plus_graph_generator)

def dijkstra_static_topology_shortest_path_algorithm(distance_file_name, topology_file_name):
    distance_csv_dataframe, constellation_name, time_step, total_time, simulation_details, nodes, _, _ = util.read_distance_file(distance_file_name)
    node_files, node_writers = util.forwarding_folder_csv_file(simulation_details, "DijkstraStaticForwardingTable", nodes)
    static_topology_graph_generator = dfg.StaticTopologyGraphGenerator(distance_csv_dataframe, constellation_name, dfg.CUGraphBuilder(), len(nodes),topology_file_name)
    calculate_shortest_paths(node_writers, node_files, total_time, time_step, static_topology_graph_generator)


def dijkstra_dynamic_topology_shortest_path_algorithm(distance_file_name, topology_file_name):
    distance_csv_dataframe, constellation_name, time_step, total_time, simulation_details, nodes, _, _ = util.read_distance_file(distance_file_name)
    node_files, node_writers = util.forwarding_folder_csv_file(simulation_details, "DijkstraDynamicForwardingTable", nodes)
    dynamic_topology_graph_generator = dfg.DynamicTopologyGraphGenerator(distance_csv_dataframe, constellation_name, dfg.CUGraphBuilder(), len(nodes),topology_file_name)
    calculate_shortest_paths(node_writers, node_files, total_time, time_step, dynamic_topology_graph_generator)

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Please provide an option as a command line argument!")
        util.printHelp()
        exit(1)
    
    if sys.argv[1] == "--help":
        util.printHelp()
    elif sys.argv[1] == "--dijkstra" and len(sys.argv) == 3:
        dijkstra_shortest_path_algorithm(sys.argv[2])
    elif sys.argv[1] == "--dijkstra_grid_plus" and len(sys.argv) == 3:
        dijkstra_grid_plus_shortest_path_algorithm(sys.argv[2])
    elif sys.argv[1] == "--dijkstra_static_topology" and len(sys.argv) == 4:
        dijkstra_static_topology_shortest_path_algorithm(sys.argv[2], sys.argv[3])
    elif sys.argv[1] == "--dijkstra_dynamic_topology" and len(sys.argv) == 4:
        dijkstra_dynamic_topology_shortest_path_algorithm(sys.argv[2], sys.argv[3])
    else:
        print("Invalid Option or Missing Arguments!")
        util.printHelp()
        exit(1)