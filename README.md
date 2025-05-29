# SatSimGo
The following project is a packet-level routing multi-process simulator for Low Earth Orbit (LEO) satellite networks. The network consists of satellites, and ground stations, that generate traffic. The main purpose of the simulator compared to state-of-the-art similar simulators, such as, Hypatia ([Kassing, 2022](https://dl.acm.org/doi/abs/10.1145/3419394.3423635)), is to build an easy-to-use, fast, light, configurable, and link-aware system that supports a shell of specified constellation. The simulator achieves the asserted proporties through:

1. **Fast, and Light:** The simulator integrates inter-satellite, and satellite-ground-station astronomical distance calculation with packet queuing and forwarding in a goroutine (Golang lightweight process) to manage thousands of satellite, and ground stations with minimal overhead due to memory efficiency, and small stack size. Moreover, by solving trigonomical astronomical equations, a linear time complexity can be achieved for inter-satellite distances compared to quadratic pair-wise approach.
2. **Configurable:** Unlike StarPerf ([Lai, 2020](https://ieeexplore.ieee.org/document/9259357)), and StarryNet([Lai, 2023](https://www.usenix.org/conference/nsdi23/presentation/lai-zeqi)), the simulator allows users to change inter-satellite topology, forwarding tables, constellation specification, packet-level network specifications, such as bandwidths, buffer sizes, and traffic matrix.
3. **Easy-to-use:** Comapred to StarryNet emulator, all the simulation can be executed on a single node, that avoids multiple nodes, and dockers builds. Moreover, instead of going through simulation files for configuration in Hypatia, the design requires a JSON file for consellation architecture, and a set of structred csv files for ISL topology, ground station locations, traffic matrix, and forwarding files.
4. **Link-aware:**: One of the most critical misdesign of the Hypatia, and StarryNet([issue 21](https://github.com/SpaceNetLab/StarryNet/issues/21)) is the constant datarate during the simulation while the changing distances between satellites, and ground stations affect the datarate due to the alternating signal-to-noise ratio. The simulator considers the calculated distance based on the location of satellites, and ground stations, and uses Shannon-Hartley theorem to model the link's datarate.

## Requirements
1. [Go 1.22.1](https://tip.golang.org/doc/go1.22)
2. [Pyhton 3.7.2](https://www.python.org/downloads/release/python-372/)
3. [NetworkX](https://networkx.org/)
4. [Pandas](https://pandas.pydata.org/)

## Input Files
1. **Ground Station Locations:** The file should be located in the `./configs` folder. The file has a _CSV_ format with the following column structure: ``Id,Latitude,Longitude``, where `Id` is the unique name of the ground station, `Latitude` is the latitude of the corresponding ground station, and `Longitude` is the longitude for the corresponding ground station.
2. **Traffic Matrix File:** The file should be located in the `./input` folder. The file has a _CSV_ format with the following column structure: ``Timestamp(ms),Source,Destination,Length(Mb)``, where `Timestamp` is the amount of time in milliseconds since the beginning of the simulation, `Source` is the identifier for a ground station sending data, `Destination` is the identifier for a ground station the corresponding data intended, and `Length(Mb)` is the amount of data in Mbs.
3. **ISL Topology File:** The file should be located in the `./input` folder. The file has a _CSV_ format with the following column structure: ``FirstSatellite, SecondSatellite``,
where each pair containing an edge of the inter-satellite static topology. It is important to include both $$(S_1, S_2)$$, and $$(S_2, S_1)$$ in the topology description where $$S_1$$, and $$S_2$$ are the unique satellite identifiers. The format of satellite identifiers is $$"[Constellation Name]-i-j"$$ where the indexes are $$0 \le i < N_O$$, and $$0 \le j < N_S$$. $$N_O$$, and $$N_S$$ are number of orbits, and number of satellites per orbit respectively.
4. **Constellation Configuration File:** The file should be located in the `./configs` folder. The file has a _JSON_ format with the following structure: <br>
   <code>{
  "name": "Starlink",
  "orbit_config": {
      "earth_radius": 6378135.0,
      "earth_rotation_period": 1.0,
      "altitude": 550000.0,
      "min_altitude_isl": 80000.0,
      "inclination": 53,
      "min_ascension_angle": 0.0,
      "max_ascension_angle": 10.0,
      "number_of_orbits": 3,
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
  - **`altitude(m)`:** The altitude of the satellites in a constellation's shell in meters.
  - **`min_altitude_isl(m)`:** A constant refering to minimum altitude an inter-satellite connection can be achieved due to undeterministic atmospheric condition. It should not be changed for a realistic Earth simulation.
  - **`inclination`:** The inclination of the constellation's orbit, that is similiar for all of the orbits in a constellation. 
  ### satellite_config:
