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

def calculate_shortest_paths(node_writers, total_time, time_step, graph_generator):
    result_df = distribute_graphs_on_gpus(total_time, time_step, graph_generator)
    for source, group_df in result_df.groupby('Source'):
        for row in group_df.itertuples(index=False):
            node_writers[source].writerow((row.TimeStamp, row.Destination, row.NextHop))

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Please provide the required options as a command line argument!")
        util.printHelp()
        exit(1)
    
    if sys.argv[1] == "--help":
        util.printHelp()
        exit(1)
    elif len(sys.argv) < 3:
        print("Please provide the required options as a command line argument!")
        util.printHelp()
        exit(1)
    elif len(sys.argv) == 3:
        print("Please provide the distance file name!")
        util.printHelp()
        exit(1)

    distance_file_name = sys.argv[3]
    (distance_csv_dataframe, constellation_name, time_step, total_time, simulation_details, 
        nodes, number_of_orbits, number_of_satellites_per_orbit) = util.read_distance_file(distance_file_name)
    
    folder_name = ""
    graph_builder = dfg.CUGraphBuilder(nodes, constellation_name)
    number_of_nodes = len(nodes)
    graph_generator = None
    link_filter_graph_generator = None

    if sys.argv[2] == "--dijkstra" and len(sys.argv) == 4:
        graph_generator = dfg.GraphGenerator()
        folder_name += "DijkstraForwardingTable"
    elif sys.argv[2] == "--dijkstra_grid_plus" and len(sys.argv) == 4:
        graph_generator = dfg.GridPlusGraphGenerator(number_of_orbits, number_of_satellites_per_orbit)
        folder_name += "DijkstraGridPlusForwardingTable"
    elif sys.argv[2] == "--dijkstra_static" and len(sys.argv) == 5:
        graph_generator = dfg.StaticTopologyGraphGenerator(sys.argv[4])
        folder_name += "DijkstraStaticForwardingTable"
    elif sys.argv[2] == "--dijkstra_dynamic" and len(sys.argv) == 5:
        graph_generator = dfg.DynamicTopologyGraphGenerator(sys.argv[4])
        folder_name += "DijkstraDynamicForwardingTable"
    else:
        print("Invalid Shortest Path Option or Missing Arguments!")
        util.printHelp()
        exit(1)

    if sys.argv[1] == "--isl":
        link_filter_graph_generator = dfg.OnlyISLLinkFilter(distance_csv_dataframe, constellation_name, graph_builder, graph_generator, number_of_nodes)
        folder_name += "(ISL_Only)"
    elif sys.argv[1] == "--gsl":
        link_filter_graph_generator = dfg.OnlyGSLLinkFilter(distance_csv_dataframe, constellation_name, graph_builder, graph_generator, number_of_nodes)
        folder_name += "(GSL_Only)"
    elif sys.argv[1] == "--isl&gsl":
        link_filter_graph_generator = dfg.ISLAndGSLLinkFilter(distance_csv_dataframe, constellation_name, graph_builder, graph_generator, number_of_nodes)
        folder_name += "(ISL_GSL)"
    else:
        print("Please provide the required options as a command line argument!")
        util.printHelp()
        exit(1)

    node_files, node_writers = util.forwarding_folder_csv_file(simulation_details, folder_name, nodes)
    calculate_shortest_paths(node_writers, total_time, time_step, link_filter_graph_generator)
    util.close_files(node_files)