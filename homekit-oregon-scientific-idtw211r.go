package main

import (
	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/service"

	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

var sensorHost string
var sensorPort int
var secondsBetweenReadings time.Duration
var developmentMode bool

func init() {
	flag.StringVar(&sensorHost, "host", "http://0.0.0.0", "sensor host, a string")
	flag.IntVar(&sensorPort, "port", 1006, "sensor port number, an int")
	flag.DurationVar(&secondsBetweenReadings, "sleep", 5*time.Second, "how many seconds between sensor readings, an int followed by the duration")
	flag.BoolVar(&developmentMode, "dev", false, "turn on development mode to return a random temperature reading, boolean")
	flag.Parse()

	if developmentMode == true {
		log.Println("Development mode on, ignoring sensor and returning random values...")
	}
}

func main() {
	bridge := accessory.NewBridge(
		accessory.Info{
			Name:             "Thermometer",
			SerialNumber:     "INDOOR-OUTDOOR",
			Manufacturer:     "Oregon Scientific",
			Model:            "IDTW211R",
			FirmwareRevision: "1.0.0",
			ID:               1,
		},
	)

	battery := service.NewBatteryService()
	batteryLevel := characteristic.NewBatteryLevel()
	battery.Service.AddCharacteristic(batteryLevel.Characteristic)
	bridge.AddService(battery.Service)

	indoor := accessory.NewTemperatureSensor(
		accessory.Info{
			Name:             "Indoor",
			SerialNumber:     "INDOOR-OUTDOOR",
			Manufacturer:     "Oregon Scientific",
			Model:            "IDTW211R",
			FirmwareRevision: "1.0.0",
			ID:               2,
		},
		0.0,  // Initial value
		-5.0, // Min sensor value
		50.0, // Max sensor value
		0.1,  // Step value
	)

	outdoor := accessory.NewTemperatureSensor(
		accessory.Info{
			Name:             "Outdoor",
			SerialNumber:     "INDOOR-OUTDOOR",
			Manufacturer:     "Oregon Scientific",
			Model:            "IDTW211R",
			FirmwareRevision: "1.0.0",
			ID:               3,
		},
		0.0,   // Initial value
		-20.0, // Min sensor value
		60.0,  // Max sensor value
		0.1,   // Step value
	)

	config := hc.Config{
		// Change the default Apple Accessory Pin if you wish
		Pin: "00102003",
		// Port: "12345",
		// StoragePath: "./db",
	}

	t, err := hc.NewIPTransport(
		config,
		bridge.Accessory,
		indoor.Accessory,
		outdoor.Accessory,
	)
	if err != nil {
		log.Fatal(err)
	}

	// Get the sensor readings every secondsBetweenReadings
	go func() {
		readings := []string{
			"temperature_indoors",
			"temperature_outdoors",
		}
		for {
			// Get readings from the Prometheus exporter
			indoorReading := 0.0
			outdoorReading := 0.0
			batteryPercentage := 0.0
			resp, err := http.Get(fmt.Sprintf("%s:%d", sensorHost, sensorPort))
			if err == nil {
				defer resp.Body.Close()
				scanner := bufio.NewScanner(resp.Body)
				for scanner.Scan() {
					line := scanner.Text()
					// Parse the temperature readings
					for _, reading := range readings {
						regexString := fmt.Sprintf("^%s", reading) + ` ([-+]?\d*\.\d+|\d+)`
						re := regexp.MustCompile(regexString)
						rs := re.FindStringSubmatch(line)
						if rs != nil {
							parsedValue, err := strconv.ParseFloat(rs[1], 64)
							if err == nil {
								if reading == "temperature_indoors" {
									indoorReading = parsedValue
								} else if reading == "temperature_outdoors"{
									outdoorReading = parsedValue
								} else if reading == "battery_percentage" {
									batteryPercentage = parsedValue
								}
							}
						}
					}
				}
			} else {
				log.Println(err)
			}

			if developmentMode == true {
				// Return a random float between 15 and 30
				indoorReading = 15 + rand.Float64()*(30-15)
				outdoorReading = 15 + rand.Float64()*(30-15)
			}

			// Set the temperature reading on the accessory
			indoor.TempSensor.CurrentTemperature.SetValue(indoorReading)
			outdoor.TempSensor.CurrentTemperature.SetValue(outdoorReading)
			batteryLevel.SetValue(int(batteryPercentage))
			log.Println(fmt.Sprintf("Indoors: %f°C, outdoors: %f°C, battery: %f%", indoorReading, outdoorReading, batteryPercentage))

			// Time between readings
			time.Sleep(secondsBetweenReadings)
		}
	}()

	hc.OnTermination(func() {
		<-t.Stop()
	})

	t.Start()
}
