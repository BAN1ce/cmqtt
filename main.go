package main

import (
	"flag"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"log"
	"os"
	"sync"
	"time"
)

var (
	ip          = flag.String("ip", "127.0.0.1:1883", "server IP")
	connections = flag.Int("conn", 10, "number of tcp connections")
	per         = flag.Int("per", 2, "number of messages count per connection")
	name        = flag.String("name", "local", "localName")
)

var f mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Println("Miss Sub Topic Message", msg.Topic(), msg.Payload())
}

func main() {

	flag.Parse()

	//mqtt.DEBUG = log.New(os.Stdout, "", 0)
	mqtt.ERROR = log.New(os.Stdout, "", 0)

	conns := make([]mqtt.Client, *connections)

	var mu sync.RWMutex

	statistic := make(map[string]int)

	fmt.Println(*ip)

	addr := fmt.Sprintf("tcp://%s", *ip)

	for i := 0; i < len(conns); i++ {

		opts := mqtt.NewClientOptions().AddBroker(addr).SetUsername(fmt.Sprintf("%s_hello", *name) + string(i)).SetClientID(fmt.Sprintf("%s111%d", *name, i))

		opts.SetKeepAlive(60 * time.Second)
		opts.SetPingTimeout(10 * time.Second)
		opts.SetConnectTimeout(5 * time.Second)

		// 设置消息回调处理函数
		opts.SetDefaultPublishHandler(f)

		c := mqtt.NewClient(opts)
		if token := c.Connect(); token.Wait() && token.Error() != nil {
			i--
			fmt.Println(token.Error())
			continue
		}

		// 订阅主题
		topic := fmt.Sprintf(*name+"product/%d", len(conns)-i-1)
		if token := c.Subscribe(topic, 1, func(client mqtt.Client, message mqtt.Message) {
			mu.Lock()
			statistic[message.Topic()] += 1
			mu.Unlock()
		}); token.Wait() && token.Error() != nil {
			fmt.Println(token.Error())
			i--
		}
		conns[i] = c

	}
	fmt.Println(fmt.Sprintf("create %d clients success", *connections))

	j := 0
	for {
		for i, c := range conns {
			// 发布消息
			timeStr := time.Now().Format("2006-01-02 15:04:05")
			token := c.Publish(fmt.Sprintf(*name+"product/%d", i), 0, false, fmt.Sprintf("Hello World %d - %s", i, timeStr))
			token.Wait()
		}
		fmt.Println(j, "times send success")
		time.Sleep(2 * time.Second)
		j++

		if j == *per {
			break
		}
	}

	fmt.Println("All message published")

	time.Sleep(5 * time.Second)

	t := time.NewTicker(120 * time.Second)

	sendTimer := time.NewTicker(60 * time.Second)

	for {
		select {

		case <-t.C:
			for k, v := range statistic {
				if v != *per {
					fmt.Println(k, v, "miss receiver")
				}
			}

			fmt.Println("all scan")

		case <-sendTimer.C:

			go func() {
				timeStr := time.Now().Format("2006-01-02 15:04:05")
				for i, c := range conns {
					// 发布消息
					token := c.Publish(fmt.Sprintf(*name+"product/%d", i), 0, false, fmt.Sprintf("Hello World %d - %s", i, timeStr))
					token.Wait()
				}
			}()
		}
	}

}
