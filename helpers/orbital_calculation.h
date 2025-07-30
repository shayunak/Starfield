#ifndef ORBITAL_CALCULATION_H
#define ORBITAL_CALCULATION_H

#ifndef M_PI
#define M_PI 3.14159265358979323846
#endif

typedef struct {
    double anomaly_sinus;
    double anomaly_cosinus;
} anomaly_elements;

typedef struct {
    double cosinal_coefficient;
    double sinal_coefficient;
    double ascension_diff;
} orbit_calc;

typedef struct {
    double inclination_sinus;
    double inclination_cosinus;
    int number_of_orbits;
    double ascension_step;
    double min_ascension_angle;
    double max_ascension_angle;
} orbital_calculations;

typedef struct {
    int min;
    int max;
} range;

void find_orbits_in_range(
    orbital_calculations orbital_calc,
    double length_limit_ratio,
    anomaly_elements anomaly_el,
    double orbital_ascension,
    int* in_range_ids,
    orbit_calc* in_range_orbits,
    int* count
);

double calculate_cosinal_coefficient(
    double inclination_cosinus,
    anomaly_elements anomaly_el, 
    double ascension_diff
);

double calculate_sinal_coefficient(
    double inclination_cosinus,
    double inclination_sinus,
    anomaly_elements anomaly_el,
    double ascension_diff
);

#endif