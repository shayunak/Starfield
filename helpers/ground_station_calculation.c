#include "ground_station_calculation.h"
#include "orbital_calculation.h"
#include "anomaly_calculation.h"

#include <stdio.h>
#include <stdlib.h>
#include <math.h>
#include <string.h>

void update_distance_with_altitude(double* distances, int i, double altitude, double earth_orbit_ratio) 
{
    distances[i] = sqrt(pow(altitude, 2.0) + earth_orbit_ratio * pow(distances[i], 2.0));
}

double calculate_gs_distance(double radius, double inclination_cosinus, double inclination_sinus, anomaly_elements head_point_anomaly_el, 
    double head_point_ascension, double anomaly, double ascension) 
{
    double ascension_diff = head_point_ascension - ascension;
    
    orbit_calc orb_calc;
    orb_calc.cosinal_coefficient = calculate_cosinal_coefficient(inclination_cosinus, head_point_anomaly_el, ascension_diff);
    orb_calc.sinal_coefficient = calculate_sinal_coefficient(inclination_cosinus, inclination_sinus, head_point_anomaly_el, ascension_diff);
    orb_calc.ascension_diff = ascension_diff;
    
    return calculate_distance(radius, orb_calc, anomaly);
}

void covering_ground_station(ground_station_calculation* gsc, double head_point_ascension, anomaly_elements head_point_anomaly_el, 
    double anomaly, double time_stamp, double earth_rotation_motion, double earth_orbit_ratio, double ascension, double altitude, 
    double* distances, char gs_in_range[][MAX_ID_LENGTH], int* count, const char* gs_name) 
{
    double gs_ascension = head_point_ascension + earth_rotation_motion * time_stamp;
    double distance = calculate_gs_distance(gsc->radius, gsc->inclination_cosinus, gsc->inclination_sinus, head_point_anomaly_el, gs_ascension, anomaly, ascension);
    
    if (distance < gsc->ground_stations_distance_limit) {
        double updated_distance = sqrt(pow(altitude, 2.0) + earth_orbit_ratio * pow(distance, 2.0));
        distances[*count] = updated_distance;
        strncpy(gs_in_range[*count], gs_name, MAX_ID_LENGTH - 1);
        gs_in_range[*count][MAX_ID_LENGTH - 1] = '\0';
        (*count)++;
    }
}

double update_gs_position(double prevAscension,
    double timeStep,
    double earthRotationMotion
) {
	return prevAscension + earthRotationMotion*timeStep;
}

void covering_ground_stations(ground_station_calculation* gsc, 
    double anomaly, 
    double time_stamp, 
    double earth_orbit_ratio, 
    double ascension, 
    double altitude, 
    double* distances, 
    char gs_in_range[][MAX_ID_LENGTH], 
    int* count
) {
    for (int i = 0; i < gsc->station_count; i++) {
        covering_ground_station(gsc,
            gsc->station_specs[i].head_point_ascension,
            gsc->station_specs[i].head_point_anomaly_el,
            anomaly,
            time_stamp,
            gsc->earth_rotation_motion,
            earth_orbit_ratio,
            ascension,
            altitude,
            distances,
            gs_in_range,
            count,
            gsc->station_names[i]
        );
    }
}

void update_distances_with_altitude(double* distances, 
    int count, 
    double altitude, 
    double earth_orbit_ratio
) {
    for (int i = 0; i < count; i++) {
        update_distance_with_altitude(distances, i, altitude, earth_orbit_ratio);
    }
}


