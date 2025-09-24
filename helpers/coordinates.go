package helpers

import "math"

type CartesianCoordinates struct {
	X float64 // in meters
	Y float64 // in meters
	Z float64 // in meters
}

type KepplerianCoordinates struct {
	Anomaly     float64 // in radians
	Radius      float64 // in meters
	Ascension   float64 // in radians
	Inclination float64 // in radians
}

type SphericalCoordinates struct {
	Radius    float64 // in meters
	Latitude  float64 // in radians
	Longitude float64 // in radians
}

func ConvertToCartesian(coordinates KepplerianCoordinates) CartesianCoordinates {
	return CartesianCoordinates{
		X: coordinates.Radius * (math.Cos(coordinates.Anomaly)*math.Cos(coordinates.Ascension) -
			math.Sin(coordinates.Anomaly)*math.Cos(coordinates.Inclination)*math.Sin(coordinates.Ascension)),
		Y: coordinates.Radius * (math.Cos(coordinates.Anomaly)*math.Sin(coordinates.Ascension) +
			math.Sin(coordinates.Anomaly)*math.Cos(coordinates.Inclination)*math.Cos(coordinates.Ascension)),
		Z: coordinates.Radius * math.Sin(coordinates.Anomaly) * math.Sin(coordinates.Inclination),
	}
}

func ConvertToSpherical(coordinates CartesianCoordinates) SphericalCoordinates {
	return SphericalCoordinates{
		Radius:    math.Sqrt(coordinates.X*coordinates.X + coordinates.Y*coordinates.Y + coordinates.Z*coordinates.Z),
		Latitude:  math.Atan2(coordinates.Z, math.Sqrt(coordinates.X*coordinates.X+coordinates.Y*coordinates.Y)),
		Longitude: math.Atan2(coordinates.Y, coordinates.X),
	}
}
