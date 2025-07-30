#include "orbital_calculation.h"
#include <math.h>
#include <stdlib.h>
#include <stdio.h>

int is_orbit_angle_valid(double min_ascension_angle, double max_ascension_angle, double angle) {
    return angle >= min_ascension_angle && angle <= max_ascension_angle;
}

double convert_orbit_id_to_ascension(double min_ascension_angle, double ascension_step, int orbit_id) {
    return (double)orbit_id * ascension_step + min_ascension_angle;
}

void calculate_limits(double inclination_cosinus, double inclination_sinus, double length_limit_ratio, double anomaly_sinus, double* LD, double* LU) 
{
    double ISL_length_limit = sqrt(1.0 - pow(length_limit_ratio, 2.0));
    double base_trig = inclination_sinus * inclination_cosinus * anomaly_sinus;
    double denominator = inclination_sinus * sqrt(1.0 - pow(anomaly_sinus * inclination_sinus, 2.0));
    double lower_limit = (base_trig - ISL_length_limit) / denominator;
    double upper_limit = (base_trig + ISL_length_limit) / denominator;
    *LU = M_PI / 2.0;
    *LD = -M_PI / 2.0;
    
    if (lower_limit > -1.0 && lower_limit < 1.0) {
        *LD = asin(lower_limit);
    } else if (lower_limit > 1.0) {
        *LD = M_PI / 2.0;
    }
    
    if (upper_limit > -1.0 && upper_limit < 1.0) {
        *LU = asin(upper_limit);
    } else if (upper_limit < -1.0) {
        *LU = -M_PI / 2.0;
    }
}

double calculate_phi(double inclination_cosinus, anomaly_elements anomaly_el) {
    return atan2(inclination_cosinus * anomaly_el.anomaly_sinus, anomaly_el.anomaly_cosinus);
}

double calculate_cosinal_coefficient(double inclination_cosinus, anomaly_elements anomaly_el, double ascension_diff) {
    double cosinal_multiplication = anomaly_el.anomaly_cosinus * cos(ascension_diff);
    double sinal_multiplication = anomaly_el.anomaly_sinus * sin(ascension_diff) * inclination_cosinus;
    
    return sinal_multiplication - cosinal_multiplication;
}

double calculate_sinal_coefficient(double inclination_cosinus, double inclination_sinus, anomaly_elements anomaly_el, double ascension_diff) {
    double cosinal_multiplication = anomaly_el.anomaly_cosinus * sin(ascension_diff) * inclination_cosinus;
    double sinal_multiplication = anomaly_el.anomaly_sinus * (pow(inclination_cosinus, 2.0) * cos(ascension_diff) + pow(inclination_sinus, 2.0));
    
    return cosinal_multiplication + sinal_multiplication;
}

void analyze_orbit(orbital_calculations* orbital_calc, int i, int* in_range_ids, orbit_calc* in_range_orbits, int* count, double orbital_ascension, anomaly_elements anomaly_el)
{
    double ascension_calculated = fmod(orbital_calc->ascension_step * (double)i + orbital_calc->min_ascension_angle + 2.0 * M_PI, 2.0 * M_PI);
    if (is_orbit_angle_valid(orbital_calc->min_ascension_angle, orbital_calc->max_ascension_angle, ascension_calculated)) {
        double ascension_diff = orbital_ascension - ascension_calculated;
        int real_id = (int)round((ascension_calculated - orbital_calc->min_ascension_angle) / orbital_calc->ascension_step);
        in_range_ids[*count] = real_id;
        in_range_orbits[*count].cosinal_coefficient = calculate_cosinal_coefficient(orbital_calc->inclination_cosinus, anomaly_el, ascension_diff);
        in_range_orbits[*count].sinal_coefficient = calculate_sinal_coefficient(orbital_calc->inclination_cosinus, orbital_calc->inclination_sinus, anomaly_el, ascension_diff);
        in_range_orbits[*count].ascension_diff = ascension_diff;
        (*count)++;
    }
}

void analyze_orbit_range(orbital_calculations* orbital_calc, range orbit_range, int* in_range_ids, orbit_calc* in_range_orbits, int* count, double orbital_ascension, anomaly_elements anomaly_el) 
{
    for (int i = orbit_range.min; i <= orbit_range.max; i++) {
        analyze_orbit(orbital_calc, i, in_range_ids, in_range_orbits, count, orbital_ascension, anomaly_el);
    }
}

void find_ranges(double ascension_step, int number_of_orbits, double LU, double LD, double Phi, double ascension_from_min, range* first_range, range* second_range) 
{
    if (LD > -M_PI/2.0 && LU < M_PI/2.0) {
        // Both limits are within valid range
        first_range->min = (int)ceil((ascension_from_min + Phi - LU) / ascension_step);
        first_range->max = (int)floor((ascension_from_min + Phi - LD) / ascension_step);
        second_range->min = (int)ceil((ascension_from_min + Phi + LD - M_PI) / ascension_step);
        second_range->max = (int)floor((ascension_from_min + Phi + LU - M_PI) / ascension_step);
    }
    else if (LD <= -M_PI/2.0 && LU < M_PI/2.0 && LU > -M_PI/2.0) {
        // Lower limit is at boundary, upper limit is valid 
        first_range->min = (int)ceil((ascension_from_min + Phi - LU - 2.0*M_PI) / ascension_step);
        first_range->max = (int)floor((ascension_from_min + Phi + LU - M_PI) / ascension_step);
        second_range->min = 0;
        second_range->max = -1;
    }
    else if (LD > -M_PI/2.0 && LD < M_PI/2.0 && LU >= M_PI/2.0) {
        // Lower limit is valid, upper limit is at boundary
        first_range->min = (int)ceil((ascension_from_min + Phi + LD - M_PI) / ascension_step);
        first_range->max = (int)floor((ascension_from_min + Phi - LD) / ascension_step);
        second_range->min = 0;
        second_range->max = -1;
    }
    else if (LU >= M_PI/2.0 && LD <= -M_PI/2.0) {
        // Both limits are at boundaries - include all orbits
        first_range->min = 0;
        first_range->max = number_of_orbits - 1;
        second_range->min = 0;
        second_range->max = -1;
    }
    else {
        // No valid range
        first_range->min = 0;
        first_range->max = -1;
        second_range->min = 0;
        second_range->max = -1;
    }
}

void find_orbits_in_range(orbital_calculations orbital_calc, double length_limit_ratio, anomaly_elements anomaly_el, 
    double orbital_ascension, int* in_range_ids, orbit_calc* in_range_orbits, int* count) {

    double ascension_from_min = orbital_ascension - orbital_calc.min_ascension_angle;
    double LD, LU;
    calculate_limits(orbital_calc.inclination_cosinus, orbital_calc.inclination_sinus, length_limit_ratio, anomaly_el.anomaly_sinus, &LD, &LU);
    double Phi = calculate_phi(orbital_calc.inclination_cosinus, anomaly_el);

    range first_range, second_range;
    find_ranges(orbital_calc.ascension_step, orbital_calc.number_of_orbits, LU, LD, Phi, ascension_from_min, &first_range, &second_range);

    // First range
    if (first_range.min != 0 || first_range.max != -1) {
        analyze_orbit_range(&orbital_calc, first_range, in_range_ids, in_range_orbits, count, orbital_ascension, anomaly_el);
    }

    // Second range
    if (second_range.min != 0 || second_range.max != -1) {
        analyze_orbit_range(&orbital_calc, second_range, in_range_ids, in_range_orbits, count, orbital_ascension, anomaly_el);
    }
}