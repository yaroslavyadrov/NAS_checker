package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/go-telegram-bot-api/telegram-bot-api"
)

// const botToken = "11122:jjhshgg"
// var allowedUsers = map[int]bool{
// 	1111: true,
// }

func main() {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Panic(err)
	}

	for update := range updates {
		log.Printf("chat_id: %d, user_id: %d, username: %s, first_name: %s, last_name: %s, text: %s", update.Message.Chat.ID, update.Message.From.ID, update.Message.From.UserName, update.Message.From.FirstName, update.Message.From.LastName, update.Message.Text)
		if update.Message == nil || !allowedUsers[update.Message.From.ID] {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Access denied")
			bot.Send(msg)
			continue
		}

		if update.Message.Text == "/status" {

			user := update.Message.From
			log.Printf("%d[%s %s] %s", user.ID, user.FirstName, user.LastName, update.Message.Text)

			var outputStr = "```\n"
			devices, err := getDevices()
			if err != nil {
				log.Fatal(err)
			}
			for _, device := range devices {
				outputStr += "DISK USAGE:\n"
				outputStr += fmt.Sprintf("%s:\n", device.Name)
				for _, partition := range device.Partitions {
					outputStr += fmt.Sprintf("%-10s %-5s/ %-5s\n", partition.Name, partition.Used, partition.Total)
				}
				outputStr += "\n"
			}
			smartStatuses, err := getDeviceSmartStatuses(devices)
			if err != nil {
				log.Fatal(err)
			}
			outputStr += "SMART:\n"
			for _, smartStatus := range smartStatuses {
				outputStr += formatSmartStatus(smartStatus) + "\n"
			}
			outputStr += "```"
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, outputStr)
			msg.ParseMode = "MarkdownV2"
			bot.Send(msg)
		}
	}
}

type Device struct {
	Name       string
	Partitions []Partition
}

type Partition struct {
	Name  string
	Used  string
	Total string
}

type DeviceSmartStatus struct {
	Device string
	Status string
	Emoji  string
}

func getDevices() ([]Device, error) {
	output, err := exec.Command("sh", "-c", "df -h | grep /dev/sd").Output()
	if err != nil {
		return nil, err
	}
	var devices []Device
	for _, line := range strings.Split(string(output), "\n") {
		log.Println(line)
		parts := strings.Fields(line)
		if len(parts) < 5 {
			continue // skip incomplete lines
		}
		device := parts[0]
		trimmedDevice := strings.TrimSuffix(device, string(device[len(device)-1]))
		foundDevice := false
		for i := range devices {
			if devices[i].Name == trimmedDevice {
				partition := Partition{
					Name:  parts[0],
					Used:  parts[2],
					Total: parts[1],
				}
				devices[i].Partitions = append(devices[i].Partitions, partition)
				foundDevice = true
				break
			}
		}
		if !foundDevice {
			partition := Partition{
				Name:  parts[0],
				Used:  parts[2],
				Total: parts[1],
			}
			devices = append(devices, Device{
				Name:       trimmedDevice,
				Partitions: []Partition{partition},
			})
		}
		log.Printf("%-12s %-5s/%-5s", device, parts[2], parts[1])
	}
	return devices, nil
}

func getDeviceSmartStatuses(devices []Device) ([]DeviceSmartStatus, error) {
	var statuses []DeviceSmartStatus
	for _, device := range devices {
		output, err := exec.Command("sudo", "smartctl", "-H", device.Name).Output()
		if err != nil {
			return nil, err
		}
		smartStatus := strings.TrimPrefix(strings.Split(string(output), "\n")[4], "SMART overall-health self-assessment test result: ")
		var emoji = "🟢"
		if smartStatus != "PASSED" {
			emoji = "🔴"
		}
		statuses = append(statuses, DeviceSmartStatus{device.Name, smartStatus, emoji})
	}
	return statuses, nil
}

func formatSmartStatus(smartStatus DeviceSmartStatus) string {
	return fmt.Sprintf("%-10s %s %s", smartStatus.Device, smartStatus.Status, smartStatus.Emoji)
}
