#ifndef ANOMALY_CALCULATION_H
#define ANOMALY_CALCULATION_H

#include "orbital_calculation.h"

#define MAX_ID_LENGTH 64

typedef struct {
    char* constellation_name;
    int number_of_satellites_per_orbit;
    double anomaly_step;          // in radians
    double mean_motion;           // in radians per second
    double radius;                // in meters
    orbital_calculations orbital_calc;
    int phase_diff_enabled;
} anomaly_calculations;

void update_sat_position(double mean_motion, 
    double prev_anomaly, 
    double time_step, 
    double* new_orbital_anomaly, 
    anomaly_elements* new_anomaly_elements
);

double calculate_distance(double radius,
     orbit_calc orbit_calc, 
     double other_satellite_anomaly
    );

void find_satellites_in_range(const anomaly_calculations* anomaly_calc,
    const int* orbit,
    const orbit_calc* orbit_calc,
    int orbit_count,
    const char* base_id,
    double length_limit_ratio,
    double time_stamp,
    char satellite_ids[][MAX_ID_LENGTH],
    double* satellite_distances,
    int* count
);

#endif