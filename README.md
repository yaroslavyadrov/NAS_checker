# NAS_checker

A simple Telegram bot to monitor NAS.

Bot also send nottifications if services are down or SMART status is not PASSED.

<img width="531" alt="sample" src="https://user-images.githubusercontent.com/10099316/221376713-6ae66bef-969d-4f6c-9f48-cf2cf12058bf.png">


## Commands list

- `/status` - Full report of the system status
- `/services` - Status of the services
- `/storage` - Disk usage
- `/smart` - SMART status
- `/report` - SMART report
- `/reboot` - Reboot the system

## Usage

Create a bot using [@BotFather](https://t.me/BotFather) and get the token.

Create file `config.go` in the same directory with `nas_cheker_bot.go` with the following content:

```go
package main

import "time"

const BotToken = "your_bot_token"

var AllowedUsers = map[int]bool{
   123123: true,
}

var ChatsToSignal = []int64{
   123123,
}

//Any services you want to check
var ServicesToCheck = []string{
   "smbd",
   "dnsmasq",
   "sshd",
}

var SMARTCheckInterval = 3 * time.Hour

var ServicesCheckInterval = 1 * time.Hour
```

Build bot with `make build`

Pass `./build/nas_checker_bot` file to your NAS.

To run a generated binary as a service on Raspberry Pi, you can create a systemd unit file. Here are the steps:

Create a new file with the .service extension in the `/etc/systemd/system/` directory. For example, `/etc/systemd/system/nas_checker_bot.service`

Edit the file and add the following configuration:

```makefile
[Unit]
Description=NAS Checker Bot

[Service]
ExecStart=/path/to/binary/nas_checker_bot
WorkingDirectory=/path/to/working/directory
Restart=always
User=pi

[Install]
WantedBy=multi-user.target
```

The **Description** field is used to give the service a name.

The **ExecStart** field is used to specify the path to the binary you want to run as a service.

The **WorkingDirectory** field is used to specify the directory where the binary should be executed.

The **Restart** field is used to specify that the service should be restarted automatically in case of a failure.

The **User** field is used to specify the user account that should run the service. In this case, the pi user.

The **WantedBy** field is used to specify when the service should be started. In this case, when the system is in multi-user mode.

Save the file and exit the editor.

- Run the following command to reload the systemd configuration: `sudo systemctl daemon-reload`
- Run the following command to enable the service to start at boot time: `sudo systemctl enable nas_checker_bot.service`
- Run the following command to start the service: `sudo systemctl start nas_checker_bot.service`
- Run the following command to check the status of the service: `sudo systemctl status nas_checker_bot.service` This command should display the current status of the service.

That's it! Your bot is now running as a service on Raspberry Pi.
