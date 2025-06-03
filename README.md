# SatSimGo
The following project is a packet-level routing multi-process simulator for Low Earth Orbit (LEO) satellite networks. The network consists of satellites, and ground stations, that generate traffic. The main purpose of the simulator compared to state-of-the-art similar simulators, such as, Hypatia ([Kassing, 2022](https://dl.acm.org/doi/abs/10.1145/3419394.3423635)), is to build an easy-to-use, fast, light, configurable, and link-aware system that supports a shell of specified constellation. The simulator achieves the asserted proporties through:

1. **Fast, and Light:** The simulator integrates inter-satellite, and satellite-ground-station astronomical distance calculation with packet queuing and forwarding in a goroutine (Golang lightweight process) to manage thousands of satellite, and ground stations with minimal overhead due to memory efficiency, and small stack size. Moreover, by solving trigonomical astronomical equations, a linear time complexity can be achieved for inter-satellite distances compared to quadratic pair-wise approach.
2. **Configurable:** Unlike StarPerf ([Lai, 2020](https://ieeexplore.ieee.org/document/9259357)), and StarryNet([Lai, 2023](https://www.usenix.org/conference/nsdi23/presentation/lai-zeqi)), the simulator allows users to change inter-satellite topology, forwarding tables, constellation specification, packet-level network specifications, such as bandwidths, buffer sizes, and traffic matrix.
3. **Easy-to-use:** Comapred to StarryNet emulator, all the simulation can be executed on a single node, that avoids multiple nodes, and dockers builds. Moreover, instead of going through simulation files for configuration in Hypatia, the design requires a JSON file for consellation architecture, and a set of structred csv files for ISL topology, ground station locations, traffic matrix, and forwarding files.
4. **Link-aware:** One of the most critical misdesign of the Hypatia, and StarryNet([issue 21](https://github.com/SpaceNetLab/StarryNet/issues/21)) is the constant datarate during the simulation while the changing distances between satellites, and ground stations affect the datarate due to the alternating signal-to-noise ratio. The simulator considers the calculated distance based on the location of satellites, and ground stations, and uses Shannon-Hartley theorem to model the link's datarate.

## Requirements
1. [Go 1.22.1](https://tip.golang.org/doc/go1.22)
2. [Pyhton 3.7.2](https://www.python.org/downloads/release/python-372/)
3. [NetworkX](https://networkx.org/)
4. [Pandas](https://pandas.pydata.org/)
5. [Numpy](https://numpy.org/)
6. [Matplotlib](https://matplotlib.org/)

## Input Files
1. **Ground Station Locations:** The file should be located in the `./configs` folder. The file has a _CSV_ format with the following column structure: ``Id,Latitude,Longitude``, where `Id` is the unique name of the ground station, `Latitude` is the latitude of the corresponding ground station, and `Longitude` is the longitude for the corresponding ground station.
2. **Traffic Matrix File:** The file should be located in the `./input` folder. The file has a _CSV_ format with the following column structure: ``Timestamp(ms),Source,Destination,Length(Mb)``, where `Timestamp` is the amount of time in milliseconds since the beginning of the simulation, `Source` is the identifier for a ground station sending data, `Destination` is the identifier for a ground station the corresponding data intended, and `Length(Mb)` is the amount of data in Mbs.
3. **ISL Topology File:** The file should be located in the `./input` folder. The file has a _CSV_ format with the following column structure: ``FirstSatellite, SecondSatellite``,
where each pair containing an edge of the inter-satellite static topology. It is important to include both $$(S_1, S_2)$$, and $$(S_2, S_1)$$ in the topology description where $$S_1$$, and $$S_2$$ are the unique satellite identifiers. The format of satellite identifiers is $$"[Constellation Name]-i-j"$$ where the indexes are $$0 \le i < N_O$$, and $$0 \le j < N_S$$. $$N_O$$, and $$N_S$$ are number of orbits, and number of satellites per orbit respectively.
4. **Forwarding Table:** There should be a file dedicated to each ground station, and satellite's forwarding table. The files should be located in a folder inside the `./forwarding_table`. The files have a _CSV_ format with the following column structure: `TimeStamp,Source,Destination,NextHop` where `Timestamp` is the amount of time in milliseconds since the beginning of the simulation, `Source` is the identifier for a ground station or satellite sending data, `Destination` is the identifier for a ground station or satellite the corresponding data intended, and `NextHop` is the best next choice to forward the data to.
5. **Constellation Configuration File:** The file should be located in the `./configs` folder. The file has a _JSON_ format with the following structure: <br>
   <code>{
  "name": "Starlink",
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
      "min_elevation_angle": 30.0,
      "number_of_isls": 4,
      "isl_bandwidth": 1000000.0,
      "isl_link_noise_coef": 1500.0,
      "isl_acquisition_time": 12.0,
      "gsl_bandwidth": 100000.0,
      "gsl_link_noise_coef": 400.0,
      "max_packet_size": 120.0,
      "interface_buffer_size": 100
  }
  }</code> <br>
  The `name` field refers to constellation's name, e.g. "Starlink". The rest of the structure can be split in two section:
  ### orbit_config:
  - **`earth_radius`:** A constant refering to the Earth's radius, that should not be changed for a realistic Earth simulation.
  - **`earth_rotation_period(rev/day)`:** A constant refering to the Earth's period in revolutions per day, that should not be changed for a realistic Earth simulation.
  - **`altitude(m)`:** The altitude of the satellites in a constellation's shell in meters, based on the Starlink v1 Shell 1 altitude (check [Everyday's Astronaut Website](https://everydayastronaut.com/starlink-group-6-15-falcon-9-block-5-2/).) Moreover, it should be synchornozied with `mean_motion_rev_per_day` due to dynamic Physics of gravity.
  - **`min_altitude_isl(m)`:** A constant refering to minimum altitude an inter-satellite connection can be achieved due to undeterministic atmospheric condition. It should not be changed for a realistic Earth simulation.
  - **`inclination(degrees)`:** The inclination of the constellation's orbit, that is similiar for all of the orbits in a constellation (check [Orbital Elements definitions](https://en.wikipedia.org/wiki/Orbital_elements).) The value is based on the Starlink v1 Shell 1 inclination (check [Everyday's Astronaut Website](https://everydayastronaut.com/starlink-group-6-15-falcon-9-block-5-2/).)
  - **`[min_ascension_angle, max_ascension_angle](degrees)`:** The closed range of ascension degrees that the unqiue ascension of each orbit is selected uniformly, based on the number of orbits. (check [Orbital Elements definitions](https://en.wikipedia.org/wiki/Orbital_elements).)
  - **`number_of_orbits`:** The number of orbits in the constellation's shell, based on the Starlink Phase v1 number of orbits (check [Everyday's Astronaut Website](https://everydayastronaut.com/starlink-group-6-15-falcon-9-block-5-2/).)
  - **`number_of_satellites_per_orbit`:** The number of satellites per orbit in the constellation's shell, based on the Starlink Phase v1 number of satellite's per orbit (Check [Everyday's Astronaut Website](https://everydayastronaut.com/starlink-group-6-15-falcon-9-block-5-2/).)
  - **`phase_diff_enabled`:** A true/false value that alternatively(odd, and even indexes of orbits) shift the first satellite for half a phase, creating an initial state modification in the distances for the simulator. 
  ### satellite_config:
  - **`speed_of_light_vac`:** A constant refering to the speed of light in vaccum for wireless signal transmission, that should not be changed for a realistic Earth simulation.
  - **`mean_motion_rev_per_day(rev/day)`:** Mean of satellite's period around the Earth in revolutions per day, that should be synchornozied with `altitude` due to dynamic Physics of gravity.
  - **`min_elevation_angle(degrees)`:** The minimum degree between the horizon, and the satellite-to-ground-station connecting line, controlling the quality of ground-to-satellite transmission due to the shadowing effect, and obstructions. The minimum degree should be $$5^{\circ}-10^{\circ}$$ according to the regulations (check [Cornell Law School Website](https://www.law.cornell.edu/cfr/text/47/25.205?utm_source=chatgpt.com);) however, for an efficient, and reliable transmission, $$30^{\circ}$$ is recommended.
  - **`number_of_isls`:** Number of inter-satellite links allowed for each satellite corresponding to Laser Communication Terminal. The number is 4 for Starlink V2 satellites (Check [Starlink Website](https://www.starlink.com/technology))
  - **`isl_bandwidth, isl_link_noise_coef`:** Parameters tuned for distance-based calculation of the inter-satellite throughput based on Shannon's Law.
  - **`isl_acquisition_time(s)`:** Amount of time it takes to establish an inter-satellite connection in seconds. It is at least 10 seconds for Starlink satellites (Check [SpaceX presentation](https://www.spiedigitallibrary.org/conference-proceedings-of-spie/12877/1287702/Achieving-99-link-uptime-on-a-fleet-of-100G-space/10.1117/12.3005057.short).)
  - **`gsl_bandwidth, gsl_link_noise_coef`:** Parameters tuned for distance-based calculation of the ground-satellite throughput based on Shannon's Law.
  - **`max_packet_size(Kb)`:** Size of the network packets in Kb.
  - **`interface_buffer_size`:** Number of packets each interface can buffer in time before packet drop happens.
## Output Files
  1. **Distances:** In the "distances" mode, the simulator generates satellite, and ground station distances that are in the range of each other according to the given configuration. The distances file can be used for shortest path calculation for fowarding tables, or for topology design purposes. The distances output file will be located in the `./generated` folder. The file has a _CSV_ format with the following column structure: ``TimeStamp(ms),FirstDeviceId,SecondDeviceId,Distance(m)``, where `Timestamp` is the amount of time in milliseconds since the beginning of the simulation, `(FirstDeviceId,SecondDeviceId)` are the identifiers for the pair of satellite or ground station in range, `Distance` is the distance in meters.
  2. **Simulation Summary:** In the "simulation" mode, the simulator generates all the network events in the packet-level. The simulation summary output file will be located in the `./generated` folder. The file has a _CSV_ format with the following column structure: ``TimeStamp(ms),Event,FromDevice,ToDevice,PacketId``, where `Timestamp` is the amount of time in milliseconds since the beginning of the simulation, `Event` can be one of the "DELIVERED", "SEND", "RECEIVE", "DROP", "CONNECTION_ESTABLISHED", `(FromDevice,ToDevice)` are the pair of satellite-to-ground-station or satellite-to-satellite that the event is related to, `PacketId` is the unique identifier of each packet in the network. When the `Event` is "CONNECTION_ESTABLISHED", the `PacketId` value is -1, since the event is unrelated to packet forwarding.
## Shortest Path Calculator
There are python files in the `./shortest_path_forwarding` folder to generate shortest path forwarding table using the all-pair Dijkstra algorithm. You can run the shortest path forwarding with the following commands; **Note that,** the distance file is expected to be at the `./generated` folder, and the topology file is expected to be at the `./input` folder:
> `shortest_path_algorithm.py --dijkstra [distance file]`

> `shortest_path_algorithm.py --dijkstra_grid_plus [distance file] [number of orbits] [number of satellites per orbit]`

> `shortest_path_algorithm.py --dijkstra_static [distance file] [topology_file_static]`

> `shortest_path_algorithm.py --dijkstra_dynamic [distance file] [topology_file_dynamic]`
- **--dijkstra:** shortest path without any inter-satellite topology, based on the distance file only
- **--dijkstra_grid_plus:** shortest path with grid plus inter-satellite topology, and the corresponding distances
- **--dijkstra_static:** shortest path with a static inter-satellite topology, and the corresponding distances
- **--dijkstra_dynamic:** shortest path with a dynamic inter-satellite topology, and the corresponding distances
## How to Run
Assuming all the input files are located in their expected folders, the simulator can be run using the following commands: 
> `main.go --distances [consellation config file] [ground station locations] [time step (ms)] [total simulation time (s)]`

> `main.go --forwarding [consellation config file] [ground station locations] [traffic generator file] [forwarding folder] [ISL Topology] [time step (ms)] [total simulation time (s)]`

> `main.go --forwarding --grid_plus [consellation config file] [ground station locations] [traffic generator file] [forwarding folder] [time step (ms)] [total simulation time (s)]`

Where `time step (ms)` is the amount of time in milliseconds that the simulator sees the distances stable, and `total simulation time (s)` is the amount of time in seconds that the simulator is being executed. The rest are the names of the input files.
- **--distances:** runs the "distances" mode, and generates distances
- **--forwarding:** runs the simulator, and generates network packet-level events
- **--forwarding --grid_plus:** runs the simulator, and generates network packet-level events; however, it does not need inter-satellite topology input file, because the simulator builds a grid plus topology itself.
