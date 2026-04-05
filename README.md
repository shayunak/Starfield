# Starfield
The following project is a packet-level distributed simulator for Low Earth Orbit (LEO) satellite networks. The network consists of satellites, and ground stations, that generate traffic. The main purpose of the simulator compared to state-of-the-art similar simulators, such as, Hypatia ([Kassing, 2022](https://dl.acm.org/doi/abs/10.1145/3419394.3423635)), is to build an easy-to-use, fast, light, configurable, and link-aware system that supports a shell of specified constellation. The simulator achieves the asserted proporties through:

1. **Fast, and Light:** The simulator integrates inter-satellite, and satellite-ground-station astronomical distance calculation with packet queuing and forwarding in a goroutine (Golang lightweight process) to manage thousands of satellite, and ground stations with minimal overhead due to memory efficiency, and small stack size. Moreover, by solving trigonomical astronomical equations, a linear time complexity can be achieved for inter-satellite distances compared to quadratic pair-wise approach.
2. **Configurable:** Unlike StarPerf ([Lai, 2020](https://ieeexplore.ieee.org/document/9259357)), and StarryNet([Lai, 2023](https://www.usenix.org/conference/nsdi23/presentation/lai-zeqi)), the simulator allows users to change inter-satellite topology, forwarding tables, constellation specification, packet-level network specifications, such as bandwidths, buffer sizes, and traffic matrix.
3. **Easy-to-use:** Comapred to StarryNet emulator, all the simulation can be executed on a single node, that avoids multiple nodes, and dockers builds. Moreover, instead of going through simulation files for configuration in Hypatia, the design requires a JSON file for consellation architecture, and a set of structred csv files for ISL topology, ground station locations, traffic matrix, and forwarding files.
4. **Link-aware:** One of the most critical misdesign of the Hypatia, and StarryNet([issue 21](https://github.com/SpaceNetLab/StarryNet/issues/21)) is the constant datarate during the simulation while the changing distances between satellites, and ground stations affect the datarate due to the alternating signal-to-noise ratio. The simulator considers the calculated distance based on the location of satellites, and ground stations, and uses Shannon-Hartley theorem to model the link's datarate.

## Requirements
1. [Go 1.22.1](https://tip.golang.org/doc/go1.22)
2. [Python 3.12.9](https://www.python.org/downloads/release/python-3129/)
3. [CesiumJS](https://cesium.com/platform/cesiumjs/)
4. [PyTorch 2.9.1](https://pytorch.org/) 
5. [NetworkX](https://networkx.org/)
6. [Pandas](https://pandas.pydata.org/)
7. [Numpy](https://numpy.org/)
8. [Matplotlib](https://matplotlib.org/)
9. [geopy 2.4.1](https://geopy.readthedocs.io/en/stable/index.html)
10. [CUDA Toolkit 12.9](https://developer.nvidia.com/cuda-12-9-0-download-archive) **[if you want GPU Acceleration, and if you have resources]**
11. [cuDF-v25.12.00](https://docs.rapids.ai/api/cudf/stable/) **[if you want GPU Acceleration, and if you have resources]**
12. [cugraph-cu12](https://pypi.nvidia.com) (Install corresponding version to CUDA Toolkit: cu{$CUDA_VERSION}) **[if you want GPU Acceleration, and if you have resources]**
13. [dask-cuda-25.12.00](https://docs.rapids.ai/api/dask-cuda/stable/)  **[if you want GPU Acceleration, and if you have resources]**

## Input Files
1. **Ground Station Locations:** The file should be located in the `./configs` folder. The file has a _CSV_ format with the following column structure: ``Id,Latitude,Longitude``, where `Id` is the unique name of the ground station, `Latitude` is the latitude of the corresponding ground station, and `Longitude` is the longitude for the corresponding ground station.
2. **Traffic Matrix File:** The file should be located in the `./input` folder. The file has a _CSV_ format with the following column structure: ``Timestamp(ms),Source,Destination,Length(Mb)``, where `Timestamp` is the amount of time in milliseconds since the beginning of the simulation, `Source` is the identifier for a ground station sending data, `Destination` is the identifier for a ground station the corresponding data intended, and `Length(Mb)` is the amount of data in Mbs.
3. **ISL Topology File:** The file should be located in the `./input` folder. The file has a _CSV_ format with the following column structure: ``FirstSatellite, SecondSatellite``,
where each pair containing an edge of the inter-satellite static topology. It is important to include both $$(S_1, S_2)$$, and $$(S_2, S_1)$$ in the topology description where $$S_1$$, and $$S_2$$ are the unique satellite identifiers. The format of satellite identifiers is $$"[Constellation Name]-i-j"$$ where the indexes are $$0 \le i < N_O$$, and $$0 \le j < N_S$$. $$N_O$$, and $$N_S$$ are number of orbits, and number of satellites per orbit respectively.
4. **Forwarding Table:** There should be a file dedicated to each ground station, and satellite's forwarding table. The files should be located in a folder inside the `./forwarding_table`. The files have a _CSV_ format with the following column structure: `TimeStamp,Source,Destination,NextHop` where `Timestamp` is the amount of time in milliseconds since the beginning of the simulation, `Source` is the identifier for a ground station or satellite sending data, `Destination` is the identifier for a ground station or satellite the corresponding data intended, and `NextHop` is the best next choice to forward the data to.
5. **Constellation Configuration File:** The file should be located in the `./configs` folder. The file has a _JSON_ format with the following structure: <br>
   <code>{
  "name": "Starlink",
  "use_gpu": true,
  "coordination_interval": 1.0,
  "orbit_config": {
      "earth_radius": 6378135.0,
      "earth_rotation_period": 1.0,
      "altitude": 550000.0,
      "min_altitude_isl": 80000.0,
      "inclination": 53,
      "min_ascension_angle": 0.0,
      "max_ascension_angle": 355.0,
      "number_of_orbits": 72,
      "number_of_satellites_per_orbit": 22,
      "phase_diff_enabled": true
  },
  "satellite_config": {
      "speed_of_light_vac": 299792458.0,
      "mean_motion_rev_per_day": 15.19,
      "min_elevation_angle": 25.0,
      "number_of_isls": 4,
      "isl_bandwidth": 1000000.0,
      "isl_link_noise_coef": 1500.0,
      "isl_acquisition_time": 12.0,
      "gsl_bandwidth": 100000.0,
      "gsl_link_noise_coef": 400.0,
      "max_packet_size": 12.0,
      "interface_buffer_size": 1000
  }
  }</code> <br>
  The `name` field refers to constellation's name, e.g. "Starlink". The `use_gpu` enables simulatior to utilize GPUs, if there is any available. The `coordination_interval` in milliseconds is the coordination interval for packets to get reordered, a parameter to increase for faster simulation; hence you will have at most this interval as your error.
  The rest of the structure can be split in two section:
  ### orbit_config:
  - **`earth_radius(m)`:** A constant refering to the Earth's radius, that should not be changed for a realistic Earth simulation.
  - **`earth_rotation_period(rev/day)`:** A constant refering to the Earth's period in revolutions per day, that should not be changed for a realistic Earth simulation.
  - **`altitude(m)`:** The altitude of the satellites in a constellation's shell in meters, based on the Starlink v1 Shell 1 altitude (check [Everyday's Astronaut Website](https://everydayastronaut.com/starlink-group-6-15-falcon-9-block-5-2/).) Moreover, it should be synchornozied with `mean_motion_rev_per_day` due to dynamic Physics of gravity.
  - **`min_altitude_isl(m)`:** A constant refering to minimum altitude an inter-satellite connection can be achieved due to undeterministic atmospheric condition. It should not be changed for a realistic Earth simulation.
  - **`inclination(degrees)`:** The inclination of the constellation's orbit, that is similiar for all of the orbits in a constellation (check [Orbital Elements definitions](https://en.wikipedia.org/wiki/Orbital_elements).) The value is based on the Starlink v1 Shell 1 inclination (check [Everyday's Astronaut Website](https://everydayastronaut.com/starlink-group-6-15-falcon-9-block-5-2/).)
  - **`[min_ascension_angle, max_ascension_angle](degrees)`:** The closed range of ascension degrees that the unqiue ascension of each orbit is selected uniformly, based on the number of orbits. (check [Orbital Elements definitions](https://en.wikipedia.org/wiki/Orbital_elements).)
  - **`number_of_orbits`:** The number of orbits in the constellation's shell, based on the Starlink Phase v1 number of orbits (check [Everyday's Astronaut Website](https://everydayastronaut.com/starlink-group-6-15-falcon-9-block-5-2/).)
  - **`number_of_satellites_per_orbit`:** The number of satellites per orbit in the constellation's shell, based on the Starlink Phase v1 number of satellite's per orbit (Check [Everyday's Astronaut Website](https://everydayastronaut.com/starlink-group-6-15-falcon-9-block-5-2/).)
  - **`phase_diff_enabled`:** A true/false value that alternatively(odd, and even indexes of orbits) shift the first satellite for half a phase, creating an initial state modification in the distances for the simulator. 
  ### satellite_config:
  - **`speed_of_light_vac(m/s)`:** A constant refering to the speed of light in vaccum for wireless signal transmission, that should not be changed for a realistic Earth simulation.
  - **`mean_motion_rev_per_day(rev/day)`:** Mean of satellite's period around the Earth in revolutions per day, that should be synchornozied with `altitude` due to dynamic Physics of gravity.
  - **`min_elevation_angle(degrees)`:** The minimum degree between the horizon, and the satellite-to-ground-station connecting line, controlling the quality of ground-to-satellite transmission due to the shadowing effect, and obstructions. The minimum degree should be $$5^{\circ}-10^{\circ}$$ according to the regulations (check [Cornell Law School Website](https://www.law.cornell.edu/cfr/text/47/25.205);) however, for an efficient, and reliable transmission, $$25^{\circ}$$ is recommended.
  - **`number_of_isls`:** Number of inter-satellite links allowed for each satellite corresponding to Laser Communication Terminal. The number is 4 for Starlink V2 satellites (Check [Starlink Website](https://www.starlink.com/technology))
  - **`isl_bandwidth(Symps), isl_link_noise_coef`:** Parameters tuned for distance-based calculation of the inter-satellite throughput based on Shannon's Law.
  - **`isl_acquisition_time(s)`:** Amount of time it takes to establish an inter-satellite connection in seconds. It is at least 10 seconds for Starlink satellites (Check [SpaceX presentation](https://www.spiedigitallibrary.org/conference-proceedings-of-spie/12877/1287702/Achieving-99-link-uptime-on-a-fleet-of-100G-space/10.1117/12.3005057.short).)
  - **`gsl_bandwidth(Symps), gsl_link_noise_coef`:** Parameters tuned for distance-based calculation of the ground-satellite throughput based on Shannon's Law.
  - **`max_packet_size(Kb)`:** Size of the network packets in Kb.
  - **`interface_buffer_size`:** Number of packets each interface can buffer in time before packet drop happens.

## Output Files
  1. **Distances:** In the "distances" mode, the simulator generates satellite, and ground station distances that are in the range of each other according to the given configuration. The distances file can be used for shortest path calculation for fowarding tables, or for topology design purposes. The distances output file will be located in the `./generated` folder. The file has a _CSV_ format with the following column structure: ``TimeStamp(ms),FirstDeviceId,SecondDeviceId,Distance(m)``, where `Timestamp` is the amount of time in milliseconds since the beginning of the simulation, `(FirstDeviceId,SecondDeviceId)` are the identifiers for the pair of satellite or ground station in range, `Distance` is the distance in meters.
  2. **Simulation Summary:** In the "simulation" mode, the simulator generates all the network events in the packet-level. The simulation summary output file will be located in the `./generated` folder. The file has a _CSV_ format with the following column structure: ``TimeStamp(ms),Event,FromDevice,ToDevice,PacketId``, where `Timestamp` is the amount of time in milliseconds since the beginning of the simulation, `Event` can be one of the "DELIVERED", "SEND", "RECEIVE", "DROP", `(FromDevice,ToDevice)` are the pair of satellite-to-ground-station or satellite-to-satellite that the event is related to, `PacketId` is the unique identifier of each packet in the network. When the `Event` is "CONNECTION_ESTABLISHED", the `PacketId` value is -1, since the event is unrelated to packet forwarding.

## Shortest Path Calculator
There are python files in the `./shortest_path_forwarding` folder to generate shortest path forwarding table using the all-pair Dijkstra algorithm. You can run the shortest path forwarding with the following commands; **Note that,** the distance file is expected to be at the `./generated` folder, and the topology file is expected to be at the `./input` folder:
> `python ./shortest_path_forwarding/shortest_path{_gpu}.py --dijkstra [distance file]`

> `python ./shortest_path_forwarding/shortest_path{_gpu}.py --dijkstra_grid_plus [distance file] [number of orbits] [number of satellites per orbit]`

> `python ./shortest_path_forwarding/shortest_path{_gpu}.py --dijkstra_static [distance file] [topology_file_static]`

> `python ./shortest_path_forwarding/shortest_path{_gpu}.py --dijkstra_dynamic [distance file] [topology_file_dynamic]`
- **--dijkstra:** shortest path without any inter-satellite topology, based on the distance file only
- **--dijkstra_grid_plus:** shortest path with grid plus inter-satellite topology, and the corresponding distances
- **--dijkstra_static:** shortest path with a static inter-satellite topology, and the corresponding distances
- **--dijkstra_dynamic:** shortest path with a dynamic inter-satellite topology, and the corresponding distances

## Traffic Generator
### Trafic Distribution Generator
The module generates ground station to ground station traffic based on a source-destination ground stations, or pair-wise ground stations with a `ground_station_file`(in `./configs` folder) for a period of time. For population or distance based traffics corresponding data should be provided. Moreover, for a fair non-overwhelming load on the ground-satellite link, buffer size, packet length and packet transmission time should be provided. The output traffic would be in `./input` folder: 
> `python ./traffic_pattern_generator/generate_traffic.py --single_uniform [source] [destination] [buffer_size] [packet_length(Kb)] [packet_transmission_time(ms)] [time_period(s)]"`

> `python ./traffic_pattern_generator/generate_traffic.py --uniform [ground_station_file] [buffer_size] [packet_length(Kb)] [packet_transmission_time(ms)] [time_period(s)]"`

> `python ./traffic_pattern_generator/generate_traffic.py --exponential_hotspot [ground_station_file] [buffer_size] [packet_length(Kb)] [packet_transmission_time(ms)] [time_period(s)] ([decay_param])"`

> `python ./traffic_pattern_generator/generate_traffic.py --distance [ground_station_file] [buffer_size] [packet_length(Kb)] [packet_transmission_time(ms)] [time_period(s)]"`

> `python ./traffic_pattern_generator/generate_traffic.py --population [ground_station_population_file] [buffer_size] [packet_length(Kb)] [packet_transmission_time(ms)] [time_period(s)]"`

> `python ./traffic_pattern_generator/generate_traffic.py --distance_population [ground_station_population_file] [buffer_size] [packet_length(Kb)] [packet_transmission_time(ms)] [time_period(s)]"`

> `./traffic_pattern_generator/generate_traffic.py --distort_gaussian [demand_file] [packet_size(Kb)] [mean(mu)] [stddev(sigma)]"`
- **--single_uniform:** generates a uniform traffic matrix between a source and destination 
- **--uniform:** generates a uniform traffic matrix
- **--exponential_hotspot:** generates a probabilistic weighted matrix with larger weights at the top left side of the matrix (hotspots), with $$e^{-\frac{i+j}{m}}$$, where $$i$$, and $$j$$ are the row and column of the matrix, and $$m$$ number of ground stations
- **--distance:** generates a distance-weighted probabilistic traffic matrix
- **--population:** generates a population-weighted traffic matrix
- **--distance_population:** generates a population&distance-weighted traffic matrix
- **--distort_guassian:** distorts a given demand matrix with packet size scale noise of $$\mathcal{N}(\mu, \sigma)$$. Demand file should be in the `./input` folder.

### Traffic Analyzer
The module analyzes traffic demand and shows latitude-longitude directions of aggregate traffic geodesics passing through regions of Earth, where regions are obtained by breaking Earth into a mesh structure through latitude and longitude uniform split. It expects demand file (in `./input` folder), ground station file (in `./configs` folder), and corresponding number of latitude and longitude split lines. The output regional analysis would be in `./generated` folder: 
> `./traffic_pattern_generator/demand_analyzer.py --geometry [demand_file] [ground_station_file] [latitude_lines] [longitude_lines]"`

## Topology Generator
The module generates inter-satellite topology for a regular number of ISLs. It needs distances or consistent distances file (in `./generated` folder), cartesian position file (in `./generated` folder), and demand file (in `./input` folder), and the inclination of the constellation. The output topology would be in the `./input` folder:
> `python ./topology/topology_generator.py --random_static [distance_file] [number of ISLs]"`

> `python ./topology/topology_generator.py --motif [distance_file] [initial_distances] [ground_station_positions] [alpha] [number of ISLs]`

> `python ./topology/topology_generator.py --riemannian_static [distance_file] [cartesian_positions_file] [demand_matrix_file] [number of ISLs] [inclination(deg)]`

> `python ./topology/topology_generator.py --riemannian_dynamic [distance_file] [cartesian_positions_file] [demand_matrix_file] [number of ISLs] [inclination(deg)] [time_period(s)] [time_interval(s)]`

> `python ./topology/topology_generator.py --riemannian_fields [cartesian_positions_file] [source] [destination] [inclination(deg)] [time_period(s)] [time_interval(s)]`
- **--random_static:** A random topology based on random inter-orbital pattern matching, and intra-orbital choices
- **--random_static:** A Motif-based topology proposed by [Bhattacherjee et. al.](https://dl-acm-org.proxy.lib.ohio-state.edu/doi/10.1145/3359989.3365407)
- **--riemannian_static:** The Riemann metric based static topology generator
- **--riemannian_dynamic:** The Riemann metric based dynamic topology generator for a period with time intervals
- **--riemannian_fields:** The field, for Riemann metric, direction and magnitude calculator for a period with time intervals

## Analyzer

**Analyze.py** is a Python utility script that performs latency analysis and visualization based on the simulation summary generated by the simulator, located in the `./generated` folder. This tool helps researchers quickly understand the distribution of key network performance metrics such as latency, hop count, RTT, and stretch factor. It will generate two CSV files located in `.result/AnalysisOfSimulationSummary/` and automatically call the `plot.py` script to produce the following plots:

1. `Hop Stretch Factor CDF Plot`: Visualizes the distribution of the **stretch factor** across different city pairs in the simulation. The **stretch factor** is the ratio between the actual latency and the latency calculated based on the geodesic distance divided by the speed of light. This plot helps us understand how efficiently packets travel through the network by comparing to idealized 'light-speed' baseline.
    - ` X-axis: Stretch factor values  `
    - `Y-axis: Cumulative distribution function (CDF), showing the proportion of city pairs with a stretch factor`

2. `RTT CDF Plot`: Shows the distribution of **round-trip time (RTT)** across the network. RTT represents the time it takes for a packet to travel from the source ground station to the destination ground station and back. This plot helps assess the latency performance of the network and identify delays or bottlenecks.

   - `X-axis: RTT values (in milliseconds)`
   - `Y-axis: Cumulative distribution function (CDF), showing the proportion of city pairs with an RTT`

3. `Link Usage Plot`: Displays how many packets traversed each satellite link during the simulation. It reveals the traffic load distribution in the satellite network.

   - `X-axis: The rank of each link based on how frequently it was used (from most used to least used). `
   - `Y-axis: Shows how many packets traversed that link during the simulation.`
---

The output CSV files follow these column structures:

`overall.csv`
This file contains the following columns:

- `FromDevice` and `ToDevice`: The city that sends and receives the packet.
- `Latency_ms`: The time (in milliseconds) it takes for a packet to transfer from source to destination.
- `Hop_Count`: The average number of hops that packets take from source ground station to the destination ground station.
- `Stretch_factor`: The ratio between the distance of the routed path and the geodesic distance.
- `Effective_Latency_Factor`: The ratio between the average latency and the total propagation delay derived form the stretch factor. This metric signifies the effect of queuing and transmission delay on latency compared to the propagation delay.
- `RTT_ms`: Round-trip time between the source and destination ground stations.
- `Jitter`: The standard deviation of packet latency (`Latency_ms`) for each city pair.
- `TotalPackets`: Total number of packets routed from the source to destination

`link_usage.csv`
This file contains the following columns:

- `link`: A tuple of (SatelliteA, SatelliteB), indicating which two satellites form this link.
- `SatelliteA` and `SatelliteB`: The name of the satellite link.
- `UsageCount`: The number of packets traversed this satellite link during the simulation.
---
You can run the analyzer with the following commands:
> `python log_analyzer/analyze.py [ground_stations_file] [ground_stations_file] [distances_file]`

## Visualizer
To visualize the simulator, fields, and traffic demands, we employed [CesiumJS](https://cesium.com/platform/cesiumjs/), a 3D geospatial modeling Javascript application. To access Cesium, you need a personal token, that can be obtained free of charge by registering in [Cesium](https://cesium.com/learn/ion/cesium-ion-access-tokens/). To run the visualizer open the following HTML files:
- **`packet_visualizer.html`**: Visualizes a packet route in space with underlying inter-satellite topology for a simulation summary
- **`field_visualizer.html`**: Visualizes fields calculated for Riemannian topology generation
- **`demand_visualizer.html`**: Visualizes traffic demands as geodesic curves


## How to Run
Assuming all the input files are located in their expected folders, the simulator can be run using the following commands: 
> `go run main.go --positions --[cartesian/spherical] [constellation config file] [ground station locations] [time step (ms)] [total simulation time (s)]`

> `go run main.go --distances [consellation config file] [ground station locations] [time step (ms)] [total simulation time (s)]`

> `go run main.go --forwarding [consellation config file] [ground station locations] [traffic generator file] [forwarding folder] [ISL Topology] [time step (ms)] [total simulation time (s)]`

> `go run main.go --forwarding --grid_plus [consellation config file] [ground station locations] [traffic generator file] [forwarding folder] [time step (ms)] [total simulation time (s)]`

Where `time step (ms)` is the amount of time in milliseconds that the simulator sees the distances stable, and `total simulation time (s)` is the amount of time in seconds that the simulator is being executed. The rest are the names of the input files.
- **--positions:** generates cartesian/spherical positions for each satellite and ground station
- **--distances:** runs the "distances" mode, and generates satellite to satellite and satellite to ground station distances 
- **--forwarding:** runs the simulator, and generates network packet-level events order by timestamp
- **--forwarding --grid_plus:** runs the simulator, and generates network packet-level events; however, it does not need inter-satellite topology input file, because the simulator builds a grid plus topology itself.
