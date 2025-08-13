import sys
import utility as util
import distance_file_graph_generator as dfg
import cugraph, cudf
import gc
import pandas as pd
import cupy as cp
import dask_cuda
import dask.distributed
import time

""""
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
    cp.get_default_memory_pool().free_all_blocks()
    cp.get_default_pinned_memory_pool().free_all_blocks()
    start_time, end_time = period
    results = run_graphs_on_gpu(graph_generator, start_time, end_time, time_step, gpu_id)
    out_q.put((gpu_id, results))
    out_q.put(None)

def run_batch_graphs(graph_generator, offset, time_step, batch_size):
    batch_result = []
    streams = [cp.cuda.Stream() for _ in range(batch_size)]
    graphs = [graph_generator.get_graph(time_stamp) for time_stamp in range(offset, offset+batch_size*time_step, time_step)]

    for i, (g, stream) in enumerate(zip(graphs, streams)):
        with stream:
           batch_result.append(all_pairs_shortest_path_async(g, offset + i * time_step))

    for s in streams:
        s.synchronize()

    gpu_result = [next_hop(df.to_pandas()) for df in batch_result]
    cpu_results = pd.concat(gpu_result, ignore_index=True)
    gc.collect()
    cp.get_default_memory_pool().free_all_blocks()
    cp.get_default_pinned_memory_pool().free_all_blocks()

    cpu_results['NextHop'] = cpu_results['NextHop'].apply(lambda x: graph_generator.graph_builder.id_to_node[int(x)])
    cpu_results['Source'] = cpu_results['Source'].apply(lambda x: graph_generator.graph_builder.id_to_node[int(x)])
    cpu_results['Destination'] = cpu_results['Destination'].apply(lambda x: graph_generator.graph_builder.id_to_node[int(x)])
    cpu_results = cpu_results[cpu_results.apply(
        lambda row: graph_generator.is_ground_station(row['Destination']),
        axis=1
    )]
    return cpu_results

def distribute_graphs_on_gpus(total_time, time_step, graph_generator):
    set_start_method("spawn")
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

def all_pairs_shortest_path_async(G, time_stamp):
    all_next_hops = []
    nodes = G.nodes().to_pandas().values

    for src in nodes:
        sssp_df = cugraph.sssp(G, source=src)
        sssp_df['Source'] = src
        sssp_df['TimeStamp'] = time_stamp
        all_next_hops.append(sssp_df)

    return cudf.concat(all_next_hops, ignore_index=True)
"""

SOURCE_COMPUTATION_CHUNKS = 2

def next_hop(df, graph_generator):
    pred_map = dict(zip(zip(df['Source'], df['Destination']), df['predecessor']))

    def find_next_hop(v, source):
        curr = v
        prev = pred_map.get((source, curr), -1)
        if prev in (-1, source):
            return curr
        while prev != -1 and prev != source:
            curr = prev
            prev = pred_map.get((source, curr), -1)
        return curr

    df = df[df['Destination'].apply(graph_generator.graph_builder.is_id_ground_station)].copy()

    df['NextHop'] = [find_next_hop(v, s) for v, s in zip(df['Destination'], df['Source'])]

    return df[['TimeStamp', 'Source', 'Destination', 'NextHop']]

def get_total_gpu_memory():
    mem_size = 0
    n_gpu = cp.cuda.runtime.getDeviceCount()
    for i in range(n_gpu):
        with cp.cuda.Device(i):
            mem_info = cp.cuda.runtime.memGetInfo()
            free_mem = mem_info[0]
            mem_size += free_mem

    return mem_size

def calculate_batch_size(number_of_nodes):
    mem_size = get_total_gpu_memory()
    computable_space = mem_size // (8 * SOURCE_COMPUTATION_CHUNKS)
    batch_size = computable_space // (24 * number_of_nodes**2)
    return max(1, batch_size)

def shortest_path_async(g_df, srcs, time_stamp):
    G = cugraph.Graph()
    G.from_cudf_edgelist(g_df, source='src', destination='dst', edge_attr='weight')
    results = []

    for src in srcs:
        sssp_df = cugraph.sssp(G, source=src)
        sssp_df['Source'] = src
        sssp_df['TimeStamp'] = time_stamp
        sssp_df.rename(columns={'vertex': 'Destination'}, inplace=True)
        results.append(sssp_df)

    return cudf.concat(results, ignore_index=True)

def split_into_chunks(lst, n):
    avg_chunk_size = len(lst) // n
    remainder = len(lst) % n
    chunks = []
    start = 0

    for i in range(n):
        end = start + avg_chunk_size + (1 if i < remainder else 0)
        chunks.append(lst[start:end])
        start = end

    return chunks

