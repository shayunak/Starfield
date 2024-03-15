package setup

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

type OrbitConfig struct {
	EarthRadius                float64 `json:"earth_radius"`        // in meters
	Altitude                   float64 `json:"altitude"`            // in meters
	Inclination                float64 `json:"inclination"`         // in meters
	MinAscensionAngle          float64 `json:"min_ascension_angle"` // in degrees
	MaxAscensionAngle          float64 `json:"max_ascension_angle"` // in meters
	NumberOfOrbits             int     `json:"number_of_orbits"`
	NumberOfSattelitesPerOrbit int     `json:"number_of_sattelites_per_orbit"`
	PhaseDiffEnabled           bool    `json:"phase_diff_enabled"` // gives a half-cycle phase difference to odd number orbits
}

type SatelliteConfig struct {
	MeanMotionRevPerDay float64 `json:"mean_motion_rev_per_day"` // in number of revolutions per day
	ConeRadius          float64 `json:"cone_radius"`             // in meters
	MaxIslLength        float64 `json:"max_isl_length"`          // in meters
}

type Config struct {
	OrbitConfig     OrbitConfig     `json:"orbit_config"`
	SatelliteConfig SatelliteConfig `json:"satellite_config"`
}

type IConfig interface {
	toString() string
}

func (config Config) ToString() string {
	return fmt.Sprintf("{ \n orbit_config: %s, \n satellite_config: %s \n}",
		config.OrbitConfig.toString(), config.SatelliteConfig.toString())
}

func (orbitConfig OrbitConfig) toString() string {
	return fmt.Sprintf("{ \n earth_radius: %v, \n altitude: %v, \n inclination: %v, \n min_ascension_angle: %v, \n"+
		"max_ascension_angle: %v, \n number_of_orbits: %v, \n number_of_sattelites_per_orbit: %v, \n phase_diff_enabled: %v\n}",
		orbitConfig.EarthRadius, orbitConfig.Altitude, orbitConfig.Inclination, orbitConfig.MinAscensionAngle,
		orbitConfig.MaxAscensionAngle, orbitConfig.NumberOfOrbits, orbitConfig.NumberOfSattelitesPerOrbit, orbitConfig.PhaseDiffEnabled)
}

func (satelliteConfig SatelliteConfig) toString() string {
	return fmt.Sprintf("{ \n mean_motion_rev_per_day: %v, \n cone_radius: %v, \n max_isl_length: %v \n}",
		satelliteConfig.MeanMotionRevPerDay, satelliteConfig.ConeRadius, satelliteConfig.MaxIslLength)
}

/*
	In order to let your config file be applied to the simulator,

you need to put your config file at the "configs" folder,

	and pass its name in the program's arguments.
*/
func GetConfig(configFileName string) Config {
	configFilePath := "./configs/" + configFileName
	config_file, err := os.Open(configFilePath)
	if err != nil {
		log.Fatal(err)
	}

	config_bytes, err := io.ReadAll(config_file)
	if err != nil {
		log.Fatal(err)
	}

	var config Config
	json.Unmarshal(config_bytes, &config)

	config_file.Close()
	return config
}
