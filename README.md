# Chameleon Enclclosure Monitoring System (CEMS)

For the past several years, I have been raising a veiled chameleon named Jeffrina. She has a lovely mesh enclosure and currently resides in a nice schefflera plant to prove hiding and hunting places. The enclosure is fairly simple but I'll break it down here:

- daylight lamp (provides vitamin D to keep her healthy)
- heat lamp (provides a place to bask and regulater her body temp)
- automated misting system

## Project Overview

Each of the core components is currently timer driven. I'd like to change that to make a more natural environment. I have three main goals:

- Control daylight lamp based on the actual lunar cycle
- Control heat lamp based on the actual enclosure temp
- Control misting duration and frequency based on actual humidity
- Monitor water level in the holding tank for the misting system

Of these requirements, monitoring the water level is the most critical at the moment. My large controller died and I was only able to source a similar system with a much smaller tank. I have to fill it constantly and it's tucked away in the cabinet under the enclosure and not entirely easy to access. I frequently run the system dry and I want to prevent that. Both for her sake and mine.

## Tech Stack

|Technology|Description|More Info
|--|--|
|Go 1.15|Programming langugage|[Main Site](https://golang.org)
|Raspberry Pi 3|Computer Platform|[Main Site](https://www.raspberrypi.org)
|DHT22|Temp/Humidity Sensor|[Adafruit](https://learn.adafruit.com/dht)
|HC-SR04|Ultrasonic Sensor|[Sparkfun](https://www.sparkfun.com/products/15569)
|Prometheus|Metrics Platform|[https://prometheus.io](http://prometheus.io)
|Grafana|Metrics Visualizatioin|[Pi Installation](https://grafana.com/tutorials/install-grafana-on-raspberry-pi/#3])
