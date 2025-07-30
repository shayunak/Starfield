#ifndef GROUND_STATION_CALCULATIONS_H
#define GROUND_STATION_CALCULATIONS_H

#include "anomaly_calculation.h"
#include "orbital_calculation.h"

#define MAX_GROUND_STATIONS 1000

typedef struct {
    double latitude;
    double longitude;
    double head_point_ascension;
    anomaly_elements head_point_anomaly_el;
} ground_station_spec;

typedef struct {
    double elevation_limit_ratio;
    double altitude;
    double earth_orbit_ratio;
    double earth_rotation_motion;
    double ground_stations_distance_limit;
    double radius;
    double inclination_cosinus;
    double inclination_sinus;
    char station_names[MAX_GROUND_STATIONS][MAX_ID_LENGTH];
    ground_station_spec station_specs[MAX_GROUND_STATIONS];
    int station_count;
} ground_station_calculation;

void covering_ground_stations(ground_station_calculation* gsc, 
    double anomaly, 
    double time_stamp, 
    double earth_orbit_ratio, 
    double ascension, 
    double altitude, 
    double* distances, 
    char gs_in_range[][MAX_ID_LENGTH], 
    int* count
);

void update_distances_with_altitude(double* distances, 
    int count, 
    double altitude, 
    double earth_orbit_ratio
);

double update_gs_position(double prevAscension,
    double timeStep,
    double earthRotationMotion
);

#endif