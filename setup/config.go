package setup

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

type OrbitConfig struct {
	EarthRadius                float64 `json:"earth_radius"`          // in meters
	Altitude                   float64 `json:"altitude"`              // in meters
	EarthRotationPeriod        float64 `json:"earth_rotation_period"` // in number of revolutions per day
	MinAltitudeISL             float64 `json:"min_altitude_isl"`      // in meters (for weather conditions)
	Inclination                float64 `json:"inclination"`           // in meters
	MinAscensionAngle          float64 `json:"min_ascension_angle"`   // in degrees
	MaxAscensionAngle          float64 `json:"max_ascension_angle"`   // in meters
	NumberOfOrbits             int     `json:"number_of_orbits"`
	NumberOfSatellitesPerOrbit int     `json:"number_of_satellites_per_orbit"`
	PhaseDiffEnabled           bool    `json:"phase_diff_enabled"` // gives a half-cycle phase difference to odd number orbits
}

type SatelliteConfig struct {
	MeanMotionRevPerDay float64 `json:"mean_motion_rev_per_day"` // in number of revolutions per day
	ConeRadius          float64 `json:"cone_radius"`             // in meters
	MinElevationAngle   float64 `json:"min_elevation_angle"`     // in degrees
	NumberOfISLs        int     `json:"number_of_isls"`
	ISLBandwidth        float64 `json:"isl_bandwidth"`         // in Mbps
	ISLLinkNoiseCoef    float64 `json:"isl_link_noise_coef"`   // in km^2
	ISLAcquisitionTime  float64 `json:"isl_acquisition_time"`  // in seconds
	GSLBandwidth        float64 `json:"gsl_bandwidth"`         // in Mbps
	GSLLinkNoiseCoef    float64 `json:"gsl_link_noise_coef"`   // in km^2
	SpeedOfLightVac     float64 `json:"speed_of_light_vac"`    // in meters per second
	MaxPacketSize       float64 `json:"max_packet_size"`       // in Kb
	InterfaceBufferSize int     `json:"interface_buffer_size"` // number of Packets
}

type Config struct {
	ConsellationName string          `json:"name"`
	OrbitConfig      OrbitConfig     `json:"orbit_config"`
	SatelliteConfig  SatelliteConfig `json:"satellite_config"`
}

type IConfig interface {
	toString() string
}

func (config Config) toString() string {
	return fmt.Sprintf("{ \n orbit_config: %s, \n satellite_config: %s \n}",
		config.OrbitConfig.toString(), config.SatelliteConfig.toString())
}

func (orbitConfig OrbitConfig) toString() string {
	return fmt.Sprintf("{ \n earth_radius: %v, \n altitude: %v, \n inclination: %v, \n min_ascension_angle: %v, \n"+
		"max_ascension_angle: %v, \n number_of_orbits: %v, \n number_of_sattelites_per_orbit: %v, \n phase_diff_enabled: %v\n}",
		orbitConfig.EarthRadius, orbitConfig.Altitude, orbitConfig.Inclination, orbitConfig.MinAscensionAngle,
		orbitConfig.MaxAscensionAngle, orbitConfig.NumberOfOrbits, orbitConfig.NumberOfSatellitesPerOrbit, orbitConfig.PhaseDiffEnabled)
}

func (satelliteConfig SatelliteConfig) toString() string {
	return fmt.Sprintf("{ \n mean_motion_rev_per_day: %v, \n cone_radius: %v \n min_elevation_angle: %v, \n number_of_isls: %v, \n"+
		"isl_bandwidth: %v, \n isl_link_noise_coef: %v, \n isl_acquisition_time: %v, \n gsl_bandwidth: %v, \n gsl_link_noise_coef: %v, \n"+
		"speed_of_light_vac: %v, \n max_packet_size: %v, \n interface_buffer_size: %v \n}",
		satelliteConfig.MeanMotionRevPerDay, satelliteConfig.ConeRadius, satelliteConfig.MinElevationAngle, satelliteConfig.NumberOfISLs,
		satelliteConfig.ISLBandwidth, satelliteConfig.ISLLinkNoiseCoef, satelliteConfig.ISLAcquisitionTime, satelliteConfig.GSLBandwidth,
		satelliteConfig.GSLLinkNoiseCoef, satelliteConfig.SpeedOfLightVac, satelliteConfig.MaxPacketSize, satelliteConfig.InterfaceBufferSize)
}

/*
	In order to let your config file be applied to the simulator,

you need to put your config file at the "configs" folder,

	and pass its name in the program's arguments.
*/
func getConfig(configFileName string) Config {
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