def run_batch_graphs(client, graph_generator, offset, time_step, batch_size):
    future_graphs, future_src, future_timestamps = [], [], []
    for time_stamp in range(offset, offset+batch_size*time_step, time_step):
        g, sources = graph_generator.get_graph(time_stamp)
        sources_chunks = split_into_chunks(sources, SOURCE_COMPUTATION_CHUNKS)
        future_graphs += [g] * SOURCE_COMPUTATION_CHUNKS
        future_src += sources_chunks
        future_timestamps += [time_stamp] * SOURCE_COMPUTATION_CHUNKS

    futures = client.map(shortest_path_async, future_graphs, future_src, future_timestamps)
    batch_result = client.gather(futures)

    gpu_result = []
    for df in batch_result:
        cpu_df = df.to_pandas()
        gpu_result.append(next_hop(cpu_df, graph_generator))
        del df
    
    cpu_results = pd.concat(gpu_result, ignore_index=True)
    gc.collect()
    cp.get_default_memory_pool().free_all_blocks()
    cp.get_default_pinned_memory_pool().free_all_blocks()

    cpu_results['NextHop'] = cpu_results['NextHop'].apply(lambda x: graph_generator.graph_builder.id_to_node[int(x)])
    cpu_results['Source'] = cpu_results['Source'].apply(lambda x: graph_generator.graph_builder.id_to_node[int(x)])
    cpu_results['Destination'] = cpu_results['Destination'].apply(lambda x: graph_generator.graph_builder.id_to_node[int(x)])
    
    return cpu_results

def distribute_graphs_on_gpus(total_time, time_step, graph_generator):
    cluster = dask_cuda.LocalCUDACluster(rmm_pool_size="6GB", enable_cudf_spill=True)
    client = dask.distributed.Client(cluster)
    print(client)

    max_batch_size = calculate_batch_size(graph_generator.number_of_nodes)
    results = []
    for offset in range(0, total_time + time_step, max_batch_size * time_step):
        start_time = time.time()
        batch_size = min(max_batch_size, (total_time - offset) // time_step + 1)
        results.append(run_batch_graphs(client, graph_generator, offset, time_step, batch_size))
        print(f"Calculated forwarding table for batch timestamps {offset} to {offset + (batch_size - 1) * time_step} in {time.time() - start_time} seconds")

    client.close()
    cluster.close()
    return pd.concat(results, ignore_index=True)
    
def calculate_shortest_paths(node_writers, node_files, total_time, time_step, graph_generator):
    result_df = distribute_graphs_on_gpus(total_time, time_step, graph_generator)
    for source, group_df in result_df.groupby('Source'):
        for row in group_df.itertuples(index=False):
            node_writers[source].writerow((row.TimeStamp, row.Destination, row.NextHop))

    util.close_files(node_files)
    
def dijkstra_shortest_path_algorithm(distance_file_name):
    distance_csv_dataframe, constellation_name, time_step, total_time, simulation_details, nodes, _, _ = util.read_distance_file(distance_file_name)
    node_files, node_writers = util.forwarding_folder_csv_file(simulation_details, "DijkstraForwardingTable", nodes)
    graph_generator = dfg.GraphGenerator(distance_csv_dataframe, constellation_name, dfg.CUGraphBuilder(nodes, constellation_name), len(nodes))
    calculate_shortest_paths(node_writers, node_files, total_time, time_step, graph_generator)

def dijkstra_grid_plus_shortest_path_algorithm(distance_file_name):
    distance_csv_dataframe, constellation_name, time_step, total_time, simulation_details, nodes, number_of_orbits, number_of_satellites_per_orbit = util.read_distance_file(distance_file_name)
    node_files, node_writers = util.forwarding_folder_csv_file(simulation_details, "DijkstraGridPlusForwardingTable", nodes)
    grid_plus_graph_generator = dfg.GridPlusGraphGenerator(distance_csv_dataframe, constellation_name, dfg.CUGraphBuilder(nodes, constellation_name), len(nodes), number_of_orbits, number_of_satellites_per_orbit)
    calculate_shortest_paths(node_writers, node_files, total_time, time_step, grid_plus_graph_generator)

def dijkstra_static_topology_shortest_path_algorithm(distance_file_name, topology_file_name):
    distance_csv_dataframe, constellation_name, time_step, total_time, simulation_details, nodes, _, _ = util.read_distance_file(distance_file_name)
    node_files, node_writers = util.forwarding_folder_csv_file(simulation_details, "DijkstraStaticForwardingTable", nodes)
    static_topology_graph_generator = dfg.StaticTopologyGraphGenerator(distance_csv_dataframe, constellation_name, dfg.CUGraphBuilder(nodes, constellation_name), len(nodes), topology_file_name)
    calculate_shortest_paths(node_writers, node_files, total_time, time_step, static_topology_graph_generator)

def dijkstra_dynamic_topology_shortest_path_algorithm(distance_file_name, topology_file_name):
    distance_csv_dataframe, constellation_name, time_step, total_time, simulation_details, nodes, _, _ = util.read_distance_file(distance_file_name)
    node_files, node_writers = util.forwarding_folder_csv_file(simulation_details, "DijkstraDynamicForwardingTable", nodes)
    dynamic_topology_graph_generator = dfg.DynamicTopologyGraphGenerator(distance_csv_dataframe, constellation_name, dfg.CUGraphBuilder(nodes, constellation_name), len(nodes), topology_file_name)
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