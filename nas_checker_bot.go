package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api"
)

// const BotToken = "11122:jjhshgg"
// var AllowedUsers = map[int]bool{
// 	1111: true,
// }
// var ChatsToSignal = []int{
// 	122112,
// }

func main() {
	bot, err := tgbotapi.NewBotAPI(BotToken)
	if err != nil {
		log.Panic(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	backgroundSmartCheck(bot)
	backgroundServicesCheck(bot)

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Panic(err)
	}

	for update := range updates {
		if update.Message == nil || !AllowedUsers[update.Message.From.ID] {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Access denied")
			bot.Send(msg)
			continue
		}

		if update.Message.Text == "/status" {

			user := update.Message.From
			log.Printf("%d[%s %s] %s", user.ID, user.FirstName, user.LastName, update.Message.Text)

			var outputStr = ""
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
			}
			smartStatuses, err := getDeviceSmartStatuses(devices)
			if err != nil {
				log.Fatal(err)
			}
			outputStr += "\nSMART:\n"
			for _, smartStatus := range smartStatuses {
				outputStr += formatSmartStatus(smartStatus) + "\n"
			}
			formattedServicesStatuses, err := getFormattedServicesStatuses(ServicesToCheck)
			if err != nil {
				log.Fatal(err)
			}
			outputStr += "\n" + formattedServicesStatuses
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, outputStr)
			sendFormattedMsg(bot, &msg)
		}
		if update.Message.Text == "/services" {
			formattedServicesStatuses, err := getFormattedServicesStatuses(ServicesToCheck)
			if err != nil {
				log.Fatal(err)
			}
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, formattedServicesStatuses)
			sendFormattedMsg(bot, &msg)
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

func backgroundSmartCheck(bot *tgbotapi.BotAPI) {
	go func() {
		for {
			devices, err := getDevices()
			if err != nil {
				log.Fatal(err)
			}
			smartStatuses, err := getDeviceSmartStatuses(devices)
			if err != nil {
				log.Fatal(err)
			}
			for _, smartStatus := range smartStatuses {
				if smartStatus.Status != "PASSED" {
					msg := fmt.Sprintf("Device %s has SMART status %s %s", smartStatus.Device, smartStatus.Status, smartStatus.Emoji)
					log.Printf("Sending message: %s", msg)
					for _, chatID := range ChatsToSignal {
						msg := tgbotapi.NewMessage(chatID, msg)
						sendFormattedMsg(bot, &msg)
					}
				}
			}
			time.Sleep(3 * time.Hour)
		}
	}()
}

func backgroundServicesCheck(bot *tgbotapi.BotAPI) {
	go func() {
		for {
			for _, service := range ServicesToCheck {
				status, err := getServiceStatus(service)
				if err != nil {
					log.Fatal(err)
				}
				if status != "active" {
					msg := fmt.Sprintf("Service %s has status %s", service, status)
					log.Printf("Sending message: %s", msg)
					for _, chatID := range ChatsToSignal {
						msg := tgbotapi.NewMessage(chatID, msg)
						sendFormattedMsg(bot, &msg)
					}
				}
			}
			time.Sleep(1 * time.Hour)
		}
	}()
}

func getServiceStatus(service string) (string, error) {
	output, err := exec.Command("sh", "-c", "systemctl is-active \""+service+"\"").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func getFormattedServicesStatuses(services []string) (string, error) {
	var outputStr = "SERVICES:\n"
	for _, service := range ServicesToCheck {
		status, err := getServiceStatus(service)
		if err != nil {
			return "", err
		}
		if status == "active" {
			status = "✅ " + status
		} else {
			status = "❌ " + status
		}
		outputStr += fmt.Sprintf("%-10s %-10s\n", service, status)
	}
	return outputStr, nil
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
	return fmt.Sprintf("%-10s %s %s", smartStatus.Device, smartStatus.Emoji, smartStatus.Status)
}

func sendFormattedMsg(bot *tgbotapi.BotAPI, message *tgbotapi.MessageConfig) {
	message.Text = fmt.Sprintf("```\n%s\n```", message.Text)
	message.ParseMode = "MarkdownV2"
	_, err := bot.Send(message)
	if err != nil {
		log.Printf("Error sending message: %v", err)
	}
}
