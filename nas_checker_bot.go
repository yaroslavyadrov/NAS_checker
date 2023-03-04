package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api"
)

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

	var keyboard = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("/status"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("/storage"),
			tgbotapi.NewKeyboardButton("/smart"),
			tgbotapi.NewKeyboardButton("/services"),
			tgbotapi.NewKeyboardButton("/report"),
		),
	)

	for update := range updates {
		if update.Message == nil || !AllowedUsers[update.Message.From.ID] {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Access denied")
			bot.Send(msg)
			continue
		}

		if update.Message.Text == "/menuon" {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ok, "+update.Message.From.FirstName)
			msg.ReplyMarkup = keyboard
			bot.Send(msg)
		}

		if update.Message.Text == "/menuoff" {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ok, "+update.Message.From.FirstName)
			msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
			bot.Send(msg)
		}

		if update.Message.Text == "/status" {
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
			sendFormattedMsg(bot, &msg, true)
		}
		if update.Message.Text == "/services" {
			formattedServicesStatuses, err := getFormattedServicesStatuses(ServicesToCheck)
			if err != nil {
				log.Fatal(err)
			}
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, formattedServicesStatuses)
			sendFormattedMsg(bot, &msg, true)
		}
		if update.Message.Text == "/storage" {
			devices, err := getDevices()
			if err != nil {
				log.Fatal(err)
			}
			var outputStr = "DISK USAGE:\n"
			for _, device := range devices {
				outputStr += fmt.Sprintf("%s:\n", device.Name)
				for _, partition := range device.Partitions {
					outputStr += fmt.Sprintf("%-10s %-5s/ %-5s\n", partition.Name, partition.Used, partition.Total)
				}
			}
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, outputStr)
			sendFormattedMsg(bot, &msg, true)
		}
		if update.Message.Text == "/smart" {
			devices, err := getDevices()
			if err != nil {
				log.Fatal(err)
			}
			smartStatuses, err := getDeviceSmartStatuses(devices)
			if err != nil {
				log.Fatal(err)
			}
			var outputStr = "SMART:\n"
			for _, smartStatus := range smartStatuses {
				outputStr += formatSmartStatus(smartStatus) + "\n"
			}
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, outputStr)
			sendFormattedMsg(bot, &msg, true)
		}
		if update.Message.Text == "/report" {
			sendDevicesFullReportAsFiles(bot, update.Message.Chat.ID)
		}
		if update.Message.Text == "/reboot" {
			exec.Command("sudo", "reboot")
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
					msg := fmt.Sprintf("Device __**%s**__ has SMART status __**%s**__ %s", smartStatus.Device, smartStatus.Status, smartStatus.Emoji)
					log.Printf("Sending message: %s", msg)
					for _, chatID := range ChatsToSignal {
						msg := tgbotapi.NewMessage(chatID, msg)
						sendFormattedMsg(bot, &msg, false)
					}
				}
			}
			time.Sleep(SMARTCheckInterval)
		}
	}()
}

func backgroundServicesCheck(bot *tgbotapi.BotAPI) {
	go func() {
		for {
			for _, service := range ServicesToCheck {
				status := getServiceStatus(service)
				if status != "active" {
					msg := fmt.Sprintf("Service __**%s**__ has status __**%s**__", service, status)
					log.Printf("Sending message: %s", msg)
					for _, chatID := range ChatsToSignal {
						msg := tgbotapi.NewMessage(chatID, msg)
						sendFormattedMsg(bot, &msg, false)
					}
				}
			}
			time.Sleep(ServicesCheckInterval)
		}
	}()
}

func getServiceStatus(service string) string {
	output, _ := exec.Command("sh", "-c", "systemctl is-active \""+service+"\"").Output()
	return strings.TrimSpace(string(output))
}

func getFormattedServicesStatuses(services []string) (string, error) {
	var outputStr = "SERVICES:\n"
	for _, service := range ServicesToCheck {
		status := getServiceStatus(service)
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
	}
	// Sort partitions by name
	for i := range devices {
		sort.Slice(devices[i].Partitions, func(j, k int) bool {
			return devices[i].Partitions[j].Name < devices[i].Partitions[k].Name
		})
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

func sendDevicesFullReportAsFiles(bot *tgbotapi.BotAPI, chatId int64) {
	devices, err := getDevices()
	if err != nil {
		log.Fatal(err)
	}
	for _, device := range devices {
		output, err := exec.Command("sudo", "smartctl", "-a", device.Name).Output()
		if err != nil {
			log.Fatal(err)
		}
		//write to tmp file with device name as current date in format 2021-01-01Thh:mm:ss
		deviceName := strings.TrimPrefix(device.Name, "/dev/")
		filename := "/tmp/" + deviceName + "-" + time.Now().Format("2006-01-02") + ".txt"
		err = os.WriteFile(filename, output, 0644)
		if err != nil {
			log.Fatal(err)
		}
		//send file to telegram
		file, err := os.Open(filename)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		msg := tgbotapi.NewDocumentUpload(chatId, tgbotapi.FileBytes{Name: device.Name + "-" + time.Now().Format("2006-01-02") + ".txt", Bytes: output})
		_, err = bot.Send(msg)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func sendFormattedMsg(bot *tgbotapi.BotAPI, message *tgbotapi.MessageConfig, code bool) {
	if code {
		message.Text = fmt.Sprintf("```\n%s\n```", message.Text)
	}
	message.ParseMode = "MarkdownV2"
	_, err := bot.Send(message)
	if err != nil {
		log.Printf("Error sending message: %v", err)
	}
}
