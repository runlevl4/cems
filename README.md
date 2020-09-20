# Chameleon Enclclosure Monitoring System (CEMS)

For the past several years, I have been raising a veiled chameleon named 
Jeffrina. She has a lovely mesh enclosure and currently resides in a nice 
schefflera plant to prove hiding and hunting places. The enclosure is fairly 
simple but I'll break it down here:

- daylight lamp (provides vitamin D to keep her healthy)
- heat lamp (provides a place to bask and regulater her body temp)
- automated misting system

## Project Overview

Each of the core components is currently timer driven. I'd like to change that 
to make a more natural environment. I have three main goals:

- Control daylight lamp based on the actual lunar cycle
- Control heat lamp based on the actual enclosure temp
- Control misting duration and frequency based on actual humidity
- Monitor water level in the holding tank for the misting system

Of these requirements, monitoring the water level is the most critical at the 
moment. My large controller died and I was only able to source a similar 
system with a much smaller tank. I have to fill it constantly and it's tucked 
away in the cabinet under the enclosure and not entirely easy to access. I 
frequently run the system dry and I want to prevent that. Both for her sake and mine.

## Tech Stack

|Technology|Description|More Info|
|-----------|-----------|-----
|Go 1.15|Programming langugage|[Main Site](https://golang.org)
|Raspberry Pi 3|Computer Platform|[Main Site](https://www.raspberrypi.org)
|DHT22|Temp/Humidity Sensor|[Adafruit](https://learn.adafruit.com/dht)
|HC-SR04|Ultrasonic Sensor|[Sparkfun](https://www.sparkfun.com/products/15569)
|Prometheus|Metrics Platform|[Website](http://prometheus.io)
|Grafana|Metrics Visualizatioin|[Pi Installation](https://grafana.com/tutorials/install-grafana-on-raspberry-pi/#3])

## General Design

CEMS is comprised of two parts. The first part is a command-line interface (CLI) 
which runs in the context of a goroutine to constantly query the DHT22 sensor.
This ensures that the custom Prometheus metrics for temp and humidity are stored
in the time-series database for historical trending.

The second part is an API which executes the same query logic via a HTTP endpoint.
Currently, the only API endpoint is `stats` which will return a JSON representation
of the data elements.