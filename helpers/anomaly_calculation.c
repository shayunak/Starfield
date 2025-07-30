#include "anomaly_calculation.h"
#include "orbital_calculation.h"
#include <math.h>
#include <stdio.h>
#include <string.h>

void update_sat_position(double mean_motion, 
    double prev_anomaly, 
    double time_step, 
    double* new_orbital_anomaly, 
    anomaly_elements* new_anomaly_elements) {
        *new_orbital_anomaly = prev_anomaly + mean_motion * time_step;
        new_anomaly_elements->anomaly_sinus = sin(*new_orbital_anomaly);
        new_anomaly_elements->anomaly_cosinus = cos(*new_orbital_anomaly);
}

double calculate_distance(double radius,
     orbit_calc orbit_calc, 
     double other_satellite_anomaly) {
    double distance_squared_factor = 2.0 * (orbit_calc.cosinal_coefficient * cos(other_satellite_anomaly) - orbit_calc.sinal_coefficient * sin(other_satellite_anomaly) + 1.0);
    if (distance_squared_factor <= 0.0) {
        return 0.0;
    }
    return radius * sqrt(distance_squared_factor);
}

double calculate_phase(int phase_diff_enabled, double anomaly_step, int satellite_id, int orbit_id) 
{
    double phase = (double)satellite_id * anomaly_step;
    if (phase_diff_enabled && orbit_id % 2 == 1) {
        phase += anomaly_step / 2.0;
    }
    return phase;
}

double calculate_distance_by_satellite_id(const anomaly_calculations* anomaly_calc, 
    int first_satellite_id, 
    int first_satellite_orbit_id, 
    int second_satellite_id, 
    int second_satellite_orbit_id, 
    double time_stamp) {

    double first_phase = calculate_phase(anomaly_calc->phase_diff_enabled, anomaly_calc->anomaly_step, first_satellite_id, first_satellite_orbit_id);
    double second_phase = calculate_phase(anomaly_calc->phase_diff_enabled, anomaly_calc->anomaly_step, second_satellite_id, second_satellite_orbit_id);
    double phase_diff = first_phase - second_phase;
    
    double ascension_diff = (double)(first_satellite_orbit_id - second_satellite_orbit_id) * anomaly_calc->orbital_calc.ascension_step;
    
    double ascension_diff_sinus = sin(ascension_diff);
    double ascension_diff_cosinus = cos(ascension_diff);
    double phase_diff_sinus = sin(phase_diff);
    double phase_diff_cosinus = cos(phase_diff);

    double inclination_sinus = anomaly_calc->orbital_calc.inclination_sinus;
    double inclination_cosinus = anomaly_calc->orbital_calc.inclination_cosinus;
    double time_term_cosinus = cos(2.0 * anomaly_calc->mean_motion * time_stamp + first_phase + second_phase);
    
    double phase_diff_sinus_term = 2.0 * inclination_cosinus * ascension_diff_sinus * phase_diff_sinus;
    double phase_diff_cosinus_term = ((1 + pow(inclination_cosinus, 2.0)) * ascension_diff_cosinus + pow(inclination_sinus, 2.0)) * phase_diff_cosinus;
    double time_term = (1 - ascension_diff_cosinus) * pow(inclination_sinus, 2.0) * time_term_cosinus;
    
    return anomaly_calc->radius * sqrt(2.0 + phase_diff_sinus_term - phase_diff_cosinus_term + time_term);
}

void calculate_satellite_id_in_range(const anomaly_calculations* anomaly_calc,
    int orbit,
    orbit_calc orbit_calc,
    const char* base_id,
    double length_limit_ratio,
    double time_stamp,
    char satellite_ids[][MAX_ID_LENGTH],
    double* satellite_distances,
    int* count)
{
    double orbital_calc_size = sqrt(pow(orbit_calc.cosinal_coefficient, 2.0) + pow(orbit_calc.sinal_coefficient, 2.0));
    double limit_term = asin(length_limit_ratio / orbital_calc_size);
    double phase_term = atan2(orbit_calc.cosinal_coefficient, orbit_calc.sinal_coefficient);
    double lower_range = phase_term + limit_term;
    double upper_range = M_PI - limit_term + phase_term;
    double initial_phase_shift = 0.0;
    if (anomaly_calc->phase_diff_enabled && orbit % 2 == 1) {
        initial_phase_shift = anomaly_calc->anomaly_step / 2.0;
    }
    
    int lower_id = (int)ceil((lower_range - initial_phase_shift - time_stamp * anomaly_calc->mean_motion) / anomaly_calc->anomaly_step);
    int upper_id = (int)floor((upper_range - initial_phase_shift - time_stamp * anomaly_calc->mean_motion) / anomaly_calc->anomaly_step);
    
    for (int i = lower_id; i <= upper_id; i++) {
        double real_anomaly = (double)i * anomaly_calc->anomaly_step + initial_phase_shift + time_stamp * anomaly_calc->mean_motion;
        int real_id = (i + anomaly_calc->number_of_satellites_per_orbit) % anomaly_calc->number_of_satellites_per_orbit;
        char satellite_id[MAX_ID_LENGTH];
        snprintf(satellite_id, sizeof(satellite_id), "%s-%d-%d", anomaly_calc->constellation_name, orbit, real_id);
        
        if (strcmp(base_id, satellite_id) != 0) {
            strcpy(satellite_ids[*count], satellite_id);
            satellite_distances[*count] = calculate_distance(anomaly_calc->radius, orbit_calc, real_anomaly);
            (*count)++;
        }
    }
}

void find_satellites_in_range(const anomaly_calculations* anomaly_calc,
    const int* orbit,
    const orbit_calc* orbit_calc,
    int orbit_count,
    const char* base_id,
    double length_limit_ratio,
    double time_stamp,
    char satellite_ids[][MAX_ID_LENGTH],
    double* satellite_distances,
    int* count){
        for (int i = 0; i < orbit_count; i++) {
            calculate_satellite_id_in_range(anomaly_calc, 
                orbit[i], 
                orbit_calc[i], 
                base_id, 
                length_limit_ratio, 
                time_stamp, 
                satellite_ids, 
                satellite_distances, 
                count);
        }
}