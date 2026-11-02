package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	log.Println("starting device simulator with 100 devices, 10 users")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			runDevice(i)
		}(i)
	}
	wg.Wait()
}

func mqttBroker() string {
	if v := os.Getenv("MQTT_BROKER"); v != "" {
		return v
	}
	return "tcp://localhost:1883"
}

func runDevice(i int) {
	userID := fmt.Sprintf("user-%d", i%10)
	deviceID := fmt.Sprintf("device-%d", i)
	broker := mqttBroker()

	for {
		client := connect(deviceID, userID, broker)
		if client == nil {
			time.Sleep(5 * time.Second)
			continue
		}

		// Subscribe to notifications and badge topics.
		notifTopic := fmt.Sprintf("users/%s/notifications", userID)
		badgeTopic := fmt.Sprintf("users/%s/badge", userID)

		client.Subscribe(notifTopic, 1, func(_ mqtt.Client, m mqtt.Message) {
			log.Printf("[%s] received notification on %s: %s", deviceID, m.Topic(), string(m.Payload()))
		})
		client.Subscribe(badgeTopic, 1, func(_ mqtt.Client, m mqtt.Message) {
			log.Printf("[%s] badge update on %s: %s", deviceID, m.Topic(), string(m.Payload()))
		})

		// Randomly disconnect and reconnect to simulate device churn.
		//nolint:gosec
		onlineDuration := time.Duration(10+rand.Intn(50)) * time.Second
		time.Sleep(onlineDuration)

		client.Disconnect(250)
		log.Printf("[%s] disconnected", deviceID)

		//nolint:gosec
		offlineDuration := time.Duration(5+rand.Intn(10)) * time.Second
		time.Sleep(offlineDuration)

		log.Printf("[%s] reconnecting...", deviceID)
	}
}

func connect(deviceID, userID, broker string) mqtt.Client {
	opts := mqtt.NewClientOptions().
		SetClientID(deviceID).
		SetCleanSession(false).
		AddBroker(broker).
		SetAutoReconnect(false).
		SetWill(
			fmt.Sprintf("devices/%s/status", deviceID),
			"offline",
			1,
			true,
		).
		SetOnConnectHandler(func(c mqtt.Client) {
			log.Printf("[%s] connected (user=%s)", deviceID, userID)
			// Publish online status.
			statusTopic := fmt.Sprintf("devices/%s/status", deviceID)
			c.Publish(statusTopic, 1, true, "online")
		}).
		SetConnectionLostHandler(func(_ mqtt.Client, err error) {
			log.Printf("[%s] connection lost: %v", deviceID, err)
		})

	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()
	if err := token.Error(); err != nil {
		log.Printf("[%s] connect error: %v", deviceID, err)
		return nil
	}
	return client
}
