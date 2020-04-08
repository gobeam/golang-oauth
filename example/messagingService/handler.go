package messagingService

import (
	"fmt"
	"github.com/gobeam/golang-oauth/example/core/models"
	"github.com/gobeam/golang-oauth/example/common/messaging"
	"github.com/streadway/amqp"
	"strconv"
)

var MessagingClient messaging.AmpqMessagingClient

//func notifyVIP(account string) {
//		go func(account string) {
//			vipNotification := fmt.Sprintf("Notification: %s", account)
//			data, _ := json.Marshal(vipNotification)
//			fmt.Printf("Notifying VIP account %v\n", account)
//			err := MessagingClient.PublishOnQueue(data, "vip_queue")
//			if err != nil {
//				fmt.Println(err.Error())
//			}
//		}(account)
//}

const (
	AsciiImageUpdateRoute = "app.service.auth.banner.update"
)

func HandleEvents(d amqp.Delivery) {
	switch d.RoutingKey {
	case AsciiImageUpdateRoute:
		banner := &models.Banner{}
		bannerId, _ := strconv.ParseUint(d.CorrelationId, 10, 32)
		banner.ID = uint(bannerId)
		banner.FindById()
		//fmt.Println("==========hhhhhhhh========")
		//fmt.Println(string(d.Body))
		//fmt.Println("==========hhhhhhhh========")
		banner.AsciiImage = string(d.Body)
		banner.Update()
	}

	err := d.Ack(false)
	if err != nil {
		fmt.Println("Error: ", err)
	}
}